import SwiftUI
import WidgetKit

private enum WidgetStore {
    static let appGroupIdentifier = "group.dev.horologia.mobile"
    static let snapshotKey = "taskWidgetSnapshotV1"
    static let errorKey = "taskWidgetRefreshError"

    static func load() -> WidgetContent {
        guard let defaults = UserDefaults(suiteName: appGroupIdentifier) else {
            return WidgetContent(snapshot: nil, error: "Widget storage is unavailable.")
        }
        let refreshError = defaults.string(forKey: errorKey)
        guard let encoded = defaults.string(forKey: snapshotKey) else {
            return WidgetContent(snapshot: nil, error: refreshError)
        }
        guard let data = encoded.data(using: .utf8) else {
            return WidgetContent(snapshot: nil, error: "Saved task data is unreadable.")
        }
        do {
            let snapshot = try JSONDecoder().decode(TaskWidgetSnapshotV1.self, from: data)
            guard snapshot.version == 1 else {
                return WidgetContent(snapshot: nil, error: "Update Horologia to refresh this widget.")
            }
            return WidgetContent(snapshot: snapshot, error: refreshError)
        } catch {
            return WidgetContent(snapshot: nil, error: "Open Horologia to refresh task data.")
        }
    }
}

private struct TaskWidgetRowV1: Codable, Identifiable {
    let id: String
    let spaceSlug: String
    let title: String
    let due: String?
    let status: String
}

private struct TaskWidgetSnapshotV1: Codable {
    let version: Int
    let serverId: String
    let accountId: String
    let generatedAt: String
    let taskCount: Int
    let hasMore: Bool
    let rows: [TaskWidgetRowV1]
}

private struct WidgetContent {
    let snapshot: TaskWidgetSnapshotV1?
    let error: String?
}

private struct HorologiaEntry: TimelineEntry {
    let date: Date
    let content: WidgetContent
}

private struct HorologiaTimelineProvider: TimelineProvider {
    func placeholder(in context: Context) -> HorologiaEntry {
        HorologiaEntry(date: Date(), content: WidgetContent(snapshot: nil, error: nil))
    }

    func getSnapshot(in context: Context, completion: @escaping (HorologiaEntry) -> Void) {
        completion(HorologiaEntry(date: Date(), content: WidgetStore.load()))
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<HorologiaEntry>) -> Void) {
        let entry = HorologiaEntry(date: Date(), content: WidgetStore.load())
        completion(Timeline(entries: [entry], policy: .after(Date(timeIntervalSinceNow: 15 * 60))))
    }
}

private struct HorologiaWidgetView: View {
    @Environment(\.widgetFamily) private var family
    let entry: HorologiaEntry

    private var rowLimit: Int {
        switch family {
        case .systemLarge: return 8
        case .systemMedium: return 3
        default: return 2
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 7) {
            HStack {
                Text("My tasks")
                    .font(.headline)
                Spacer()
                if let snapshot = entry.content.snapshot {
                    Text("\(snapshot.taskCount)")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                }
            }

            if let snapshot = entry.content.snapshot {
                if snapshot.rows.isEmpty, let error = entry.content.error {
                    Spacer()
                    Label(error, systemImage: "exclamationmark.triangle.fill")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                    Spacer()
                } else if snapshot.rows.isEmpty {
                    Spacer()
                    Text("No tasks assigned to you.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                    Spacer()
                } else {
                    if let error = entry.content.error {
                        Label(error, systemImage: "exclamationmark.triangle.fill")
                            .font(.caption2)
                            .foregroundStyle(.orange)
                            .lineLimit(1)
                    }
                    ForEach(snapshot.rows.prefix(rowLimit)) { task in
                        Link(destination: taskURL(task, serverId: snapshot.serverId)) {
                            TaskRow(task: task)
                        }
                    }
                    if snapshot.hasMore || snapshot.rows.count > rowLimit {
                        Text("More tasks in Horologia")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            } else {
                Spacer()
                if let error = entry.content.error {
                    Label(error, systemImage: "exclamationmark.triangle.fill")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                } else {
                    Text("Open Horologia to load your tasks.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
                Spacer()
            }
        }
        .padding()
        .widgetURL(tasksURL(serverId: entry.content.snapshot?.serverId))
    }

    private func taskURL(_ task: TaskWidgetRowV1, serverId: String) -> URL {
        scopedURL(host: "tasks", path: "/\(task.spaceSlug)/\(task.id)", serverId: serverId)
    }

    private func tasksURL(serverId: String?) -> URL {
        scopedURL(host: "tasks", path: "", serverId: serverId)
    }

    private func scopedURL(host: String, path: String, serverId: String?) -> URL {
        var components = URLComponents()
        components.scheme = "horologia"
        components.host = host
        components.path = path
        if let serverId {
            components.queryItems = [URLQueryItem(name: "server", value: serverId)]
        }
        return components.url ?? URL(string: "horologia://tasks")!
    }
}

private struct TaskRow: View {
    let task: TaskWidgetRowV1

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 7) {
            Image(systemName: "circle")
                .font(.caption)
                .foregroundStyle(.tint)
            VStack(alignment: .leading, spacing: 1) {
                Text(task.title)
                    .font(.subheadline.weight(.medium))
                    .lineLimit(1)
                HStack(spacing: 5) {
                    Text(task.spaceSlug)
                    if let due = task.due, !due.isEmpty {
                        Text("·")
                        Text(due)
                    }
                }
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineLimit(1)
            }
        }
    }
}

private struct HorologiaTasksWidget: Widget {
    let kind = "HorologiaTasksWidget"

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: HorologiaTimelineProvider()) { entry in
            HorologiaWidgetView(entry: entry)
        }
        .configurationDisplayName("My tasks")
        .description("See your assigned Horologia tasks and due dates.")
        .supportedFamilies([.systemSmall, .systemMedium, .systemLarge])
    }
}

@main
struct HorologiaWidgetsBundle: WidgetBundle {
    var body: some Widget {
        HorologiaTasksWidget()
    }
}
