import HorologiaShared
import SwiftUI

// MARK: - Core adapter

/// Main-actor observable bridge over the shared `MobileAppCore`.
///
/// The core emits state from a Kotlin coroutine dispatcher, so emissions are
/// re-hopped to the main actor before publication. Every suspending core
/// command is invoked from a Swift `Task`; the main actor is never blocked.
@MainActor
final class MobileCoreAdapter: ObservableObject {
    @Published private(set) var state: MobileAppState?

    let core: MobileAppCore
    private var observation: KotlinAutoCloseable?
    private var didBoot = false

    init(core: MobileAppCore) {
        self.core = core
    }

    /// Subscribes to core state (first emission is immediate) and starts the
    /// default server profile. Idempotent — safe to call from `.task`.
    func boot() {
        guard !didBoot else { return }
        didBoot = true
        observation = core.observe { [weak self] newState in
            Task { @MainActor [weak self] in
                self?.state = newState
            }
        }
        Task { try? await core.start() }
    }

    // MARK: Session

    func connect(baseUrl: String) {
        let trimmed = baseUrl.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        Task { try? await core.start(serverId: "default", baseUrl: trimmed) }
    }

    func authorize() {
        Task { try? await core.authorize() }
    }

    func retry() {
        Task { try? await core.retry() }
    }

    func signOut() {
        core.clearSelection()
        Task { try? await core.signOut() }
    }

    // MARK: Tasks

    func refreshMyTasks() async {
        try? await core.refreshMyTasks()
    }

    func loadMoreMyTasks() {
        Task { try? await core.loadMoreMyTasks() }
    }

    /// Fetches the task only when the core's selected task no longer matches.
    func ensureTaskSelected(spaceSlug: String, taskId: String) async {
        if let selected = state?.selectedTask,
           selected.id == taskId, selected.spaceSlug == spaceSlug { return }
        if state?.loading.task == true { return }
        try? await core.selectTask(spaceSlug: spaceSlug, taskId: taskId)
    }

    func updateTask(spaceSlug: String, taskId: String, update: MobileTaskUpdate) async {
        try? await core.updateTask(spaceSlug: spaceSlug, taskId: taskId, update: update)
    }

    // MARK: Spaces & recipes

    func loadSpaces() async {
        try? await core.loadSpaces()
    }

    func reloadSpace(_ slug: String) async {
        try? await core.selectSpace(spaceSlug: slug)
    }

    func ensureSpaceSelected(_ slug: String) async {
        if state?.selectedSpace?.slug == slug,
           state?.loading.space != true { return }
        if state?.loading.space == true { return }
        try? await core.selectSpace(spaceSlug: slug)
    }

    func loadMoreSpaceTasks() {
        Task { try? await core.loadMoreSpaceTasks() }
    }

    func loadMoreSpaceRecipes() {
        Task { try? await core.loadMoreSpaceRecipes() }
    }

    func ensureRecipeSelected(spaceSlug: String, recipeId: String) async {
        if let selected = state?.selectedRecipe,
           selected.id == recipeId, selected.spaceSlug == spaceSlug { return }
        if state?.loading.recipe == true { return }
        try? await core.selectRecipe(spaceSlug: spaceSlug, recipeId: recipeId)
    }

    func updateRecipe(spaceSlug: String, recipeId: String, update: MobileRecipeUpdate) async {
        try? await core.updateRecipe(spaceSlug: spaceSlug, recipeId: recipeId, update: update)
    }

    // MARK: Search

    func search(_ query: String) {
        Task { try? await core.search(query: query) }
    }

    // MARK: Account

    func updateProfile(name: String?, email: String?) async {
        try? await core.updateProfile(update: MobileProfileUpdate(name: name, email: email))
    }

    // MARK: Deep links

    /// Parses an incoming URL through the shared deep-link parser and routes
    /// it into navigation state. Only honored while signed in, since every
    /// destination is scoped to the active server/account.
    func handleDeepLink(_ url: URL, router: AppRouter) {
        guard let state, state.phase == MobileSessionPhase.signedIn else { return }
        guard let destination = HorologiaDeepLinks.shared.parse(
            link: url.absoluteString,
            expectedServerId: state.server.serverId,
            expectedBaseUrl: state.server.baseUrl
        ) else { return }
        router.route(to: destination)
    }
}

// MARK: - Selection identifiers

/// Stable, value-semantic selection keys. Lists and navigation destinations
/// select by id, never by index or object identity, so selection survives
/// list reloads, resizes, and rotation.
struct TaskRef: Hashable {
    let spaceSlug: String
    let id: String

    init(spaceSlug: String, id: String) {
        self.spaceSlug = spaceSlug
        self.id = id
    }

    init(_ task: MobileTask) {
        self.init(spaceSlug: task.spaceSlug, id: task.id)
    }
}

struct RecipeRef: Hashable {
    let spaceSlug: String
    let id: String

    init(spaceSlug: String, id: String) {
        self.spaceSlug = spaceSlug
        self.id = id
    }

    init(_ recipe: MobileRecipe) {
        self.init(spaceSlug: recipe.spaceSlug, id: recipe.id)
    }
}

enum SpaceItemSelection: Hashable {
    case task(TaskRef)
    case recipe(RecipeRef)
}

// MARK: - Root

struct ContentView: View {
    @EnvironmentObject private var adapter: MobileCoreAdapter

    var body: some View {
        content
            .task { adapter.boot() }
    }

    @ViewBuilder
    private var content: some View {
        if let state = adapter.state, state.phase == MobileSessionPhase.signedIn {
            SignedInRootView()
        } else if adapter.state != nil {
            ServerConnectView()
        } else {
            ProgressView("Starting…")
                .controlSize(.large)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
    }
}

// MARK: - Signed-out / connect flow

private struct ServerConnectView: View {
    @EnvironmentObject private var adapter: MobileCoreAdapter
    @State private var baseUrl = ""
    @State private var didEditUrl = false

    private var state: MobileAppState? { adapter.state }

    private var isBootstrapping: Bool {
        guard let state else { return true }
        return state.phase == MobileSessionPhase.bootstrap || state.loading.bootstrap
    }

    private var isAuthorizing: Bool {
        state?.phase == MobileSessionPhase.authorizing
    }

