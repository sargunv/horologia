import BackgroundTasks
import Foundation
import HorologiaShared
import UserNotifications
import WidgetKit

@MainActor
final class AppleWidgetBridge {
    static let shared = AppleWidgetBridge()

    static let appGroupIdentifier = "group.dev.horologia.mobile"
    static let refreshTaskIdentifier = "dev.horologia.mobile.refresh"
    static let snapshotKey = "taskWidgetSnapshotV1"
    static let errorKey = "taskWidgetRefreshError"

    nonisolated static let notificationDeepLinkKey = "deepLink"
    nonisolated static let notificationURL = Notification.Name("HorologiaNotificationDeepLink")

    private let defaults = UserDefaults(suiteName: appGroupIdentifier)
    private let notificationCenter = UNUserNotificationCenter.current()
    private let notificationDelegate = HorologiaNotificationDelegate()
    private var core: MobileAppCore?
    private var observation: KotlinAutoCloseable?
    private var currentState: MobileAppState?
    private var registeredBackgroundRefresh = false

    private init() {}

    func configure(core: MobileAppCore) {
        guard self.core == nil else { return }
        self.core = core
        notificationCenter.delegate = notificationDelegate
        registeredBackgroundRefresh = BGTaskScheduler.shared.register(
            forTaskWithIdentifier: Self.refreshTaskIdentifier,
            using: nil
        ) { [weak self] task in
            guard let refreshTask = task as? BGAppRefreshTask else {
                task.setTaskCompleted(success: false)
                return
            }
            Task { @MainActor [weak self] in
                guard let self else {
                    refreshTask.setTaskCompleted(success: false)
                    return
                }
                self.performBackgroundRefresh(refreshTask)
            }
        }
        observation = core.observe { [weak self] state in
            Task { @MainActor [weak self] in
                self?.consume(state)
            }
        }
    }

    /// The Account screen can call: `Task { _ = await AppleWidgetBridge.shared.requestAuthorization() }`.
    @discardableResult
    func requestAuthorization() async -> Bool {
        do {
            let granted = try await notificationCenter.requestAuthorization(options: [.alert, .badge, .sound])
            if granted, let currentState {
                await updateDueTaskNotifications(for: currentState)
            }
            return granted
        } catch {
            return false
        }
    }

    private func consume(_ state: MobileAppState) {
        let previous = currentState
        currentState = state

        guard state.phase == MobileSessionPhase.signedIn, let accountId = state.accountId else {
            if previous?.phase == MobileSessionPhase.signedIn {
                clearPublishedState()
                if let previous, let previousAccountId = previous.accountId {
                    removeNotifications(serverId: previous.server.serverId, accountId: previousAccountId)
                }
            }
            return
        }

        publish(state, accountId: accountId)
        scheduleBackgroundRefresh()
        Task { await updateDueTaskNotifications(for: state) }
    }

    private func publish(_ state: MobileAppState, accountId: String) {
        guard let defaults else { return }
        let generatedAt = Date(
            timeIntervalSince1970: state.myTasksGeneratedAtEpochSeconds?.doubleValue ?? Date().timeIntervalSince1970
        )
        let scope = SessionScope(
            serverId: state.server.serverId,
            baseUrl: state.server.baseUrl,
            accountId: accountId,
            accessToken: ""
        )
        let snapshot = TaskWidgetSnapshotKt.projectTaskWidgetSnapshotV1(
            scope: scope,
            tasks: state.myTasks,
            generatedAt: Self.iso8601.string(from: generatedAt),
            hasMore: state.myTasksCursor != nil,
            limit: 12
        )
        let encoded = TaskWidgetSnapshotJson.shared.encode(snapshot: snapshot)

        // A UserDefaults value is replaced as one property-list transaction; readers never see partial JSON.
        defaults.set(encoded, forKey: Self.snapshotKey)
        if let message = state.error?.message, !message.isEmpty {
            defaults.set(message, forKey: Self.errorKey)
        } else {
            defaults.removeObject(forKey: Self.errorKey)
        }
        WidgetCenter.shared.reloadAllTimelines()
    }

    private func clearPublishedState() {
        defaults?.removeObject(forKey: Self.snapshotKey)
        defaults?.removeObject(forKey: Self.errorKey)
        WidgetCenter.shared.reloadAllTimelines()
        BGTaskScheduler.shared.cancel(taskRequestWithIdentifier: Self.refreshTaskIdentifier)
    }

    private func scheduleBackgroundRefresh() {
        guard registeredBackgroundRefresh else { return }
        BGTaskScheduler.shared.cancel(taskRequestWithIdentifier: Self.refreshTaskIdentifier)
        let request = BGAppRefreshTaskRequest(identifier: Self.refreshTaskIdentifier)
        request.earliestBeginDate = Date(timeIntervalSinceNow: 15 * 60)
        try? BGTaskScheduler.shared.submit(request)
    }

