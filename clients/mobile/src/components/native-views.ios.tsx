import {
  Button,
  ConfirmationDialog,
  ContentUnavailableView,
  Form,
  Host,
  HStack,
  LabeledContent,
  List,
  ProgressView,
  Section,
  Spacer,
  Text,
  TextField,
  useNativeState,
  VStack,
} from "@expo/ui/swift-ui";
import {
  accessibilityLabel,
  buttonStyle,
  contentShape,
  disabled,
  font,
  foregroundStyle,
  frame,
  keyboardType,
  lineLimit,
  listStyle,
  multilineTextAlignment,
  padding,
  refreshable,
  textContentType,
  shapes,
  textFieldStyle,
} from "@expo/ui/swift-ui/modifiers";
import { PlatformColor } from "react-native";
import { useState } from "react";

import {
  recipeDetailSections,
  recipeSubtitle,
  taskAccessibilityLabel,
  taskDetailSections,
  taskSubtitle,
} from "@/components/read-model";
import type {
  AccountViewProps,
  DetailSection,
  LibraryHubViewProps,
  OnboardingViewProps,
  RecipeDetailViewProps,
  RecipeListViewProps,
  SearchResultItem,
  SearchViewProps,
  SpacesViewProps,
  SpaceWorkspaceViewProps,
  Task,
  TaskDetailViewProps,
  TaskListViewProps,
} from "@/components/native-views.types";

const primary = foregroundStyle(PlatformColor("label"));
const secondary = foregroundStyle(PlatformColor("secondaryLabel"));
const tertiary = foregroundStyle(PlatformColor("tertiaryLabel"));

export function OnboardingView(props: OnboardingViewProps) {
  const serverUrl = useNativeState(props.initialServerUrl);
  const [url, setUrl] = useState(props.initialServerUrl);
  const busy =
    props.status === "restoring" ||
    props.status === "connecting" ||
    props.status === "authorizing" ||
    props.status === "signing-in" ||
    props.status === "signing-out";

  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <Form>
        <Section>
          <VStack spacing={12} modifiers={[frame({ maxWidth: 520 }), padding({ vertical: 24 })]}>
            <Text modifiers={[font({ textStyle: "largeTitle", weight: "bold" })]}>Horologia</Text>
            <Text modifiers={[secondary, multilineTextAlignment("center")]}>
              Household information and routines, on your own server.
            </Text>
            {busy ? <ProgressView /> : null}
          </VStack>
        </Section>
        {props.serverName ? (
          <Section title="Server">
            <LabeledContent label="Connected to">
              <Text>{props.serverName}</Text>
            </LabeledContent>
            {props.status === "signed-out" ? (
              <Button
                label="Sign In"
                systemImage="person.crop.circle"
                onPress={props.onSignIn}
                modifiers={[buttonStyle("borderedProminent")]}
              />
            ) : null}
          </Section>
        ) : null}
        {props.status !== "restoring" ? (
          <Section title={props.serverName ? "Use another server" : "Connect to your server"}>
            <TextField
              text={serverUrl}
              placeholder="https://home.example.com"
              onTextChange={setUrl}
              modifiers={[
                textFieldStyle("roundedBorder"),
                keyboardType("url"),
                textContentType("URL"),
              ]}
            />
            <Button
              label="Connect"
              systemImage="network"
              onPress={() => props.onConnect(url)}
              modifiers={[buttonStyle("bordered"), disabled(busy || url.trim().length === 0)]}
            />
          </Section>
        ) : null}
        {props.detail ? (
          <Section title={props.status === "error" ? "Needs attention" : "Status"}>
            <Text modifiers={props.status === "error" ? [] : [secondary]}>{props.detail}</Text>
            {props.status === "authorizing" ? (
              <Button label="Cancel" role="cancel" onPress={props.onCancel} />
            ) : null}
            {props.status === "error" ? (
              <Button
                label="Recover Session"
                onPress={props.onRecover}
                modifiers={[buttonStyle("bordered")]}
              />
            ) : null}
          </Section>
        ) : null}
      </Form>
    </Host>
  );
}

