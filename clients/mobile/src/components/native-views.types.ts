import type { components } from "@horologia/client-core/schema";

export type Task = components["schemas"]["Task"];
export type Recipe = components["schemas"]["Recipe"];
export type RecipeSummary = components["schemas"]["RecipeSummary"];
export type Space = components["schemas"]["Space"];
export type User = components["schemas"]["User"];
export type ServerInfo = components["schemas"]["ServerInfoResponse"];

export interface PropertyItem {
  label: string;
  value: string;
}

export interface DetailSection {
  title: string;
  properties?: PropertyItem[];
  paragraphs?: string[];
  numbered?: boolean;
}

export interface OnboardingViewProps {
  status:
    | "restoring"
    | "signed-out"
    | "connecting"
    | "authorizing"
    | "signing-in"
    | "signing-out"
    | "error";
  detail: string | null;
  serverName: string | null;
  initialServerUrl: string;
  onConnect: (serverUrl: string) => void;
  onSignIn: () => void;
  onCancel: () => void;
  onRecover: () => void;
}

export interface TaskListViewProps {
  emptyTitle?: string;
  emptyDetail?: string;
  tasks: Task[];
  selectedTaskId?: string | undefined;
  source: "network" | "cache";
  timestamp: string | null;
  isStale: boolean;
  isInitialLoading: boolean;
  isRefreshing: boolean;
  isLoadingMore: boolean;
  initialError: string | null;
  loadMoreError: string | null;
  canLoadMore: boolean;
  cachedHasMore: boolean;
  onSelect: (task: Task) => void;
  onRefresh: () => Promise<void>;
  onLoadMore: () => void;
  onRetry: () => void;
}

export interface TaskDetailViewProps {
  task: Task | null;
  isLoading: boolean;
  error: string | null;
  showTitle?: boolean;
  emptyTitle?: string;
  emptyDetail?: string;
  onRetry: () => void;
}

export interface LibraryHubViewProps {
  spaces: Space[];
  recipePreview: RecipeSummary[];
  isLoading: boolean;
  error: string | null;
  onOpenSpaces: () => void;
  onOpenRecipes: () => void;
  onOpenRecipe: (recipe: RecipeSummary) => void;
  onRetry: () => void;
}

export interface SpacesViewProps {
  spaces: Space[];
  isLoading: boolean;
  error: string | null;
  onOpen: (space: Space) => void;
  onRetry: () => void;
}

export interface SpaceWorkspaceViewProps {
  space: Space | null;
  isLoading: boolean;
  error: string | null;
  onOpenTasks: () => void;
  onOpenRecipes: () => void;
  onRetry: () => void;
}

export interface RecipeListViewProps {
  recipes: RecipeSummary[];
  spacesBySlug: ReadonlyMap<string, string>;
  scopedSpaceName?: string | undefined;
  isLoading: boolean;
  isRefreshing: boolean;
  isLoadingMore: boolean;
  error: string | null;
  loadMoreError: string | null;
  canLoadMore: boolean;
  onOpen: (recipe: RecipeSummary) => void;
  onRefresh: () => Promise<void>;
  onLoadMore: () => void;
  onRetry: () => void;
}

export interface RecipeDetailViewProps {
  recipe: Recipe | null;
  spaceName: string | null;
  isLoading: boolean;
  error: string | null;
  showTitle?: boolean;
  onRetry: () => void;
}

export interface SearchResultItem {
  kind: "task" | "recipe";
  id: string;
  spaceSlug: string;
  title: string;
  meta: string;
}

export interface SearchViewProps {
  query: string;
  results: SearchResultItem[];
  isSearching: boolean;
  error: string | null;
  onOpen: (result: SearchResultItem) => void;
}

export interface AccountViewProps {
  user: User | null;
  serverUrl: string;
  serverInfo: ServerInfo | null;
  appVersion: string;
  isLoading: boolean;
  error: string | null;
  isSigningOut: boolean;
  signOutError: string | null;
  onRetry: () => void;
  onSignOut: () => void;
}
