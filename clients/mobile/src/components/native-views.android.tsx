import {
  AlertDialog,
  Button,
  CircularProgressIndicator,
  Column,
  HorizontalDivider,
  Host,
  LazyColumn,
  ListItem,
  OutlinedButton,
  OutlinedTextField,
  PullToRefreshBox,
  Row,
  Spacer,
  Text,
  TextButton,
  useNativeState,
} from "@expo/ui/jetpack-compose";
import {
  clickable,
  fillMaxSize,
  fillMaxWidth,
  padding,
  paddingAll,
  weight,
} from "@expo/ui/jetpack-compose/modifiers";
import { useState } from "react";

import {
  recipeDetailSections,
  recipeSubtitle,
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
    <NativeHost>
      <LazyColumn
        contentPadding={{ start: 24, top: 32, end: 24, bottom: 32 }}
        verticalArrangement={{ spacedBy: 16 }}
      >
        <Text style={{ typography: "displaySmall", fontWeight: "700" }}>Horologia</Text>
        <Text style={{ typography: "bodyLarge" }}>
          Household information and routines, on your own server.
        </Text>
        {busy ? <CircularProgressIndicator /> : null}
        {props.serverName ? (
          <SectionBlock title="Server">
            <PropertyRow label="Connected to" value={props.serverName} />
            {props.status === "signed-out" ? (
              <Button onClick={props.onSignIn}>
                <Text>Sign in</Text>
              </Button>
            ) : null}
          </SectionBlock>
        ) : null}
        {props.status !== "restoring" ? (
          <SectionBlock title={props.serverName ? "Use another server" : "Connect to your server"}>
            <OutlinedTextField
              value={serverUrl}
              singleLine
              keyboardOptions={{ keyboardType: "uri", autoCorrectEnabled: false }}
              onValueChange={setUrl}
              modifiers={[fillMaxWidth()]}
            >
              <OutlinedTextField.Label>
                <Text>Server URL</Text>
              </OutlinedTextField.Label>
              <OutlinedTextField.Placeholder>
                <Text>https://home.example.com</Text>
              </OutlinedTextField.Placeholder>
            </OutlinedTextField>
            <OutlinedButton
              onClick={() => props.onConnect(url)}
              enabled={!busy && url.trim().length > 0}
            >
              <Text>Connect</Text>
            </OutlinedButton>
          </SectionBlock>
        ) : null}
        {props.detail ? (
          <SectionBlock title={props.status === "error" ? "Needs attention" : "Status"}>
            <Text>{props.detail}</Text>
            {props.status === "authorizing" ? (
              <TextButton onClick={props.onCancel}>
                <Text>Cancel</Text>
              </TextButton>
            ) : null}
            {props.status === "error" ? (
              <OutlinedButton onClick={props.onRecover}>
                <Text>Recover session</Text>
              </OutlinedButton>
            ) : null}
          </SectionBlock>
        ) : null}
      </LazyColumn>
    </NativeHost>
  );
}

export function TaskListView(props: TaskListViewProps) {
  if (props.isInitialLoading && props.tasks.length === 0)
    return <LoadingState label="Loading your tasks…" />;
  if (props.initialError && props.tasks.length === 0)
    return (
      <ErrorState
        title="Could not load tasks"
        detail={props.initialError}
        onRetry={props.onRetry}
      />
    );
  return (
    <NativeHost>
      <PullToRefreshBox
        isRefreshing={props.isRefreshing}
        onRefresh={props.onRefresh}
        modifiers={[fillMaxSize()]}
      >
        <LazyColumn contentPadding={{ top: 8, bottom: 24 }} modifiers={[fillMaxSize()]}>
          {props.source === "cache" && props.timestamp ? (
            <CachedDataNotice
              timestamp={props.timestamp}
              stale={props.isStale}
              hasMore={props.cachedHasMore}
            />
          ) : null}
          {props.initialError && props.tasks.length > 0 ? (
            <ActionNotice
              title="Could not refresh"
              detail={props.initialError}
              action="Try again"
              onAction={props.onRetry}
            />
          ) : null}
          {props.tasks.length === 0 ? (
            <EmptyStateContent
              title={props.emptyTitle ?? "No tasks assigned to you"}
              detail={
                props.emptyDetail ?? "Tasks assigned to you across all spaces will appear here."
              }
            />
          ) : (
            props.tasks.map((task) => (
              <TaskRow
                key={`${task.spaceSlug}/${task.id}`}
                task={task}
                selected={task.id === props.selectedTaskId}
                onPress={() => props.onSelect(task)}
              />
            ))
          )}
          {props.loadMoreError ? (
            <ActionNotice
              title="Could not load more"
              detail={props.loadMoreError}
              action="Try again"
              onAction={props.onLoadMore}
            />
          ) : null}
          {props.canLoadMore ? (
            <Button
              onClick={props.onLoadMore}
              enabled={!props.isLoadingMore}
              modifiers={[padding(16, 12, 16, 12)]}
            >
              <Text>{props.isLoadingMore ? "Loading…" : "Load more"}</Text>
            </Button>
          ) : null}
        </LazyColumn>
      </PullToRefreshBox>
    </NativeHost>
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
    <NativeHost>
      <LazyColumn contentPadding={{ top: 8, bottom: 24 }} modifiers={[fillMaxSize()]}>
        <SectionHeading title="Browse" />
        <NavigationRow
          title="Spaces"
          detail={`${String(props.spaces.length)} available`}
          onPress={props.onOpenSpaces}
        />
        <NavigationRow title="Recipes" detail="Across all spaces" onPress={props.onOpenRecipes} />
        {props.recipePreview.length > 0 ? (
          <SectionHeading title="Recently updated recipes" />
        ) : null}
        {props.recipePreview.map((recipe) => (
          <RecipeRow
            key={`${recipe.spaceSlug}/${recipe.id}`}
            recipe={recipe}
            onPress={() => props.onOpenRecipe(recipe)}
          />
        ))}
      </LazyColumn>
    </NativeHost>
  );
}