export function TaskListView(props: TaskListViewProps) {
  if (props.isInitialLoading && props.tasks.length === 0)
    return <LoadingState label="Loading your tasks…" />;
  if (props.initialError && props.tasks.length === 0) {
    return (
      <ErrorState
        title="Could not load tasks"
        detail={props.initialError}
        onRetry={props.onRetry}
      />
    );
  }
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <List modifiers={[listStyle("insetGrouped"), refreshable(async () => props.onRefresh())]}>
        {props.source === "cache" && props.timestamp ? (
          <CachedDataNotice
            timestamp={props.timestamp}
            stale={props.isStale}
            hasMore={props.cachedHasMore}
          />
        ) : null}
        {props.initialError && props.tasks.length > 0 ? (
          <Section title="Could not refresh">
            <Text>{props.initialError}</Text>
            <Button label="Try Again" onPress={props.onRetry} />
          </Section>
        ) : null}
        {props.tasks.length === 0 ? (
          <EmptyStateContent
            title={props.emptyTitle ?? "No tasks assigned to you"}
            detail={
              props.emptyDetail ?? "Tasks assigned to you across all spaces will appear here."
            }
          />
        ) : (
          <Section>
            {props.tasks.map((task) => (
              <TaskRow
                key={`${task.spaceSlug}/${task.id}`}
                task={task}
                selected={task.id === props.selectedTaskId}
                onPress={() => props.onSelect(task)}
              />
            ))}
          </Section>
        )}
        {props.loadMoreError ? (
          <Section title="Could not load more">
            <Text>{props.loadMoreError}</Text>
            <Button label="Try Again" onPress={props.onLoadMore} />
          </Section>
        ) : null}
        {props.canLoadMore ? (
          <Section>
            <Button
              label={props.isLoadingMore ? "Loading…" : "Load More"}
              onPress={props.onLoadMore}
              modifiers={[disabled(props.isLoadingMore)]}
            />
          </Section>
        ) : null}
      </List>
    </Host>
  );
}

export function TaskDetailView(props: TaskDetailViewProps) {
  if (props.isLoading) return <LoadingState label="Loading task…" />;
  if (props.error) {
    return <ErrorState title="Could not load task" detail={props.error} onRetry={props.onRetry} />;
  }
  if (!props.task) {
    return (
      <EmptyDetailState
        title={props.emptyTitle ?? "Task not found"}
        detail={props.emptyDetail ?? "This task is no longer available."}
      />
    );
  }
  return (
    <DetailView
      title={props.showTitle === false ? null : props.task.title}
      sections={taskDetailSections(props.task)}
    />
  );
}

export function LibraryHubView(props: LibraryHubViewProps) {
  if (props.isLoading) return <LoadingState label="Loading library…" />;
  if (props.error)
    return (
      <ErrorState title="Could not load library" detail={props.error} onRetry={props.onRetry} />
    );
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <List modifiers={[listStyle("insetGrouped")]}>
        <Section title="Browse">
          <NavigationRow
            title="Spaces"
            detail={`${String(props.spaces.length)} available`}
            onPress={props.onOpenSpaces}
          />
          <NavigationRow title="Recipes" detail="Across all spaces" onPress={props.onOpenRecipes} />
        </Section>
        {props.recipePreview.length > 0 ? (
          <Section title="Recently updated recipes">
            {props.recipePreview.map((recipe) => (
              <RecipeRow
                key={`${recipe.spaceSlug}/${recipe.id}`}
                recipe={recipe}
                onPress={() => props.onOpenRecipe(recipe)}
              />
            ))}
          </Section>
        ) : null}
      </List>
    </Host>
  );
}

export function SpacesView(props: SpacesViewProps) {
  if (props.isLoading) return <LoadingState label="Loading spaces…" />;
  if (props.error)
    return (
      <ErrorState title="Could not load spaces" detail={props.error} onRetry={props.onRetry} />
    );
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <List modifiers={[listStyle("insetGrouped")]}>
        {props.spaces.length === 0 ? (
          <EmptyStateContent
            title="No spaces"
            detail="Spaces created on your server will appear here."
          />
        ) : (
          <Section>
            {props.spaces.map((space) => (
              <NavigationRow
                key={space.slug}
                title={space.name}
                detail={space.description || space.slug}
                onPress={() => props.onOpen(space)}
              />
            ))}
          </Section>
        )}
      </List>
    </Host>
  );
}