    private var normalizedUrl: String {
        baseUrl.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var urlChanged: Bool {
        normalizedUrl != (state?.server.baseUrl ?? "")
    }

    private var urlIsValid: Bool {
        guard let url = URL(string: normalizedUrl),
              let scheme = url.scheme?.lowercased(),
              scheme == "http" || scheme == "https",
              url.host != nil else { return false }
        return true
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("https://server.example.com", text: $baseUrl)
                        .keyboardType(.URL)
                        .textContentType(.URL)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                        .disabled(isBootstrapping || isAuthorizing)
                        .onChange(of: baseUrl) { _ in didEditUrl = true }
                } header: {
                    Text("Server")
                } footer: {
                    Text("The Horologia server this device syncs with.")
                }

                if let error = state?.error {
                    Section {
                        ErrorBanner(error: error) { adapter.retry() }
                            .listRowInsets(EdgeInsets())
                            .listRowBackground(Color.clear)
                    }
                }

                Section {
                    if isBootstrapping {
                        LoadingRow(label: "Contacting server…")
                    } else if isAuthorizing {
                        LoadingRow(label: "Waiting for sign-in…")
                    } else if urlChanged || state?.authConfig == nil {
                        Button("Connect") { adapter.connect(baseUrl: normalizedUrl) }
                            .disabled(!urlIsValid)
                    } else if let config = state?.authConfig, config.oidcEnabled {
                        Button("Sign In with \(config.oidcLabel)") { adapter.authorize() }
                    } else {
                        Text("This server does not offer single sign-on.")
                            .foregroundStyle(.secondary)
                    }
                }

                if let config = state?.authConfig, !urlChanged, !isBootstrapping {
                    Section("Server Capabilities") {
                        LabeledContent(
                            "Single Sign-On",
                            value: config.oidcEnabled ? config.oidcLabel : "Unavailable"
                        )
                        LabeledContent(
                            "Password Sign-In",
                            value: config.passwordEnabled ? "Available" : "Unavailable"
                        )
                    }
                }
            }
            .navigationTitle("Horologia")
            .onAppear(perform: syncUrl)
            .onChange(of: adapter.state?.server.baseUrl) { _ in syncUrl() }
        }
    }

    private func syncUrl() {
        guard !didEditUrl else { return }
        baseUrl = adapter.state?.server.baseUrl ?? ""
    }
}

// MARK: - Signed-in shell

enum AppTab: Hashable {
    case tasks, recipes, spaces, search, account
}

/// App-level navigation state. Selections are stable value types keyed by id,
/// so they survive size-class changes, rotation, and list reloads, and can be
/// driven externally (e.g. deep links). Compact-width tab stacks mirror these
/// properties as their `NavigationStack` path. Replacing a parent selection
/// (`spaceSlug`, `recipeSpace`) clears its now-stale child selection
/// synchronously, so path writes never race a separate `onChange` pass.
@MainActor
final class AppRouter: ObservableObject {
    @Published var tab: AppTab = .tasks
    @Published var taskRef: TaskRef?
    @Published var recipeSpace: String? {
        didSet { if recipeSpace != oldValue { recipeRef = nil } }
    }
    @Published var recipeRef: RecipeRef?
    @Published var spaceSlug: String? {
        didSet { if spaceSlug != oldValue { spaceItem = nil } }
    }
    @Published var spaceItem: SpaceItemSelection?
    @Published var searchQuery = ""
    @Published var searchItem: SpaceItemSelection?

    /// Maps a parsed semantic destination onto tab + selection state.
    /// OAuth callbacks are deliberately ignored here: they belong to the
    /// authorization flow (ASWebAuthenticationSession), not to navigation.
    func route(to destination: SemanticDestination) {
        switch destination {
        case let task as SemanticDestinationTask:
            tab = .tasks
            taskRef = TaskRef(spaceSlug: task.spaceSlug, id: task.taskId)
        case let recipe as SemanticDestinationRecipe:
            tab = .recipes
            recipeSpace = recipe.spaceSlug
            recipeRef = RecipeRef(spaceSlug: recipe.spaceSlug, id: recipe.recipeId)
        case let space as SemanticDestinationSpace:
            tab = .spaces
            spaceSlug = space.spaceSlug
            spaceItem = nil
        case let search as SemanticDestinationSearch:
            tab = .search
            searchItem = nil
            searchQuery = search.query ?? ""
        case is SemanticDestinationTasks:
            tab = .tasks
            taskRef = nil
        case is SemanticDestinationRecipes:
            tab = .recipes
            recipeSpace = nil
            recipeRef = nil
        case is SemanticDestinationSpaces:
            tab = .spaces
            spaceSlug = nil
            spaceItem = nil
        case is SemanticDestinationAccount:
            tab = .account
        default:
            break
        }
    }
}

private struct SignedInRootView: View {
    @EnvironmentObject private var router: AppRouter

    var body: some View {
        TabView(selection: $router.tab) {
            TasksTab()
                .tabItem { Label("Tasks", systemImage: "checklist") }
                .tag(AppTab.tasks)
            RecipesTab()
                .tabItem { Label("Recipes", systemImage: "fork.knife") }
                .tag(AppTab.recipes)
            SpacesTab()
                .tabItem { Label("Spaces", systemImage: "square.grid.2x2") }
                .tag(AppTab.spaces)
            SearchTab()
                .tabItem { Label("Search", systemImage: "magnifyingglass") }
                .tag(AppTab.search)
            AccountTab()
                .tabItem { Label("Account", systemImage: "person.crop.circle") }
                .tag(AppTab.account)
        }
    }
}

// MARK: - Tasks tab

private struct TasksTab: View {
    @EnvironmentObject private var adapter: MobileCoreAdapter
    @EnvironmentObject private var router: AppRouter
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    private var selection: Binding<TaskRef?> { $router.taskRef }

    /// Compact-width stack: at most one pushed task detail, mirrored from
    /// `router.taskRef` so taps, deep links, and back navigation agree.
    private var taskPath: Binding<[TaskRef]> {
        Binding(
            get: { router.taskRef.map { [$0] } ?? [] },
            set: { router.taskRef = $0.last }
        )
    }

    var body: some View {
        if horizontalSizeClass == .regular {
            NavigationSplitView {
                List(selection: selection) {
                    TaskListContent { task in
                        TaskRow(task: task).tag(TaskRef(task))
                    }
                }
                .navigationTitle("Tasks")
                .refreshable { await adapter.refreshMyTasks() }
            } detail: {
                if let ref = selection.wrappedValue {
                    TaskDetailView(ref: ref)
                } else {
                    EmptyStateView(
                        icon: "checklist",
                        title: "Select a Task",
                        message: "Choose a task from the list to see its details."
                    )
                }
            }
        } else {
            NavigationStack(path: taskPath) {
                List {
                    TaskListContent { task in
                        NavigationLink(value: TaskRef(task)) {
                            TaskRow(task: task)
                        }
                    }
                }
                .navigationTitle("Tasks")
                .navigationDestination(for: TaskRef.self) { TaskDetailView(ref: $0) }
                .refreshable { await adapter.refreshMyTasks() }
            }
        }
    }
}

private struct TaskListContent<Row: View>: View {
    @EnvironmentObject private var adapter: MobileCoreAdapter
    @ViewBuilder let row: (MobileTask) -> Row