    private func performBackgroundRefresh(_ refreshTask: BGAppRefreshTask) {
        guard let core else {
            refreshTask.setTaskCompleted(success: false)
            return
        }

        let work = Task { @MainActor [weak self] in
            var succeeded = false
            defer {
                refreshTask.setTaskCompleted(success: succeeded)
                self?.scheduleBackgroundRefresh()
            }
            do {
                if self?.currentState?.phase != MobileSessionPhase.signedIn {
                    try await core.start()
                }
                try Task.checkCancellation()
                try await core.refreshMyTasks()
                try Task.checkCancellation()
                if let state = self?.currentState,
                   state.phase == MobileSessionPhase.signedIn,
                   let accountId = state.accountId {
                    self?.publish(state, accountId: accountId)
                    await self?.updateDueTaskNotifications(for: state)
                    succeeded = state.error == nil
                }
            } catch {
                succeeded = false
            }
        }
        refreshTask.expirationHandler = { work.cancel() }
    }

    private func updateDueTaskNotifications(for state: MobileAppState) async {
        guard state.phase == MobileSessionPhase.signedIn, let accountId = state.accountId else { return }
        let settings = await notificationCenter.notificationSettings()
        guard settings.authorizationStatus == .authorized || settings.authorizationStatus == .provisional else {
            return
        }

        let serverId = state.server.serverId
        let prefix = notificationPrefix(serverId: serverId, accountId: accountId)
        let now = Date()
        var desired = Set<String>()

        for task in state.myTasks {
            guard !Self.completedStatuses.contains(task.status.lowercased()),
                  let due = task.dueText.flatMap(Self.dueDate),
                  due > now else { continue }

            let identifier = prefix + task.id
            desired.insert(identifier)
            let content = UNMutableNotificationContent()
            content.title = "Task due"
            content.body = task.title
            content.sound = .default
            content.userInfo = [
                Self.notificationDeepLinkKey: Self.taskDeepLink(
                    serverId: serverId,
                    spaceSlug: task.spaceSlug,
                    taskId: task.id
                )
            ]
            let components = Calendar.current.dateComponents(
                [.year, .month, .day, .hour, .minute, .second],
                from: due
            )
            let request = UNNotificationRequest(
                identifier: identifier,
                content: content,
                trigger: UNCalendarNotificationTrigger(dateMatching: components, repeats: false)
            )
            try? await notificationCenter.add(request)
        }

        let pending = await notificationCenter.pendingNotificationRequests()
        let stale = pending.map(\.identifier).filter { $0.hasPrefix(prefix) && !desired.contains($0) }
        notificationCenter.removePendingNotificationRequests(withIdentifiers: stale)
    }

    private func removeNotifications(serverId: String, accountId: String) {
        let prefix = notificationPrefix(serverId: serverId, accountId: accountId)
        Task {
            let pending = await notificationCenter.pendingNotificationRequests()
            let identifiers = pending.map(\.identifier).filter { $0.hasPrefix(prefix) }
            notificationCenter.removePendingNotificationRequests(withIdentifiers: identifiers)
        }
    }

    private func notificationPrefix(serverId: String, accountId: String) -> String {
        "horologia.due.\(serverId.utf8.count):\(serverId).\(accountId.utf8.count):\(accountId)."
    }

    private static let completedStatuses: Set<String> = ["done", "completed", "cancelled", "canceled"]

    private static let iso8601: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    private static let iso8601WithoutFractionalSeconds = ISO8601DateFormatter()

    private static let dateOnly: DateFormatter = {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = .current
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter
    }()

    private static func dueDate(_ value: String) -> Date? {
        if value.count == 10, let day = dateOnly.date(from: value) {
            return Calendar.current.date(bySettingHour: 9, minute: 0, second: 0, of: day)
        }
        return iso8601.date(from: value) ?? iso8601WithoutFractionalSeconds.date(from: value)
    }

    private static func taskDeepLink(serverId: String, spaceSlug: String, taskId: String) -> String {
        var components = URLComponents()
        components.scheme = "horologia"
        components.host = "tasks"
        components.path = "/\(spaceSlug)/\(taskId)"
        components.queryItems = [URLQueryItem(name: "server", value: serverId)]
        return components.string ?? "horologia://tasks"
    }
}

private final class HorologiaNotificationDelegate: NSObject, UNUserNotificationCenterDelegate {
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        completionHandler([.banner, .sound])
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        defer { completionHandler() }
        guard let value = response.notification.request.content.userInfo[AppleWidgetBridge.notificationDeepLinkKey] as? String,
              let url = URL(string: value) else { return }
        NotificationCenter.default.post(name: AppleWidgetBridge.notificationURL, object: url)
    }
}
