import { createLink } from "@tanstack/react-router";
import { ChevronDown, UserRound } from "lucide-react";
import { useLayoutEffect, useRef, useState } from "react";
import type { components } from "@horologia/client-core/schema";
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
  /** A denser presentation, optionally contextualized to an entity detail page. */
  variant?: "default" | "compact" | "task" | "recipe";
}

interface CompactActivityGroup {
  entry: ActivityLogEntry;
  count: number;
}

const COMPACT_GROUP_WINDOW_MS = 5 * 60 * 1000;

function isOpaqueUpdate(detail: ActivityLogEntry["details"][number]): boolean {
  return detail.from === null && detail.to === "updated";
}

function canGroupCompactEntries(newer: ActivityLogEntry, older: ActivityLogEntry): boolean {
  const olderDetails = new Map(older.details.map((detail) => [detail.field, detail]));
  return (
    newer.action === "updated" &&
    older.action === "updated" &&
    newer.details.length > 0 &&
    older.details.length > 0 &&
    newer.actorId === older.actorId &&
    newer.tokenId === older.tokenId &&
    newer.entityType === older.entityType &&
    newer.entityId === older.entityId &&
    newer.spaceSlug === older.spaceSlug &&
    newer.details.every((detail) => {
      const olderDetail = olderDetails.get(detail.field);
      return (
        !olderDetail ||
        olderDetail.to === detail.from ||
        (isOpaqueUpdate(olderDetail) && isOpaqueUpdate(detail))
      );
    }) &&
    new Date(newer.createdAt).getTime() - new Date(older.createdAt).getTime() <=
      COMPACT_GROUP_WINDOW_MS
  );
}

/** Collapse adjacent autosave-like updates while retaining the full before/after range. */
export function groupCompactActivityEntries(entries: ActivityLogEntry[]): CompactActivityGroup[] {
  const groups: CompactActivityGroup[] = [];
  for (const entry of entries) {
    const group = groups.at(-1);
    if (!group || !canGroupCompactEntries(group.entry, entry)) {
      groups.push({ entry, count: 1 });
      continue;
    }

    const olderDetails = new Map(entry.details.map((detail) => [detail.field, detail]));
    const newerFields = new Set(group.entry.details.map((detail) => detail.field));
    group.entry = {
      ...group.entry,
      details: [
        ...group.entry.details.map((detail) => {
          const olderDetail = olderDetails.get(detail.field);
          return { ...detail, from: olderDetail ? olderDetail.from : detail.from };
        }),
        ...entry.details.filter((detail) => !newerFields.has(detail.field)),
      ],
    };
    group.count += 1;
  }
  return groups;
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
  recipe: "recipe",
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

function displayDetailValue(value: string): string {
  return value.replaceAll("&nbsp;", " ").replaceAll("\u00a0", " ");
}

// ── Sub-components ──────────────────────────────────────────────────────────

const TaskLink = createLink("a");
const RecipeLink = createLink("a");
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
  if (entityType === "recipe") {
    return (
      <RecipeLink
        to="/spaces/$spaceSlug/recipes/$recipeId"
        params={{ spaceSlug, recipeId: entityId }}
        className="text-primary font-mono hover:underline"
      >
        {label}
      </RecipeLink>
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
  return <span>{label}</span>;
}

function DetailRow({ field, from, to }: { field: string; from: string | null; to: string | null }) {
  const label = humanizeField(field);
  if (from === null && to !== null) {
    return (
      <li>
        <span className="text-base-content/70">set {label}: </span>
        <span className="text-base-content">{displayDetailValue(to)}</span>
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
            <span className="text-base-content">{displayDetailValue(from)}</span>
            <span className="text-base-content/70">)</span>
          </>
        )}
      </li>
    );
  }
  return (
    <li>
      <span className="text-base-content/70">{label}: </span>
      <span className="line-through text-base-content/60">
        {from === null ? null : displayDetailValue(from)}
      </span>
      <span className="text-base-content/70"> → </span>
      <span className="text-base-content">{displayDetailValue(to)}</span>
    </li>
  );
}