    var body: some View {
        if let error = adapter.state?.error {
            Section {
                ErrorBanner(error: error) {
                    Task { await adapter.refreshMyTasks() }
                }
                .listRowInsets(EdgeInsets())
                .listRowBackground(Color.clear)
            }
        }

        if adapter.state == nil || (adapter.state?.loading.myTasks == true && tasks.isEmpty) {
            Section { LoadingRow(label: "Loading tasks…") }
        } else if tasks.isEmpty {
            Section {
                EmptyStateView(
                    icon: "checklist",
                    title: "No Tasks",
                    message: "Tasks assigned to you will appear here. Pull down to refresh."
                )
                .listRowSeparator(.hidden)
                .listRowBackground(Color.clear)
            }
        } else {
            Section {
                ForEach(tasks, id: \.id) { task in
                    row(task)
                }
                if adapter.state?.myTasksCursor != nil {
                    PaginationRow(isLoading: adapter.state?.loading.moreMyTasks == true) {
                        adapter.loadMoreMyTasks()
                    }
                }
            } footer: {
                SnapshotNote(
                    fromCache: adapter.state?.myTasksFromCache == true,
                    generatedAt: epochDate(adapter.state?.myTasksGeneratedAtEpochSeconds ?? nil)
                )
            }
        }
    }

    private var tasks: [MobileTask] {
        adapter.state?.myTasks ?? []
    }
}

// MARK: - Spaces tab

/// Compact-width Spaces stack levels: a space overview, then a task or
/// recipe pushed on top of it.
private enum SpacesRoute: Hashable {
    case overview(String)
    case item(SpaceItemSelection)
}

private struct SpacesTab: View {
    @EnvironmentObject private var router: AppRouter
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    private var spaceSlug: Binding<String?> { $router.spaceSlug }
    private var item: Binding<SpaceItemSelection?> { $router.spaceItem }

    /// Compact-width stack: `[.overview(slug)]` plus an optional `.item`,
    /// mirrored from `router.spaceSlug`/`router.spaceItem`. Writes assign the
    /// parent first; `AppRouter` clears a stale child when the parent changes.
    private var spacesPath: Binding<[SpacesRoute]> {
        Binding(
            get: {
                guard let slug = router.spaceSlug else { return [] }
                var path: [SpacesRoute] = [.overview(slug)]
                if let selection = router.spaceItem { path.append(.item(selection)) }
                return path
            },
            set: { path in
                var slug: String?
                var selection: SpaceItemSelection?
                for route in path {
                    switch route {
                    case .overview(let value): slug = value
                    case .item(let value): selection = value
                    }
                }
                router.spaceSlug = slug
                router.spaceItem = slug == nil ? nil : selection
            }
        )
    }

    var body: some View {
        Group {
            if horizontalSizeClass == .regular {
                NavigationSplitView {
                    List(selection: spaceSlug) {
                        SpaceListContent { space in
                            SpaceRow(space: space).tag(space.slug)
                        }
                    }
                    .navigationTitle("Spaces")
                } content: {
                    if let slug = spaceSlug.wrappedValue {
                        SpaceOverviewView(slug: slug, itemSelection: item)
                    } else {
                        EmptyStateView(
                            icon: "square.grid.2x2",
                            title: "Select a Space",
                            message: "Choose a space to see its tasks and recipes."
                        )
                    }
                } detail: {
                    if let selection = item.wrappedValue {
                        SpaceItemDetailView(item: selection)
                    } else {
                        EmptyStateView(
                            icon: "doc.text.magnifyingglass",
                            title: "Nothing Selected",
                            message: "Choose a task or recipe to see its details."
                        )
                    }
                }
            } else {
                NavigationStack(path: spacesPath) {
                    List {
                        SpaceListContent { space in
                            NavigationLink(value: SpacesRoute.overview(space.slug)) {
                                SpaceRow(space: space)
                            }
                        }
                    }
                    .navigationTitle("Spaces")
                    .navigationDestination(for: SpacesRoute.self) { route in
                        switch route {
                        case .overview(let slug):
                            SpaceOverviewView(slug: slug, itemSelection: item)
                        case .item(let selection):
                            SpaceItemDetailView(item: selection)
                        }
                    }
                }
            }
        }
    }
}

// MARK: - Recipes tab

/// Compact-width Recipes stack levels: a space's recipe list, then a recipe
/// pushed on top of it.
private enum RecipesRoute: Hashable {
    case space(String)
    case recipe(RecipeRef)
}

private struct RecipesTab: View {
    @EnvironmentObject private var router: AppRouter
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    private var spaceSlug: Binding<String?> { $router.recipeSpace }
    private var recipe: Binding<RecipeRef?> { $router.recipeRef }

    /// Compact-width stack: `[.space(slug)]` plus an optional `.recipe`,
    /// mirrored from `router.recipeSpace`/`router.recipeRef`. Writes assign
    /// the parent first; `AppRouter` clears a stale child when it changes.
    private var recipesPath: Binding<[RecipesRoute]> {
        Binding(
            get: {
                guard let slug = router.recipeSpace else { return [] }
                var path: [RecipesRoute] = [.space(slug)]
                if let ref = router.recipeRef { path.append(.recipe(ref)) }
                return path
            },
            set: { path in
                var slug: String?
                var ref: RecipeRef?
                for route in path {
                    switch route {
                    case .space(let value): slug = value
                    case .recipe(let value): ref = value
                    }
                }
                router.recipeSpace = slug
                router.recipeRef = slug == nil ? nil : ref
            }
        )
    }

    var body: some View {
        Group {
            if horizontalSizeClass == .regular {
                NavigationSplitView {
                    List(selection: spaceSlug) {
                        SpaceListContent { space in
                            SpaceRow(space: space).tag(space.slug)
                        }
                    }
                    .navigationTitle("Recipes")
                } content: {
                    if let slug = spaceSlug.wrappedValue {
                        SpaceRecipesView(slug: slug, recipeSelection: recipe)
                    } else {
                        EmptyStateView(
                            icon: "fork.knife",
                            title: "Select a Space",
                            message: "Choose a space to browse its recipes."
                        )
                    }
                } detail: {
                    if let ref = recipe.wrappedValue {
                        RecipeDetailView(ref: ref)
                    } else {
                        EmptyStateView(
                            icon: "fork.knife",
                            title: "Select a Recipe",
                            message: "Choose a recipe to see its details."
                        )
                    }
                }
            } else {
                NavigationStack(path: recipesPath) {
                    List {
                        SpaceListContent { space in
                            NavigationLink(value: RecipesRoute.space(space.slug)) {
                                SpaceRow(space: space)
                            }
                        }
                    }
                    .navigationTitle("Recipes")
                    .navigationDestination(for: RecipesRoute.self) { route in
                        switch route {
                        case .space(let slug):
                            SpaceRecipesView(slug: slug, recipeSelection: recipe)
                        case .recipe(let ref):
                            RecipeDetailView(ref: ref)
                        }
                    }
                }
            }
        }
    }
}

// MARK: - Space browsing

