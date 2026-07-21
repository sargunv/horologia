import Foundation
import HorologiaShared
import SwiftUI

@main
@MainActor
struct HorologiaApp: App {
    @StateObject private var adapter: MobileCoreAdapter
    @StateObject private var router = AppRouter()
    init() {
        let core = IosAppCoreFactory().create()
        _adapter = StateObject(wrappedValue: MobileCoreAdapter(core: core))
        AppleWidgetBridge.shared.configure(core: core)
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(adapter)
                .environmentObject(router)
                .onOpenURL { url in
                    adapter.handleDeepLink(url, router: router)
                }
                .onReceive(
                    NotificationCenter.default.publisher(for: AppleWidgetBridge.notificationURL)
                ) { notification in
                    guard let url = notification.object as? URL else { return }
                    adapter.handleDeepLink(url, router: router)
                }
        }
    }
}