export function SpacesView(props: SpacesViewProps) {
  if (props.isLoading) return <LoadingState label="Loading spaces…" />;
  if (props.error)
    return (
      <ErrorState title="Could not load spaces" detail={props.error} onRetry={props.onRetry} />
    );
  return (
    <NativeHost>
      <LazyColumn contentPadding={{ top: 8, bottom: 24 }} modifiers={[fillMaxSize()]}>
        {props.spaces.length === 0 ? (
          <EmptyStateContent
            title="No spaces"
            detail="Spaces created on your server will appear here."
          />
        ) : (
          props.spaces.map((space) => (
            <NavigationRow
              key={space.slug}
              title={space.name}
              detail={space.description || space.slug}
              onPress={() => props.onOpen(space)}
            />
          ))
        )}
      </LazyColumn>
    </NativeHost>
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
    <NativeHost>
      <LazyColumn contentPadding={{ top: 8, bottom: 24 }} modifiers={[fillMaxSize()]}>
        <SupportingNotice text={props.space.description || "This space has no description."} />
        <SectionHeading title="Workspace" />
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
      </LazyColumn>
    </NativeHost>
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
    <NativeHost>
      <PullToRefreshBox
        isRefreshing={props.isRefreshing}
        onRefresh={props.onRefresh}
        modifiers={[fillMaxSize()]}
      >
        <LazyColumn contentPadding={{ top: 8, bottom: 24 }} modifiers={[fillMaxSize()]}>
          {props.error && props.recipes.length > 0 ? (
            <ActionNotice
              title="Could not refresh recipes"
              detail={props.error}
              action="Try again"
              onAction={props.onRetry}
            />
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
            props.recipes.map((recipe) => (
              <RecipeRow
                key={`${recipe.spaceSlug}/${recipe.id}`}
                recipe={recipe}
                spaceName={props.spacesBySlug.get(recipe.spaceSlug)}
                onPress={() => props.onOpen(recipe)}
              />
            ))
          )}
          {props.loadMoreError ? (
            <ActionNotice
              title="Could not load more"
              detail={props.loadMoreError}
              action="Try again"
              onAction={props.onLoadMore}
            />
          ) : null}
          {props.canLoadMore ? (
            <Button
              onClick={props.onLoadMore}
              enabled={!props.isLoadingMore}
              modifiers={[padding(16, 12, 16, 12)]}
            >
              <Text>{props.isLoadingMore ? "Loading…" : "Load more"}</Text>
            </Button>
          ) : null}
        </LazyColumn>
      </PullToRefreshBox>
    </NativeHost>
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
    <NativeHost>
      <LazyColumn
        contentPadding={{ start: 16, top: 16, end: 16, bottom: 24 }}
        verticalArrangement={{ spacedBy: 8 }}
        modifiers={[fillMaxSize()]}
      >
        {props.error ? <ActionNotice title="Search failed" detail={props.error} /> : null}
        {!props.isSearching && !props.error && props.query.trim() === "" ? (
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
        {props.results.map((result) => (
          <SearchRow
            key={`${result.kind}/${result.spaceSlug}/${result.id}`}
            result={result}
            onPress={() => props.onOpen(result)}
          />
        ))}
      </LazyColumn>
    </NativeHost>
  );
}

export function AccountView(props: AccountViewProps) {
  const [confirming, setConfirming] = useState(false);
  if (props.isLoading) return <LoadingState label="Loading account…" />;
  return (
    <NativeHost>
      <LazyColumn contentPadding={{ top: 8, bottom: 24 }} modifiers={[fillMaxSize()]}>
        {props.error ? (
          <ActionNotice
            title="Could not refresh account"
            detail={props.error}
            action="Try again"
            onAction={props.onRetry}
          />
        ) : null}
        {props.user ? (
          <>
            <SectionHeading title="Account" />
            <PropertyRow label="Name" value={props.user.name} />
            <PropertyRow label="Email" value={props.user.email} />
            <PropertyRow label="Role" value={props.user.isOwner ? "Owner" : "Member"} />
          </>
        ) : null}
        <SectionHeading title="Active server" />
        <PropertyRow label="URL" value={props.serverUrl} />
        <PropertyRow
          label="API version"
          value={props.serverInfo ? String(props.serverInfo.apiVersion) : "Unavailable"}
        />
        <PropertyRow
          label="Capabilities"
          value={props.serverInfo?.capabilities.join(", ") || "Unavailable"}
        />
        <PropertyRow label="App version" value={props.appVersion} />
        {props.signOutError ? (
          <ActionNotice title="Sign-out notice" detail={props.signOutError} />
        ) : null}
        <TextButton
          onClick={() => setConfirming(true)}
          enabled={!props.isSigningOut}
          modifiers={[padding(16, 12, 16, 12)]}
        >
          <Text>{props.isSigningOut ? "Signing out…" : "Sign out"}</Text>
        </TextButton>
      </LazyColumn>
      {confirming ? (
        <AlertDialog onDismissRequest={() => setConfirming(false)}>
          <AlertDialog.Title>
            <Text>Sign out of Horologia?</Text>
          </AlertDialog.Title>
          <AlertDialog.Text>
            <Text>
              Your local credentials and saved task snapshot will be removed from this device.
            </Text>
          </AlertDialog.Text>
          <AlertDialog.ConfirmButton>
            <TextButton onClick={props.onSignOut}>
              <Text>Sign out</Text>
            </TextButton>
          </AlertDialog.ConfirmButton>
          <AlertDialog.DismissButton>
            <TextButton onClick={() => setConfirming(false)}>
              <Text>Cancel</Text>
            </TextButton>
          </AlertDialog.DismissButton>
        </AlertDialog>
      ) : null}
    </NativeHost>
  );
}

function NativeHost({ children }: { children: React.ReactNode }) {
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      {children}
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
    <ListItem modifiers={[clickable(onPress)]}>
      <ListItem.LeadingContent>
        <Text style={{ typography: "titleLarge" }}>{selected ? "●" : "○"}</Text>
      </ListItem.LeadingContent>
      <ListItem.HeadlineContent>
        <Text
          maxLines={2}
          overflow="ellipsis"
          style={{ typography: "bodyLarge", fontWeight: "500" }}
        >
          {task.title}
        </Text>
      </ListItem.HeadlineContent>
      <ListItem.SupportingContent>
        <Text maxLines={1} overflow="ellipsis" style={{ typography: "bodySmall" }}>
          {taskSubtitle(task)}
        </Text>
      </ListItem.SupportingContent>
    </ListItem>
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
    <ListItem modifiers={[clickable(onPress)]}>
      <ListItem.HeadlineContent>
        <Text
          maxLines={2}
          overflow="ellipsis"
          style={{ typography: "bodyLarge", fontWeight: "500" }}
        >
          {title}
        </Text>
      </ListItem.HeadlineContent>
      <ListItem.SupportingContent>
        <Text maxLines={2} overflow="ellipsis" style={{ typography: "bodySmall" }}>
          {detail}
        </Text>
      </ListItem.SupportingContent>
      <ListItem.TrailingContent>
        <Text>›</Text>
      </ListItem.TrailingContent>
    </ListItem>
  );
}

function DetailView({ title, sections }: { title: string | null; sections: DetailSection[] }) {
  return (
    <NativeHost>
      <LazyColumn contentPadding={{ top: 16, bottom: 32 }} modifiers={[fillMaxSize()]}>
        {title ? (
          <Text
            modifiers={[padding(16, 8, 16, 16)]}
            style={{ typography: "headlineMedium", fontWeight: "700" }}
          >
            {title}
          </Text>
        ) : null}
        {sections.map((section, index) => (
          <Column key={`${section.title}/${String(index)}`}>
            <SectionHeading title={section.title} />
            {section.properties?.map((property) => (
              <PropertyRow key={property.label} label={property.label} value={property.value} />
            ))}
            {section.paragraphs?.map((paragraph, paragraphIndex) => (
              <Row
                key={`${paragraph}/${String(paragraphIndex)}`}
                modifiers={[padding(16, 10, 16, 10)]}
              >
                {section.numbered ? (
                  <Text style={{ typography: "bodyMedium" }}>{String(paragraphIndex + 1)}. </Text>
                ) : null}
                <Text modifiers={[weight(1)]} style={{ typography: "bodyLarge" }}>
                  {paragraph}
                </Text>
              </Row>
            ))}
          </Column>
        ))}
      </LazyColumn>
    </NativeHost>
  );
}

function SectionBlock({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Column verticalArrangement={{ spacedBy: 10 }} modifiers={[fillMaxWidth(), paddingAll(16)]}>
      <Text style={{ typography: "titleMedium", fontWeight: "600" }}>{title}</Text>
      {children}
    </Column>
  );
}

function SectionHeading({ title }: { title: string }) {
  return (
    <Text
      modifiers={[padding(16, 18, 16, 8)]}
      style={{ typography: "titleMedium", fontWeight: "600" }}
    >
      {title}
    </Text>
  );
}

function PropertyRow({ label, value }: { label: string; value: string }) {
  return (
    <ListItem>
      <ListItem.HeadlineContent>
        <Text style={{ typography: "bodyMedium" }}>{label}</Text>
      </ListItem.HeadlineContent>
      <ListItem.SupportingContent>
        <Text style={{ typography: "bodyLarge" }}>{value}</Text>
      </ListItem.SupportingContent>
    </ListItem>
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
    <ActionNotice
      title={stale ? "Saved data may be out of date" : "Showing saved data"}
      detail={`Saved ${timestamp}. ${hasMore ? "More tasks may be available when the server reconnects." : "Pull to refresh when the server is available."}`}
    />
  );
}

function SupportingNotice({ text }: { text: string }) {
  return (
    <Text modifiers={[padding(16, 12, 16, 12)]} style={{ typography: "bodyMedium" }}>
      {text}
    </Text>
  );
}

function EmptyStateContent({ title, detail }: { title: string; detail: string }) {
  return (
    <Column
      horizontalAlignment="center"
      verticalArrangement={{ spacedBy: 8 }}
      modifiers={[fillMaxWidth(), padding(24, 48, 24, 48)]}
    >
      <Text style={{ typography: "titleLarge", fontWeight: "600", textAlign: "center" }}>
        {title}
      </Text>
      <Text style={{ typography: "bodyMedium", textAlign: "center" }}>{detail}</Text>
    </Column>
  );
}

function EmptyDetailState({ title, detail }: { title: string; detail: string }) {
  return (
    <NativeHost>
      <Column
        horizontalAlignment="center"
        verticalArrangement={{ spacedBy: 8 }}
        modifiers={[fillMaxSize(), paddingAll(24)]}
      >
        <Spacer modifiers={[weight(1)]} />
        <Text style={{ typography: "headlineSmall", fontWeight: "600", textAlign: "center" }}>
          {title}
        </Text>
        <Text style={{ typography: "bodyMedium", textAlign: "center" }}>{detail}</Text>
        <Spacer modifiers={[weight(1)]} />
      </Column>
    </NativeHost>
  );
}

function ActionNotice({
  title,
  detail,
  action,
  onAction,
}: {
  title: string;
  detail: string;
  action?: string;
  onAction?: () => void;
}) {
  return (
    <Column
      verticalArrangement={{ spacedBy: 8 }}
      modifiers={[fillMaxWidth(), padding(16, 16, 16, 16)]}
    >
      <Text style={{ typography: "titleMedium", fontWeight: "600" }}>{title}</Text>
      <Text style={{ typography: "bodyMedium" }}>{detail}</Text>
      {action && onAction ? (
        <OutlinedButton onClick={onAction}>
          <Text>{action}</Text>
        </OutlinedButton>
      ) : null}
      <HorizontalDivider />
    </Column>
  );
}

function LoadingState({ label }: { label: string }) {
  return (
    <NativeHost>
      <Column
        horizontalAlignment="center"
        verticalArrangement={{ spacedBy: 12 }}
        modifiers={[fillMaxSize()]}
      >
        <Spacer modifiers={[weight(1)]} />
        <CircularProgressIndicator />
        <Text>{label}</Text>
        <Spacer modifiers={[weight(1)]} />
      </Column>
    </NativeHost>
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
    <NativeHost>
      <Column
        horizontalAlignment="center"
        verticalArrangement={{ spacedBy: 12 }}
        modifiers={[fillMaxSize(), paddingAll(24)]}
      >
        <Spacer modifiers={[weight(1)]} />
        <Text style={{ typography: "headlineSmall", fontWeight: "600", textAlign: "center" }}>
          {title}
        </Text>
        <Text style={{ typography: "bodyMedium", textAlign: "center" }}>{detail}</Text>
        <Button onClick={onRetry}>
          <Text>Try again</Text>
        </Button>
        <Spacer modifiers={[weight(1)]} />
      </Column>
    </NativeHost>
  );
}