private struct SpaceListContent<Row: View>: View {
    @EnvironmentObject private var adapter: MobileCoreAdapter
    @ViewBuilder let row: (MobileSpace) -> Row

    var body: some View {
        if let error = adapter.state?.error {
            Section {
                ErrorBanner(error: error) {
                    Task { await adapter.loadSpaces() }
                }
                .listRowInsets(EdgeInsets())
                .listRowBackground(Color.clear)
            }
        }

        if adapter.state == nil || (adapter.state?.loading.spaces == true && spaces.isEmpty) {
            Section { LoadingRow(label: "Loading spaces…") }
        } else if spaces.isEmpty {
            Section {
                EmptyStateView(
                    icon: "square.grid.2x2",
                    title: "No Spaces",
                    message: "Spaces you join will appear here. Pull down to refresh."
                )
                .listRowSeparator(.hidden)
                .listRowBackground(Color.clear)
            }
        } else {
            Section {
                ForEach(spaces, id: \.slug) { space in
                    row(space)
                }
            } footer: {
                SnapshotNote(
                    fromCache: adapter.state?.spacesFromCache == true,
                    generatedAt: epochDate(adapter.state?.spacesGeneratedAtEpochSeconds ?? nil)
                )
            }
        }
    }

    private var spaces: [MobileSpace] {
        adapter.state?.spaces ?? []
    }
}

private struct SpaceRow: View {
    let space: MobileSpace

    var body: some View {
        Label(space.name, systemImage: "number")
            .accessibilityLabel("Space: \(space.name)")
    }
}

/// Tasks + recipes of one space. Renders tagged selection rows on regular
/// width and navigation links on compact width.
private struct SpaceOverviewView: View {
    let slug: String
    @Binding var itemSelection: SpaceItemSelection?

    @EnvironmentObject private var adapter: MobileCoreAdapter
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    var body: some View {
        // A selection binding makes a compact List treat taps as selection and
        // swallow NavigationLink pushes, so only the regular-width list gets one.
        Group {
            if horizontalSizeClass == .regular {
                List(selection: $itemSelection) { content }
            } else {
                List { content }
            }
        }
        .navigationTitle(adapter.state?.selectedSpace?.name ?? slug)
        .task(id: slug) { await adapter.ensureSpaceSelected(slug) }
        .onAppear { Task { await adapter.ensureSpaceSelected(slug) } }
        .refreshable { await adapter.reloadSpace(slug) }
    }

    @ViewBuilder
    private var content: some View {
        if let error = adapter.state?.error {
            Section {
                ErrorBanner(error: error) {
                    Task { await adapter.reloadSpace(slug) }
                }
                .listRowInsets(EdgeInsets())
                .listRowBackground(Color.clear)
            }
        }

        if adapter.state?.loading.space == true, tasks.isEmpty, recipes.isEmpty {
            Section { LoadingRow(label: "Loading space…") }
        } else {
            Section {
                if tasks.isEmpty {
                    Text("No tasks in this space.")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(tasks, id: \.id) { task in
                        taskRow(task)
                    }
                    if adapter.state?.spaceTasksCursor != nil {
                        PaginationRow(isLoading: adapter.state?.loading.moreSpaceTasks == true) {
                            adapter.loadMoreSpaceTasks()
                        }
                    }
                }
            } header: {
                Text("Tasks")
            }

            Section {
                if recipes.isEmpty {
                    Text("No recipes in this space.")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(recipes, id: \.id) { recipe in
                        recipeRow(recipe)
                    }
                    if adapter.state?.spaceRecipesCursor != nil {
                        PaginationRow(isLoading: adapter.state?.loading.moreSpaceRecipes == true) {
                            adapter.loadMoreSpaceRecipes()
                        }
                    }
                }
            } header: {
                Text("Recipes")
            }
        }
    }

    @ViewBuilder
    private func taskRow(_ task: MobileTask) -> some View {
        let item = SpaceItemSelection.task(TaskRef(task))
        if horizontalSizeClass == .regular {
            TaskRow(task: task).tag(item)
        } else {
            NavigationLink(value: SpacesRoute.item(item)) { TaskRow(task: task) }
        }
    }

    @ViewBuilder
    private func recipeRow(_ recipe: MobileRecipe) -> some View {
        let item = SpaceItemSelection.recipe(RecipeRef(recipe))
        if horizontalSizeClass == .regular {
            RecipeRow(recipe: recipe).tag(item)
        } else {
            NavigationLink(value: SpacesRoute.item(item)) { RecipeRow(recipe: recipe) }
        }
    }

    private var tasks: [MobileTask] { adapter.state?.spaceTasks ?? [] }
    private var recipes: [MobileRecipe] { adapter.state?.spaceRecipes ?? [] }
}

/// Recipes-only view of one space, used by the Recipes tab.
private struct SpaceRecipesView: View {
    let slug: String
    @Binding var recipeSelection: RecipeRef?

    @EnvironmentObject private var adapter: MobileCoreAdapter
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    var body: some View {
        // See SpaceOverviewView: a selection binding on compact intercepts
        // NavigationLink taps, so only regular width keeps one.
        Group {
            if horizontalSizeClass == .regular {
                List(selection: $recipeSelection) { content }
            } else {
                List { content }
            }
        }
        .navigationTitle(adapter.state?.selectedSpace?.name ?? "Recipes")
        .task(id: slug) { await adapter.ensureSpaceSelected(slug) }
        .onAppear { Task { await adapter.ensureSpaceSelected(slug) } }
        .refreshable { await adapter.reloadSpace(slug) }
    }

    @ViewBuilder
    private var content: some View {
        if let error = adapter.state?.error {
            Section {
                ErrorBanner(error: error) {
                    Task { await adapter.reloadSpace(slug) }
                }
                .listRowInsets(EdgeInsets())
                .listRowBackground(Color.clear)
            }
        }

        if adapter.state?.loading.space == true, recipes.isEmpty {
            Section { LoadingRow(label: "Loading recipes…") }
        } else if recipes.isEmpty {
            Section {
                EmptyStateView(
                    icon: "fork.knife",
                    title: "No Recipes",
                    message: "This space has no recipes yet."
                )
                .listRowSeparator(.hidden)
                .listRowBackground(Color.clear)
            }
        } else {
            Section {
                ForEach(recipes, id: \.id) { recipe in
                    row(recipe)
                }
                if adapter.state?.spaceRecipesCursor != nil {
                    PaginationRow(isLoading: adapter.state?.loading.moreSpaceRecipes == true) {
                        adapter.loadMoreSpaceRecipes()
                    }
                }
            } header: {
                Text(adapter.state?.selectedSpace.map { "Recipes in \($0.name)" } ?? "Recipes")
            }
        }
    }

    @ViewBuilder
    private func row(_ recipe: MobileRecipe) -> some View {
        let ref = RecipeRef(recipe)
        if horizontalSizeClass == .regular {
            RecipeRow(recipe: recipe).tag(ref)
        } else {
            NavigationLink(value: RecipesRoute.recipe(ref)) { RecipeRow(recipe: recipe) }
        }
    }