export function SpaceWorkspaceView(props: SpaceWorkspaceViewProps) {
  if (props.isLoading) return <LoadingState label="Loading space…" />;
  if (props.error || !props.space)
    return (
      <ErrorState
        title="Could not load space"
        detail={props.error ?? "Space not found."}
        onRetry={props.onRetry}
      />
    );
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <List modifiers={[listStyle("insetGrouped")]}>
        <Section>
          <Text modifiers={[secondary]}>
            {props.space.description || "This space has no description."}
          </Text>
        </Section>
        <Section title="Workspace">
          <NavigationRow
            title="Tasks"
            detail="View tasks in this space"
            onPress={props.onOpenTasks}
          />
          <NavigationRow
            title="Recipes"
            detail="View recipes in this space"
            onPress={props.onOpenRecipes}
          />
        </Section>
      </List>
    </Host>
  );
}

export function RecipeListView(props: RecipeListViewProps) {
  if (props.isLoading && props.recipes.length === 0)
    return <LoadingState label="Loading recipes…" />;
  if (props.error && props.recipes.length === 0)
    return (
      <ErrorState title="Could not load recipes" detail={props.error} onRetry={props.onRetry} />
    );
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <List modifiers={[listStyle("insetGrouped"), refreshable(async () => props.onRefresh())]}>
        {props.error && props.recipes.length > 0 ? (
          <Section title="Could not refresh recipes">
            <Text>{props.error}</Text>
            <Button label="Try Again" onPress={props.onRetry} />
          </Section>
        ) : null}
        {props.recipes.length === 0 ? (
          <EmptyStateContent
            title="No recipes"
            detail={
              props.scopedSpaceName
                ? `Recipes in ${props.scopedSpaceName} will appear here.`
                : "Recipes across your spaces will appear here."
            }
          />
        ) : (
          <Section>
            {props.recipes.map((recipe) => (
              <RecipeRow
                key={`${recipe.spaceSlug}/${recipe.id}`}
                recipe={recipe}
                spaceName={props.spacesBySlug.get(recipe.spaceSlug)}
                onPress={() => props.onOpen(recipe)}
              />
            ))}
          </Section>
        )}
        {props.loadMoreError ? (
          <Section title="Could not load more">
            <Text>{props.loadMoreError}</Text>
            <Button label="Try Again" onPress={props.onLoadMore} />
          </Section>
        ) : null}
        {props.canLoadMore ? (
          <Section>
            <Button
              label={props.isLoadingMore ? "Loading…" : "Load More"}
              onPress={props.onLoadMore}
              modifiers={[disabled(props.isLoadingMore)]}
            />
          </Section>
        ) : null}
      </List>
    </Host>
  );
}

export function RecipeDetailView(props: RecipeDetailViewProps) {
  if (props.isLoading) return <LoadingState label="Loading recipe…" />;
  if (props.error || !props.recipe)
    return (
      <ErrorState
        title="Could not load recipe"
        detail={props.error ?? "Recipe not found."}
        onRetry={props.onRetry}
      />
    );
  const sections = recipeDetailSections(props.recipe);
  if (props.spaceName)
    sections[0]?.properties?.unshift({ label: "Space name", value: props.spaceName });
  return (
    <DetailView title={props.showTitle === false ? null : props.recipe.name} sections={sections} />
  );
}

export function SearchView(props: SearchViewProps) {
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <List modifiers={[listStyle("inset")]}>
        {props.error ? (
          <Section title="Search failed">
            <Text>{props.error}</Text>
          </Section>
        ) : null}
        {!props.error && props.query.trim() === "" ? (
          <EmptyStateContent
            title="Search Horologia"
            detail="Find tasks and recipes across every space."
          />
        ) : null}
        {!props.isSearching &&
        !props.error &&
        props.query.trim() !== "" &&
        props.results.length === 0 ? (
          <EmptyStateContent title="No matches" detail="Try a different word or phrase." />
        ) : null}
        {props.results.length > 0 ? (
          <Section>
            {props.results.map((result) => (
              <SearchRow
                key={`${result.kind}/${result.spaceSlug}/${result.id}`}
                result={result}
                onPress={() => props.onOpen(result)}
              />
            ))}
          </Section>
        ) : null}
      </List>
    </Host>
  );
}