function CompactDetails({ details }: { details: ActivityLogEntry["details"] }) {
  const [expanded, setExpanded] = useState(false);
  const [canExpand, setCanExpand] = useState(false);
  const contentRef = useRef<HTMLUListElement>(null);

  useLayoutEffect(() => {
    if (expanded) return;
    const content = contentRef.current;
    if (!content) return;
    const measure = () => setCanExpand(content.scrollHeight > content.clientHeight + 1);
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(content);
    return () => observer.disconnect();
  }, [details, expanded]);

  return (
    <div>
      <ul
        ref={contentRef}
        className={`text-base-content/70 mt-0.5 list-none space-y-0.5 text-xs ${
          !expanded ? "line-clamp-2" : ""
        }`}
      >
        {details.map((detail) => (
          <DetailRow key={detail.field} {...detail} />
        ))}
      </ul>
      {canExpand && (
        <button
          type="button"
          className="text-primary mt-0.5 text-xs hover:underline"
          onClick={() => setExpanded((value) => !value)}
        >
          {expanded ? "Show less" : "Show more"}
        </button>
      )}
    </div>
  );
}

function compactActionLabel(
  entry: ActivityLogEntry,
  detailContext: "task" | "recipe" | null,
): string {
  if (entry.action === "updated" && entry.details.length > 0) {
    const fields = entry.details.map((detail) => humanizeField(detail.field));
    if (fields.length === 1) return `updated ${fields[0]}`;
    if (fields.length === 2) return `updated ${fields[0]} and ${fields[1]}`;
    return `updated ${fields.slice(0, -1).join(", ")}, and ${fields.at(-1)}`;
  }
  if (entry.action === "created") {
    return detailContext ? `created this ${detailContext}` : "created";
  }
  if (entry.action === "deleted") {
    return detailContext ? `deleted this ${detailContext}` : "deleted";
  }
  return ACTION_LABEL[entry.action] ?? entry.action;
}

function CompactActivityEntry({
  group,
  memberMap,
  detailContext,
  showSpace,
}: {
  group: CompactActivityGroup;
  memberMap?: Map<string, ActivityMember> | undefined;
  detailContext: "task" | "recipe" | null;
  showSpace?: boolean | undefined;
}) {
  const { entry } = group;
  return (
    <div className="flex gap-3 py-2.5">
      <div className="mt-0.5 shrink-0">
        <span className="bg-base-200 flex size-7 items-center justify-center rounded-full">
          <UserRound className="text-base-content/70 size-4" aria-hidden="true" />
        </span>
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 items-baseline gap-x-1.5 text-sm">
          <span className="text-base-content shrink-0 font-medium">
            {actorLabel(entry, memberMap)}
          </span>
          <span className="text-base-content/70 min-w-0 truncate">
            {compactActionLabel(entry, detailContext)}
            {!detailContext && (
              <>
                {entry.action === "updated" ? " on " : " "}
                <EntityRef
                  entityType={entry.entityType}
                  entityId={entry.entityId}
                  spaceSlug={entry.spaceSlug}
                />
              </>
            )}
            {showSpace && (
              <span className="text-base-content/60">
                {" in "}
                <SpaceLink
                  to="/spaces/$spaceSlug"
                  params={{ spaceSlug: entry.spaceSlug }}
                  className="text-primary hover:underline"
                >
                  {entry.spaceSlug}
                </SpaceLink>
              </span>
            )}
          </span>
          <span
            className="text-base-content/50 shrink-0 text-xs"
            title={new Date(entry.createdAt).toLocaleString()}
          >
            {formatRelativeTime(entry.createdAt)}
          </span>
        </div>
        {entry.details.length > 0 && <CompactDetails details={entry.details} />}
      </div>
    </div>
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
  variant = "default",
}: ActivityFeedProps) {
  if (entries.length === 0 && !hasNextPage) {
    return <div className="text-base-content/60 py-8 text-center text-sm">No activity yet.</div>;
  }

  if (variant !== "default") {
    const groups = groupCompactActivityEntries(entries);
    return (
      <div>
        <div className="divide-base-300/70 divide-y">
          {groups.map((group) => (
            <CompactActivityEntry
              key={group.entry.id}
              group={group}
              memberMap={memberMap}
              detailContext={variant === "task" || variant === "recipe" ? variant : null}
              showSpace={showSpace}
            />
          ))}
        </div>
        {hasNextPage && (
          <div className="mt-3 flex justify-center">
            <button
              className="btn btn-ghost btn-sm"
              onClick={() => fetchNextPage()}
              disabled={isFetchingNextPage}
            >
              {isFetchingNextPage ? "Loading..." : "Load more"}
            </button>
          </div>
        )}
      </div>
    );
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