    private var recipes: [MobileRecipe] { adapter.state?.spaceRecipes ?? [] }
}

private struct SpaceItemDetailView: View {
    let item: SpaceItemSelection

    var body: some View {
        switch item {
        case .task(let ref):
            TaskDetailView(ref: ref)
        case .recipe(let ref):
            RecipeDetailView(ref: ref)
        }
    }
}

// MARK: - Search tab

private struct SearchTab: View {
    @EnvironmentObject private var adapter: MobileCoreAdapter
    @EnvironmentObject private var router: AppRouter
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass

    private var query: Binding<String> { $router.searchQuery }
    private var item: Binding<SpaceItemSelection?> { $router.searchItem }

    /// Compact-width stack: at most one pushed search result, mirrored from
    /// `router.searchItem` so taps, deep links, and back navigation agree.
    private var searchPath: Binding<[SpaceItemSelection]> {
        Binding(
            get: { router.searchItem.map { [$0] } ?? [] },
            set: { router.searchItem = $0.last }
        )
    }

    var body: some View {
        Group {
            if horizontalSizeClass == .regular {
                NavigationSplitView {
                    searchList
                        .navigationTitle("Search")
                } detail: {
                    if let selection = item.wrappedValue {
                        SpaceItemDetailView(item: selection)
                    } else {
                        EmptyStateView(
                            icon: "magnifyingglass",
                            title: "Select a Result",
                            message: "Choose a search result to see its details."
                        )
                    }
                }
            } else {
                NavigationStack(path: searchPath) {
                    searchList
                        .navigationTitle("Search")
                        .navigationDestination(for: SpaceItemSelection.self) { selection in
                            SpaceItemDetailView(item: selection)
                        }
                }
            }
        }
        .task(id: router.searchQuery) {
            let trimmed = router.searchQuery.trimmingCharacters(in: .whitespacesAndNewlines)
            try? await Task.sleep(nanoseconds: 300_000_000)
            guard !Task.isCancelled else { return }
            adapter.search(trimmed)
        }
    }

    private var searchList: some View {
        // See SpaceOverviewView: a selection binding on compact intercepts
        // NavigationLink taps, so only regular width keeps one.
        Group {
            if horizontalSizeClass == .regular {
                List(selection: item) { searchContent }
            } else {
                List { searchContent }
            }
        }
        .searchable(text: query, prompt: "Tasks and recipes")
        .toolbar {
            if !router.searchQuery.isEmpty {
                ToolbarItem(placement: .primaryAction) {
                    Button("Clear") { router.searchQuery = "" }
                        .accessibilityHint("Clears the search field and results")
                }
            }
        }
    }

    @ViewBuilder
    private var searchContent: some View {
        if let error = adapter.state?.error {
            Section {
                ErrorBanner(error: error) { adapter.search(query.wrappedValue) }
                    .listRowInsets(EdgeInsets())
                    .listRowBackground(Color.clear)
            }
        }

        if router.searchQuery.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            Section {
                EmptyStateView(
                    icon: "magnifyingglass",
                    title: "Search Horologia",
                    message: "Find tasks and recipes across your spaces."
                )
                .listRowSeparator(.hidden)
                .listRowBackground(Color.clear)
            }
        } else if adapter.state?.loading.search == true, results.isEmpty {
            Section { LoadingRow(label: "Searching…") }
        } else if results.isEmpty {
            Section {
                EmptyStateView(
                    icon: "magnifyingglass",
                    title: "No Results",
                    message: "Nothing matches “\(router.searchQuery.trimmingCharacters(in: .whitespacesAndNewlines))”."
                )
                .listRowSeparator(.hidden)
                .listRowBackground(Color.clear)
            }
        } else {
            ForEach(groupedKinds, id: \.self) { kind in
                Section {
                    ForEach(resultsForKind(kind), id: \.id) { result in
                        resultRow(result)
                    }
                } header: {
                    Text(kindTitle(kind))
                } footer: {
                    if kind == groupedKinds.last {
                        SnapshotNote(
                            fromCache: adapter.state?.searchFromCache == true,
                            generatedAt: epochDate(adapter.state?.searchGeneratedAtEpochSeconds ?? nil)
                        )
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func resultRow(_ result: MobileSearchResult) -> some View {
        let selection = selectionFor(result)
        if let selection {
            if horizontalSizeClass == .regular {
                SearchResultRow(result: result).tag(selection)
            } else {
                NavigationLink(value: selection) {
                    SearchResultRow(result: result)
                }
            }
        } else {
            SearchResultRow(result: result)
        }
    }

    private func selectionFor(_ result: MobileSearchResult) -> SpaceItemSelection? {
        switch result.kind.lowercased() {
        case "task":
            return .task(TaskRef(spaceSlug: result.spaceSlug, id: result.id))
        case "recipe":
            return .recipe(RecipeRef(spaceSlug: result.spaceSlug, id: result.id))
        default:
            return nil
        }
    }

    private var results: [MobileSearchResult] {
        adapter.state?.searchResults ?? []
    }

    private var groupedKinds: [String] {
        let kinds = Set(results.map { $0.kind.lowercased() })
        return kinds.sorted { lhs, rhs in
            let l = kindRank(lhs), r = kindRank(rhs)
            return l == r ? lhs < rhs : l < r
        }
    }

    private func resultsForKind(_ kind: String) -> [MobileSearchResult] {
        results.filter { $0.kind.lowercased() == kind }
    }

    private func kindRank(_ kind: String) -> Int {
        switch kind {
        case "task": return 0
        case "recipe": return 1
        default: return 2
        }
    }

    private func kindTitle(_ kind: String) -> String {
        switch kind {
        case "task": return "Tasks"
        case "recipe": return "Recipes"
        default: return kind.capitalized
        }
    }
}

private struct SearchResultRow: View {
    let result: MobileSearchResult

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Label(result.title, systemImage: icon)
                .font(.headline)
            if !result.detail.isEmpty {
                Text(result.detail)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }
            Text(result.spaceSlug)
                .font(.caption)
                .foregroundStyle(.tertiary)
        }
        .padding(.vertical, 2)
        .accessibilityElement(children: .combine)
    }

    private var icon: String {
        switch result.kind.lowercased() {
        case "task": return "checkmark.circle"
        case "recipe": return "fork.knife"
        default: return "doc"
        }
    }
}

// MARK: - Account tab

private struct AccountTab: View {
    @EnvironmentObject private var adapter: MobileCoreAdapter
    @State private var editingProfile = false
    @State private var confirmingSignOut = false

    var body: some View {
        NavigationStack {
            Form {
                if let error = adapter.state?.error {
                    Section {
                        ErrorBanner(error: error, retry: nil)
                            .listRowInsets(EdgeInsets())
                            .listRowBackground(Color.clear)
                    }
                }

                Section("Profile") {
                    if let user = adapter.state?.user {
                        LabeledContent("Name", value: user.name)
                        LabeledContent("Email", value: user.email)
                        if user.isOwner {
                            Label("Server Owner", systemImage: "star.fill")
                                .foregroundStyle(.secondary)
                        }
                        Button("Edit Profile") { editingProfile = true }
                    } else {
                        LoadingRow(label: "Loading profile…")
                    }
                }

                Section("Server") {
                    LabeledContent("Address", value: adapter.state?.server.baseUrl ?? "—")
                    LabeledContent("Server ID", value: adapter.state?.server.serverId ?? "—")
                    LabeledContent("Account ID", value: adapter.state?.accountId ?? "—")
                    if let label = adapter.state?.authConfig?.oidcLabel,
                       adapter.state?.authConfig?.oidcEnabled == true {
                        LabeledContent("Sign-In Method", value: label)
                    }
                }

                Section("Notifications") {
                    Button {
                        Task {
                            _ = await AppleWidgetBridge.shared.requestAuthorization()
                        }
                    } label: {
                        VStack(alignment: .leading, spacing: 4) {
                            Label("Allow Notifications", systemImage: "bell.badge")
                            Text("Choose whether Horologia can send notifications.")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }
                    .accessibilityLabel("Allow Horologia Notifications")
                }

                Section {
                    Button(role: .destructive) { confirmingSignOut = true } label: {
                        HStack {
                            Spacer()
                            Text("Sign Out")
                            Spacer()
                        }
                    }
                }
            }
            .navigationTitle("Account")
            .sheet(isPresented: $editingProfile) {
                if let user = adapter.state?.user {
                    ProfileEditView(user: user)
                }
            }
            .confirmationDialog(
                "Sign Out?",
                isPresented: $confirmingSignOut,
                titleVisibility: .visible
            ) {
                Button("Sign Out", role: .destructive) { adapter.signOut() }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("This removes the saved session and cached data from this device.")
            }
        }
    }
}

private struct ProfileEditView: View {
    let user: MobileUser

    @EnvironmentObject private var adapter: MobileCoreAdapter
    @Environment(\.dismiss) private var dismiss
    @State private var name: String
    @State private var email: String
    @State private var saving = false

    init(user: MobileUser) {
        self.user = user
        _name = State(initialValue: user.name)
        _email = State(initialValue: user.email)
    }

    private var trimmedName: String { name.trimmingCharacters(in: .whitespacesAndNewlines) }
    private var trimmedEmail: String { email.trimmingCharacters(in: .whitespacesAndNewlines) }

    private var hasChanges: Bool {
        trimmedName != user.name || trimmedEmail != user.email
    }

    var body: some View {
        NavigationStack {
            Form {
                if let error = adapter.state?.error {
                    Section {
                        ErrorBanner(error: error, retry: nil)
                            .listRowInsets(EdgeInsets())
                            .listRowBackground(Color.clear)
                    }
                }

                Section {
                    TextField("Name", text: $name)
                        .textContentType(.name)
                    TextField("Email", text: $email)
                        .textContentType(.emailAddress)
                        .keyboardType(.emailAddress)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }
            }
            .navigationTitle("Edit Profile")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    if saving || adapter.state?.loading.accountUpdate == true {
                        ProgressView()
                    } else {
                        Button("Save", action: save)
                            .disabled(trimmedName.isEmpty || !hasChanges)
                    }
                }
            }
        }
    }

    private func save() {
        let newName = trimmedName != user.name ? trimmedName : nil
        let newEmail = trimmedEmail != user.email ? trimmedEmail : nil
        Task {
            saving = true
            await adapter.updateProfile(name: newName, email: newEmail)
            saving = false
            if adapter.state?.error == nil {
                dismiss()
            }
        }
    }
}

// MARK: - Task detail & edit

private struct TaskDetailView: View {
    let ref: TaskRef