export function AccountView(props: AccountViewProps) {
  const [confirming, setConfirming] = useState(false);
  if (props.isLoading) return <LoadingState label="Loading account…" />;
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <Form>
        {props.error ? (
          <Section title="Could not refresh account">
            <Text>{props.error}</Text>
            <Button label="Try Again" onPress={props.onRetry} />
          </Section>
        ) : null}
        {props.user ? (
          <Section title="Account">
            <LabeledContent label="Name">
              <Text>{props.user.name}</Text>
            </LabeledContent>
            <LabeledContent label="Email">
              <Text>{props.user.email}</Text>
            </LabeledContent>
            <LabeledContent label="Role">
              <Text>{props.user.isOwner ? "Owner" : "Member"}</Text>
            </LabeledContent>
          </Section>
        ) : null}
        <Section title="Active server">
          <LabeledContent label="URL">
            <Text modifiers={[lineLimit(2)]}>{props.serverUrl}</Text>
          </LabeledContent>
          <LabeledContent label="API version">
            <Text>{props.serverInfo ? String(props.serverInfo.apiVersion) : "Unavailable"}</Text>
          </LabeledContent>
          <LabeledContent label="Capabilities">
            <Text>{props.serverInfo?.capabilities.join(", ") || "Unavailable"}</Text>
          </LabeledContent>
          <LabeledContent label="App version">
            <Text>{props.appVersion}</Text>
          </LabeledContent>
        </Section>
        {props.signOutError ? (
          <Section title="Sign-out notice">
            <Text>{props.signOutError}</Text>
          </Section>
        ) : null}
        <Section>
          <ConfirmationDialog
            title="Sign out of Horologia?"
            isPresented={confirming}
            onIsPresentedChange={setConfirming}
          >
            <ConfirmationDialog.Trigger>
              <Button
                label={props.isSigningOut ? "Signing Out…" : "Sign Out"}
                role="destructive"
                onPress={() => setConfirming(true)}
                modifiers={[disabled(props.isSigningOut)]}
              />
            </ConfirmationDialog.Trigger>
            <ConfirmationDialog.Actions>
              <Button label="Sign Out" role="destructive" onPress={props.onSignOut} />
              <Button label="Cancel" role="cancel" onPress={() => setConfirming(false)} />
            </ConfirmationDialog.Actions>
            <ConfirmationDialog.Message>
              <Text>
                Your local credentials and saved task snapshot will be removed from this device.
              </Text>
            </ConfirmationDialog.Message>
          </ConfirmationDialog>
        </Section>
      </Form>
    </Host>
  );
}

function TaskRow({
  task,
  selected,
  onPress,
}: {
  task: Task;
  selected: boolean;
  onPress: () => void;
}) {
  return (
    <Button onPress={onPress} modifiers={[accessibilityLabel(taskAccessibilityLabel(task))]}>
      <HStack
        spacing={10}
        modifiers={[
          frame({ maxWidth: Infinity, alignment: "leading" }),
          contentShape(shapes.rectangle()),
          primary,
        ]}
      >
        <Text modifiers={[font({ textStyle: "title3" })]}>{selected ? "●" : "○"}</Text>
        <VStack alignment="leading" spacing={3}>
          <Text modifiers={[font({ textStyle: "body", weight: "medium" }), lineLimit(2)]}>
            {task.title}
          </Text>
          <Text modifiers={[font({ textStyle: "caption" }), secondary, lineLimit(1)]}>
            {taskSubtitle(task)}
          </Text>
        </VStack>
        <Spacer />
      </HStack>
    </Button>
  );
}

function RecipeRow({
  recipe,
  spaceName,
  onPress,
}: {
  recipe: RecipeListViewProps["recipes"][number];
  spaceName?: string | undefined;
  onPress: () => void;
}) {
  return (
    <NavigationRow
      title={recipe.name}
      detail={[spaceName, recipeSubtitle(recipe), recipe.tags.slice(0, 3).join(" · ")]
        .filter(Boolean)
        .join(" · ")}
      onPress={onPress}
    />
  );
}

function SearchRow({ result, onPress }: { result: SearchResultItem; onPress: () => void }) {
  return (
    <NavigationRow
      title={result.title}
      detail={`${result.kind === "task" ? "Task" : "Recipe"} · ${result.spaceSlug} · ${result.meta}`}
      onPress={onPress}
    />
  );
}

