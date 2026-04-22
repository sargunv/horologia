import { createLink } from "@tanstack/react-router";
import { ChevronDown, UserRound } from "lucide-react";
import type { components } from "../api/schema.d.ts";
import { Card } from "../ui/Card.tsx";

type ActivityLogEntry = components["schemas"]["ActivityLogEntry"];
type ActivityEntityType = components["schemas"]["ActivityEntityType"];
type ActivityAction = components["schemas"]["ActivityAction"];

export interface ActivityMember {
  userName: string;
}

export interface ActivityFeedProps {
  entries: ActivityLogEntry[];
  hasNextPage: boolean;
  fetchNextPage: () => void;
  isFetchingNextPage: boolean;
  /** Map of userId → member info for actor name resolution */
  memberMap?: Map<string, ActivityMember>;
  /** When true, each entry shows the space as context (for cross-space feeds) */
  showSpace?: boolean;
}

// ── Helpers ────────────────────────────────────────────────────────────────

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const diffMs = Date.now() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);
  if (diffSec < 60) return "just now";
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHr < 24) return `${diffHr}h ago`;
  if (diffDay < 7) return `${diffDay}d ago`;
  return date.toLocaleDateString();
}

function actorLabel(entry: ActivityLogEntry, memberMap?: Map<string, ActivityMember>): string {
  let name = "System";
  if (entry.actorId) {
    const member = memberMap?.get(entry.actorId);
    name = member ? member.userName : `User ${entry.actorId.slice(0, 8)}`;
  }
  if (entry.tokenName) {
    name = `${name} (via ${entry.tokenName})`;
  }
  return name;
}

const ENTITY_TYPE_LABEL: Record<ActivityEntityType, string> = {
  task: "task",
  space: "space",
  member: "member",
  tag: "tag",
  status: "status",
  effort_level: "effort level",
  priority_level: "priority level",
  relation: "relation",
};

const ACTION_LABEL: Record<ActivityAction, string> = {
  created: "created",
  updated: "updated",
  deleted: "deleted",
};

/** Turn camelCase / snake_case field names into readable labels */
function humanizeField(field: string): string {
  const overrides: Record<string, string> = {
    assigneeIds: "assignees",
    rotationPool: "rotation pool",
    recurrenceType: "recurrence type",
    recurrenceRule: "recurrence rule",
    overdueActionRule: "overdue action",
    due: "due date",
  };
  if (overrides[field]) return overrides[field];
  // split camelCase then replace underscores
  return field
    .replace(/([A-Z])/g, " $1")
    .replace(/_/g, " ")
    .toLowerCase()
    .trim();
}

// ── Sub-components ──────────────────────────────────────────────────────────

const TaskLink = createLink("a");
const SpaceLink = createLink("a");

function EntityRef({
  entityType,
  entityId,
  spaceSlug,
}: {
  entityType: ActivityEntityType;
  entityId: string;
  spaceSlug: string;
}) {
  const label = `${ENTITY_TYPE_LABEL[entityType] ?? entityType} ${entityId}`;
  if (entityType === "task") {
    return (
      <TaskLink
        to="/spaces/$spaceSlug/tasks/$taskId"
        params={{ spaceSlug, taskId: entityId }}
        className="text-primary font-mono hover:underline"
      >
        {label}
      </TaskLink>
    );
  }
  if (entityType === "space") {
    return (
      <SpaceLink
        to="/spaces/$spaceSlug"
        params={{ spaceSlug: entityId }}
        className="text-primary font-mono hover:underline"
      >
        {label}
      </SpaceLink>
    );
  }
  return <span className="font-mono">{label}</span>;
}

function DetailRow({ field, from, to }: { field: string; from: string | null; to: string | null }) {
  const label = humanizeField(field);
  if (from === null && to !== null) {
    return (
      <li>
        <span className="text-base-content/70">set {label}: </span>
        <span className="text-base-content">{to}</span>
      </li>
    );
  }
  if (to === null) {
    return (
      <li>
        <span className="text-base-content/70">cleared {label}</span>
        {from !== null && (
          <>
            <span className="text-base-content/70"> (was </span>
            <span className="text-base-content">{from}</span>
            <span className="text-base-content/70">)</span>
          </>
        )}
      </li>
    );
  }
  return (
    <li>
      <span className="text-base-content/70">{label}: </span>
      <span className="line-through text-base-content/60">{from}</span>
      <span className="text-base-content/70"> → </span>
      <span className="text-base-content">{to}</span>
    </li>
  );
}

function ActivityEntry({
  entry,
  memberMap,
  showSpace,
}: {
  entry: ActivityLogEntry;
  memberMap?: Map<string, ActivityMember> | undefined;
  showSpace?: boolean | undefined;
}) {
  const actor = actorLabel(entry, memberMap);
  const actionLabel = ACTION_LABEL[entry.action] ?? entry.action;

  return (
    <div className="flex gap-3 px-4 py-3">
      <div className="mt-0.5 shrink-0">
        <span className="bg-base-200 flex size-7 items-center justify-center rounded-full">
          <UserRound className="text-base-content/70 size-4" aria-hidden="true" />
        </span>
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-baseline gap-x-2">
          <span className="text-base-content text-sm font-medium">{actor}</span>
          <span className="text-base-content/60 text-xs">
            {formatRelativeTime(entry.createdAt)}
          </span>
        </div>
        <p className="text-base-content/70 mt-0.5 text-sm">
          {actionLabel}{" "}
          <EntityRef
            entityType={entry.entityType}
            entityId={entry.entityId}
            spaceSlug={entry.spaceSlug}
          />
          {showSpace && (
            <span className="text-base-content/60 ml-1 text-xs">in {entry.spaceSlug}</span>
          )}
        </p>
        {entry.details.length > 0 && (
          <ul className="text-base-content/70 mt-1 list-none space-y-0.5 text-xs">
            {entry.details.map((d) => (
              <DetailRow key={d.field} field={d.field} from={d.from} to={d.to} />
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

// ── Main export ─────────────────────────────────────────────────────────────

export function ActivityFeed({
  entries,
  hasNextPage,
  fetchNextPage,
  isFetchingNextPage,
  memberMap,
  showSpace,
}: ActivityFeedProps) {
  if (entries.length === 0 && !hasNextPage) {
    return <div className="text-base-content/60 py-8 text-center text-sm">No activity yet.</div>;
  }

  return (
    <div>
      <Card className="divide-base-300 divide-y overflow-hidden">
        {entries.map((entry) => (
          <ActivityEntry key={entry.id} entry={entry} memberMap={memberMap} showSpace={showSpace} />
        ))}
      </Card>

      {hasNextPage && (
        <div className="mt-4 flex justify-center">
          <button
            className="btn btn-soft flex items-center gap-2 disabled:cursor-not-allowed disabled:opacity-50"
            onClick={() => fetchNextPage()}
            disabled={isFetchingNextPage}
          >
            {isFetchingNextPage ? (
              "Loading..."
            ) : (
              <>
                <ChevronDown className="size-4" aria-hidden="true" />
                Load more
              </>
            )}
          </button>
        </div>
      )}
    </div>
  );
}