    @EnvironmentObject private var adapter: MobileCoreAdapter
    @State private var editing = false

    private var task: MobileTask? {
        guard let task = adapter.state?.selectedTask,
              task.id == ref.id, task.spaceSlug == ref.spaceSlug else { return nil }
        return task
    }

    var body: some View {
        Group {
            if let task {
                Form {
                    if let error = adapter.state?.error {
                        Section {
                            ErrorBanner(error: error) {
                                Task { await adapter.ensureTaskSelected(spaceSlug: ref.spaceSlug, taskId: ref.id) }
                            }
                            .listRowInsets(EdgeInsets())
                            .listRowBackground(Color.clear)
                        }
                    }

                    Section {
                        LabeledContent("Status") { StatusBadge(status: task.status) }
                        if let due = task.dueText {
                            LabeledContent("Due", value: due)
                        }
                        LabeledContent("Effort", value: task.effort ?? "None")
                        LabeledContent("Priority", value: task.priority ?? "None")
                        LabeledContent("Space", value: task.spaceSlug)
                    }

                    if !task.description_.isEmpty {
                        Section("Description") {
                            Text(task.description_)
                        }
                    }

                    if !task.tags.isEmpty {
                        Section("Tags") {
                            Text(task.tags.joined(separator: ", "))
                        }
                    }
                }
            } else {
                VStack(spacing: 12) {
                    ProgressView()
                        .controlSize(.large)
                    Text("Loading task…")
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .navigationTitle(task?.title ?? "Task")
        .toolbar {
            if task != nil {
                ToolbarItem(placement: .primaryAction) {
                    if adapter.state?.loading.taskUpdate == true {
                        ProgressView()
                    } else {
                        Button("Edit") { editing = true }
                    }
                }
            }
        }
        .sheet(isPresented: $editing) {
            if let task {
                TaskEditView(task: task)
            }
        }
        .task(id: ref) { await adapter.ensureTaskSelected(spaceSlug: ref.spaceSlug, taskId: ref.id) }
        .onAppear { Task { await adapter.ensureTaskSelected(spaceSlug: ref.spaceSlug, taskId: ref.id) } }
    }
}

private struct TaskEditView: View {
    let task: MobileTask

    @EnvironmentObject private var adapter: MobileCoreAdapter
    @Environment(\.dismiss) private var dismiss

    @State private var title: String
    @State private var details: String
    @State private var status: String
    @State private var effort: String?
    @State private var priority: String?
    @State private var tagsText: String
    @State private var dueMode: DueMode
    @State private var dueDate: Date
    @State private var saving = false

    private enum DueMode: Hashable {
        case keep, set, clear
    }

    private static let dayFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter
    }()

    private static let statusOptions = ["pending", "in_progress", "blocked", "completed", "cancelled"]
    private static let effortOptions = ["xs", "s", "m", "l", "xl"]
    private static let priorityOptions = ["low", "medium", "high", "urgent"]

    init(task: MobileTask) {
        self.task = task
        _title = State(initialValue: task.title)
        _details = State(initialValue: task.description_)
        _status = State(initialValue: task.status)
        _effort = State(initialValue: task.effort)
        _priority = State(initialValue: task.priority)
        _tagsText = State(initialValue: task.tags.joined(separator: ", "))
        _dueMode = State(initialValue: .keep)
        _dueDate = State(initialValue: Date())
    }

    private var parsedTags: [String] {
        tagsText.split(separator: ",")
            .map { $0.trimmingCharacters(in: .whitespaces) }
            .filter { !$0.isEmpty }
    }

    private var hasChanges: Bool {
        title.trimmingCharacters(in: .whitespacesAndNewlines) != task.title
            || details != task.description_
            || status != task.status
            || effort != task.effort
            || priority != task.priority
            || parsedTags != task.tags
            || dueMode != .keep
    }

    var body: some View {
        NavigationStack {
            Form {
                if let error = adapter.state?.error {
                    Section {
                        ErrorBanner(error: error, retry: nil)
                            .listRowInsets(EdgeInsets())
                            .listRowBackground(Color.clear)
                    }
                }

                Section {
                    TextField("Title", text: $title)
                }

                Section("Description") {
                    TextEditor(text: $details)
                        .frame(minHeight: 88)
                        .accessibilityLabel("Description")
                }

                Section("Scheduling") {
                    Picker("Status", selection: $status) {
                        ForEach(statusChoices, id: \.self) { option in
                            Text(displayName(option)).tag(option)
                        }
                    }
                    Picker("Due Date", selection: $dueMode) {
                        Text(task.dueText.map { "Keep \($0)" } ?? "No Due Date").tag(DueMode.keep)
                        Text("Set Date…").tag(DueMode.set)
                        if task.dueText != nil {
                            Text("Clear Due Date").tag(DueMode.clear)
                        }
                    }
                    if dueMode == .set {
                        DatePicker("Due", selection: $dueDate, displayedComponents: .date)
                    }
                }

                Section("Classification") {
                    Picker("Effort", selection: $effort) {
                        Text("None").tag(String?.none)
                        ForEach(effortChoices, id: \.self) { option in
                            Text(option.uppercased()).tag(String?.some(option))
                        }
                    }
                    Picker("Priority", selection: $priority) {
                        Text("None").tag(String?.none)
                        ForEach(priorityChoices, id: \.self) { option in
                            Text(displayName(option)).tag(String?.some(option))
                        }
                    }
                    TextField("Tags (comma separated)", text: $tagsText)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }
            }
            .navigationTitle("Edit Task")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    if saving || adapter.state?.loading.taskUpdate == true {
                        ProgressView()
                    } else {
                        Button("Save", action: save)
                            .disabled(title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || !hasChanges)
                    }
                }
            }
        }
    }