function NavigationRow({
  title,
  detail,
  onPress,
}: {
  title: string;
  detail: string;
  onPress: () => void;
}) {
  return (
    <Button onPress={onPress} modifiers={[accessibilityLabel(`${title}, ${detail}`)]}>
      <HStack
        spacing={10}
        modifiers={[
          frame({ maxWidth: Infinity, alignment: "leading" }),
          contentShape(shapes.rectangle()),
          primary,
        ]}
      >
        <VStack alignment="leading" spacing={3}>
          <Text modifiers={[font({ textStyle: "body", weight: "medium" }), lineLimit(2)]}>
            {title}
          </Text>
          <Text modifiers={[font({ textStyle: "caption" }), secondary, lineLimit(2)]}>
            {detail}
          </Text>
        </VStack>
        <Spacer />
        <Text modifiers={[tertiary]}>›</Text>
      </HStack>
    </Button>
  );
}

function CachedDataNotice({
  timestamp,
  stale,
  hasMore,
}: {
  timestamp: string;
  stale: boolean;
  hasMore: boolean;
}) {
  return (
    <Section title={stale ? "Saved data may be out of date" : "Showing saved data"}>
      <Text modifiers={[font({ textStyle: "footnote" }), secondary]}>
        Saved {timestamp}.{" "}
        {hasMore
          ? "More tasks may be available when the server reconnects."
          : "Pull to refresh when the server is available."}
      </Text>
    </Section>
  );
}

function DetailView({ title, sections }: { title: string | null; sections: DetailSection[] }) {
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <List modifiers={[listStyle("insetGrouped")]}>
        {title ? (
          <Section>
            <Text modifiers={[font({ textStyle: "title", weight: "bold" }), lineLimit(4)]}>
              {title}
            </Text>
          </Section>
        ) : null}
        {sections.map((section, index) => (
          <Section key={`${section.title}/${String(index)}`} title={section.title}>
            {section.properties?.map((property) => (
              <PropertyRow key={property.label} label={property.label} value={property.value} />
            ))}
            {section.paragraphs?.map((paragraph, paragraphIndex) => (
              <HStack key={`${paragraph}/${String(paragraphIndex)}`} alignment="top" spacing={8}>
                {section.numbered ? (
                  <Text modifiers={[secondary]}>{String(paragraphIndex + 1)}.</Text>
                ) : null}
                <Text modifiers={[frame({ maxWidth: Infinity, alignment: "leading" })]}>
                  {paragraph}
                </Text>
              </HStack>
            ))}
          </Section>
        ))}
      </List>
    </Host>
  );
}

function PropertyRow({ label, value }: { label: string; value: string }) {
  return (
    <LabeledContent label={label}>
      <Text modifiers={[multilineTextAlignment("trailing"), lineLimit(4)]}>{value}</Text>
    </LabeledContent>
  );
}

function LoadingState({ label }: { label: string }) {
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <VStack spacing={12} modifiers={[frame({ maxWidth: Infinity, maxHeight: Infinity })]}>
        <ProgressView />
        <Text modifiers={[secondary]}>{label}</Text>
      </VStack>
    </Host>
  );
}

function EmptyStateContent({ title, detail }: { title: string; detail: string }) {
  return <ContentUnavailableView title={title} systemImage="tray" description={detail} />;
}

function EmptyDetailState({ title, detail }: { title: string; detail: string }) {
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <VStack
        spacing={12}
        modifiers={[
          frame({ maxWidth: Infinity, maxHeight: Infinity }),
          padding({ horizontal: 24 }),
        ]}
      >
        <ContentUnavailableView
          title={title}
          systemImage="rectangle.split.2x1"
          description={detail}
        />
      </VStack>
    </Host>
  );
}

function ErrorState({
  title,
  detail,
  onRetry,
}: {
  title: string;
  detail: string;
  onRetry: () => void;
}) {
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <VStack
        spacing={16}
        modifiers={[
          frame({ maxWidth: Infinity, maxHeight: Infinity }),
          padding({ horizontal: 24 }),
        ]}
      >
        <ContentUnavailableView
          title={title}
          systemImage="exclamationmark.triangle"
          description={detail}
        />
        <Button
          label="Try Again"
          onPress={onRetry}
          modifiers={[buttonStyle("borderedProminent")]}
        />
      </VStack>
    </Host>
  );
}