    private var statusChoices: [String] {
        Self.statusOptions.contains(task.status) ? Self.statusOptions : [task.status] + Self.statusOptions
    }

    private var effortChoices: [String] {
        if let current = task.effort, !Self.effortOptions.contains(current) {
            return [current] + Self.effortOptions
        }
        return Self.effortOptions
    }

    private var priorityChoices: [String] {
        if let current = task.priority, !Self.priorityOptions.contains(current) {
            return [current] + Self.priorityOptions
        }
        return Self.priorityOptions
    }

    private func displayName(_ raw: String) -> String {
        raw.replacingOccurrences(of: "_", with: " ").capitalized
    }

    private func save() {
        let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
        let update = MobileTaskUpdate(
            title: trimmedTitle != task.title ? trimmedTitle : nil,
            description: details != task.description_ ? details : nil,
            status: status != task.status ? status : nil,
            effort: stringPatch(original: task.effort, edited: effort),
            priority: stringPatch(original: task.priority, edited: priority),
            recurrenceType: nil,
            recurrenceRule: PatchFieldAbsent.shared,
            assigneeIds: nil,
            rotationPool: nil,
            tags: parsedTags != task.tags ? parsedTags : nil,
            due: duePatch,
            overdueActionRule: PatchFieldAbsent.shared
        )
        Task {
            saving = true
            await adapter.updateTask(spaceSlug: task.spaceSlug, taskId: task.id, update: update)
            saving = false
            // Saved state is only displayed after the core finishes: on success
            // the sheet dismisses and the detail shows the core's saved task.
            if adapter.state?.error == nil {
                dismiss()
            }
        }
    }

    private var duePatch: any PatchField {
        switch dueMode {
        case .keep:
            return PatchFieldAbsent.shared
        case .clear:
            return PatchFieldNull.shared
        case .set:
            let due = MobileTaskDue(
                date: Self.dayFormatter.string(from: dueDate),
                timezone: TimeZone.current.identifier
            )
            return PatchFieldValue<MobileTaskDue>(value: due)
        }
    }
}

// MARK: - Recipe detail & edit

private struct RecipeDetailView: View {
    let ref: RecipeRef

    @EnvironmentObject private var adapter: MobileCoreAdapter
    @State private var editing = false

    private var recipe: MobileRecipe? {
        guard let recipe = adapter.state?.selectedRecipe,
              recipe.id == ref.id, recipe.spaceSlug == ref.spaceSlug else { return nil }
        return recipe
    }

    var body: some View {
        Group {
            if let recipe {
                Form {
                    if let error = adapter.state?.error {
                        Section {
                            ErrorBanner(error: error) {
                                Task { await adapter.ensureRecipeSelected(spaceSlug: ref.spaceSlug, recipeId: ref.id) }
                            }
                            .listRowInsets(EdgeInsets())
                            .listRowBackground(Color.clear)
                        }
                    }

                    Section {
                        LabeledContent("Space", value: recipe.spaceSlug)
                    }

                    if !recipe.description_.isEmpty {
                        Section("Description") {
                            Text(recipe.description_)
                        }
                    }

                    if !recipe.tags.isEmpty {
                        Section("Tags") {
                            Text(recipe.tags.joined(separator: ", "))
                        }
                    }
                }
            } else {
                VStack(spacing: 12) {
                    ProgressView()
                        .controlSize(.large)
                    Text("Loading recipe…")
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .navigationTitle(recipe?.title ?? "Recipe")
        .toolbar {
            if recipe != nil {
                ToolbarItem(placement: .primaryAction) {
                    if adapter.state?.loading.recipeUpdate == true {
                        ProgressView()
                    } else {
                        Button("Edit") { editing = true }
                    }
                }
            }
        }
        .sheet(isPresented: $editing) {
            if let recipe {
                RecipeEditView(recipe: recipe)
            }
        }
        .task(id: ref) { await adapter.ensureRecipeSelected(spaceSlug: ref.spaceSlug, recipeId: ref.id) }
        .onAppear { Task { await adapter.ensureRecipeSelected(spaceSlug: ref.spaceSlug, recipeId: ref.id) } }
    }
}

private struct RecipeEditView: View {
    let recipe: MobileRecipe

    @EnvironmentObject private var adapter: MobileCoreAdapter
    @Environment(\.dismiss) private var dismiss

    @State private var title: String
    @State private var details: String
    @State private var tagsText: String
    @State private var saving = false

    init(recipe: MobileRecipe) {
        self.recipe = recipe
        _title = State(initialValue: recipe.title)
        _details = State(initialValue: recipe.description_)
        _tagsText = State(initialValue: recipe.tags.joined(separator: ", "))
    }

    private var parsedTags: [String] {
        tagsText.split(separator: ",")
            .map { $0.trimmingCharacters(in: .whitespaces) }
            .filter { !$0.isEmpty }
    }

    private var hasChanges: Bool {
        title.trimmingCharacters(in: .whitespacesAndNewlines) != recipe.title
            || details != recipe.description_
            || parsedTags != recipe.tags
    }

    var body: some View {
        NavigationStack {
            Form {
                if let error = adapter.state?.error {
                    Section {
                        ErrorBanner(error: error, retry: nil)
                            .listRowInsets(EdgeInsets())
                            .listRowBackground(Color.clear)
                    }
                }

                Section {
                    TextField("Title", text: $title)
                }

                Section("Description") {
                    TextEditor(text: $details)
                        .frame(minHeight: 120)
                        .accessibilityLabel("Description")
                }

                Section {
                    TextField("Tags (comma separated)", text: $tagsText)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }
            }
            .navigationTitle("Edit Recipe")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    if saving || adapter.state?.loading.recipeUpdate == true {
                        ProgressView()
                    } else {
                        Button("Save", action: save)
                            .disabled(title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || !hasChanges)
                    }
                }
            }
        }
    }

    private func save() {
        let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
        let update = MobileRecipeUpdate(
            title: trimmedTitle != recipe.title ? trimmedTitle : nil,
            description: details != recipe.description_ ? details : nil,
            yield: PatchFieldAbsent.shared,
            prepMinutes: PatchFieldAbsent.shared,
            cookMinutes: PatchFieldAbsent.shared,
            tags: parsedTags != recipe.tags ? parsedTags : nil,
            ingredientSections: nil,
            instructionSections: nil
        )
        Task {
            saving = true
            await adapter.updateRecipe(spaceSlug: recipe.spaceSlug, recipeId: recipe.id, update: update)
            saving = false
            if adapter.state?.error == nil {
                dismiss()
            }
        }
    }
}

// MARK: - Rows & shared components

private struct TaskRow: View {
    let task: MobileTask

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(task.title)
                .font(.headline)
            HStack(spacing: 8) {
                StatusBadge(status: task.status)
                if let due = task.dueText {
                    Label(due, systemImage: "calendar")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                if let priority = task.priority {
                    Label(priority.capitalized, systemImage: "flag")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            if !task.tags.isEmpty {
                Text(task.tags.joined(separator: " · "))
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .lineLimit(1)
            }
        }
        .padding(.vertical, 2)
        .accessibilityElement(children: .combine)
        .accessibilityHint("Opens task details")
    }
}

private struct RecipeRow: View {
    let recipe: MobileRecipe

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(recipe.title)
                .font(.headline)
            if !recipe.description_.isEmpty {
                Text(recipe.description_)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }
            if !recipe.tags.isEmpty {
                Text(recipe.tags.joined(separator: " · "))
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .lineLimit(1)
            }
        }
        .padding(.vertical, 2)
        .accessibilityElement(children: .combine)
        .accessibilityHint("Opens recipe details")
    }
}

private struct StatusBadge: View {
    let status: String

    var body: some View {
        Text(status.replacingOccurrences(of: "_", with: " ").capitalized)
            .font(.caption.weight(.medium))
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(tint.opacity(0.14), in: Capsule())
            .foregroundStyle(tint)
            .accessibilityLabel("Status: \(status.replacingOccurrences(of: "_", with: " "))")
    }

    private var tint: Color {
        switch status.lowercased() {
        case "completed", "complete", "done":
            return .green
        case "in_progress", "in-progress", "active", "started":
            return .accentColor
        case "blocked":
            return .red
        case "cancelled", "canceled":
            return .gray
        case "pending", "todo", "open":
            return .orange
        default:
            return .secondary
        }
    }
}

private struct ErrorBanner: View {
    let error: MobileAppError
    var retry: (() -> Void)?

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: "exclamationmark.triangle.fill")
                .foregroundStyle(.red)
                .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: 2) {
                Text(error.message)
                    .font(.callout)
                if let code = error.statusCode {
                    Text("HTTP \(code.int32Value)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            Spacer(minLength: 8)
            if let retry {
                Button("Retry", action: retry)
                    .font(.callout.weight(.semibold))
            }
        }
        .padding(12)
        .background(Color.red.opacity(0.08), in: RoundedRectangle(cornerRadius: 12))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Error: \(error.message)")
    }
}

private struct EmptyStateView: View {
    let icon: String
    let title: String
    let message: String

    var body: some View {
        VStack(spacing: 10) {
            Image(systemName: icon)
                .font(.largeTitle)
                .foregroundStyle(.secondary)
                .accessibilityHidden(true)
            Text(title)
                .font(.headline)
            Text(message)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
        }
        .padding(32)
        .frame(maxWidth: .infinity)
        .accessibilityElement(children: .combine)
    }
}

private struct LoadingRow: View {
    let label: String

    var body: some View {
        HStack(spacing: 12) {
            ProgressView()
            Text(label)
                .foregroundStyle(.secondary)
        }
        .accessibilityElement(children: .combine)
    }
}

private struct PaginationRow: View {
    let isLoading: Bool
    let loadMore: () -> Void

    var body: some View {
        HStack {
            Spacer()
            if isLoading {
                ProgressView("Loading more…")
            } else {
                Button("Load More", action: loadMore)
            }
            Spacer()
        }
        .onAppear {
            if !isLoading {
                loadMore()
            }
        }
    }
}

private struct SnapshotNote: View {
    let fromCache: Bool
    let generatedAt: Date?

    var body: some View {
        if let generatedAt {
            HStack(spacing: 4) {
                if fromCache {
                    Label("Cached", systemImage: "internaldrive")
                }
                Text("Updated \(generatedAt, style: .relative)")
            }
            .accessibilityElement(children: .combine)
        }
    }
}

// MARK: - Helpers

private func epochDate(_ value: KotlinLong?) -> Date? {
    guard let value else { return nil }
    return Date(timeIntervalSince1970: TimeInterval(value.int64Value))
}

/// Maps an optional string edit to the core's patch semantics: unchanged →
/// absent, cleared → explicit null, changed → new value.
private func stringPatch(original: String?, edited: String?) -> any PatchField {
    if original == edited {
        return PatchFieldAbsent.shared
    }
    guard let edited else {
        return PatchFieldNull.shared
    }
    return PatchFieldValue<NSString>(value: edited as NSString)
}
