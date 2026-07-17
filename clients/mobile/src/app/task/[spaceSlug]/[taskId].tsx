import {
  createQueries,
  createTaskCommands,
  type HorologiaClient,
  type ServerProfile,
} from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";
import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
} from "@tanstack/react-query";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Alert, Pressable, ScrollView, Share, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { ChoiceChips, FormField } from "@/components/forms";
import { ScreenState } from "@/components/screen-state";
import { colors } from "@/design/tokens";
import { refreshMyTasksWidget } from "@/widgets/refreshTaskWidget";

type Task = components["schemas"]["Task"];
type TaskRelationKind = components["schemas"]["TaskRelationKind"];

export default function TaskDetailScreen() {
  const { spaceSlug, taskId } = useLocalSearchParams<{ spaceSlug: string; taskId: string }>();
  const session = useSession();
  if (!session.profile || !session.accountId || !session.client || !spaceSlug || !taskId) {
    return <ScreenState loading title="Opening task" />;
  }
  return (
    <TaskDetail
      client={session.client}
      accountId={session.accountId}
      profile={session.profile}
      spaceSlug={spaceSlug}
      taskId={taskId}
    />
  );
}

function TaskDetail(props: {
  accountId: string;
  client: HorologiaClient;
  profile: ServerProfile;
  spaceSlug: string;
  taskId: string;
}) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [relationSearch, setRelationSearch] = useState("");
  const [relationKind, setRelationKind] = useState<TaskRelationKind>("relates_to");
  const [notice, setNotice] = useState<string | null>(null);
  const queries = useMemo(
    () =>
      createQueries({
        serverId: props.profile.id,
        apiClient: props.client,
        appClient: props.client,
      }),
    [props.client, props.profile.id],
  );
  const query = useQuery(queries.spaceTaskQueryOptions(props.spaceSlug, props.taskId));
  const activity = useInfiniteQuery(
    queries.taskActivityInfiniteQueryOptions(props.spaceSlug, props.taskId),
  );
  const search = useQuery({
    ...queries.taskSearchQueryOptions({
      query: relationSearch,
      excludeTaskId: props.taskId,
      limit: 8,
    }),
    enabled: relationSearch.trim().length >= 2,
  });
  const commands = createTaskCommands({
    serverId: props.profile.id,
    apiClient: props.client,
    queryClient,
    onCacheError() {
      setNotice("Saved, but cached task lists may need a refresh.");
    },
  });
  const addRelation = useMutation({
    mutationFn: (relatedTaskId: string) =>
      commands.addRelation(props.spaceSlug, props.taskId, {
        kind: relationKind,
        relatedTaskId,
      }),
    onSuccess() {
      setRelationSearch("");
    },
  });
  const removeRelation = useMutation({
    mutationFn: ({ kind, relatedTaskId }: { kind: TaskRelationKind; relatedTaskId: string }) =>
      commands.deleteRelation(props.spaceSlug, props.taskId, kind, relatedTaskId),
  });
  const deleteTask = useMutation({
    mutationFn: () => commands.delete(props.spaceSlug, props.taskId),
    async onSuccess() {
      try {
        await refreshMyTasksWidget({
          profile: props.profile,
          accountId: props.accountId,
          client: props.client,
          queryClient,
        });
      } catch {
        setNotice("Task deleted. The widget will refresh when My Tasks opens.");
      }
      router.replace("/(tabs)/tasks");
    },
  });
  const shareUrl = new URL(
    `spaces/${encodeURIComponent(props.spaceSlug)}/tasks/${encodeURIComponent(props.taskId)}`,
    `${props.profile.baseUrl.replace(/\/+$/u, "")}/`,
  ).toString();
  if (query.isPending) return <ScreenState loading title="Loading task" />;
  if (query.isError) {
    return (
      <ScreenState
        actionLabel="Try again"
        detail={query.error.message}
        onAction={() => void query.refetch()}
        title="Task couldn't load"
      />
    );
  }
  return (
    <TaskReadView
      activity={activity.data?.pages.flatMap((page) => page.items) ?? []}
      addRelation={addRelation}
      deleteTask={deleteTask}
      notice={notice}
      onDelete={() =>
        Alert.alert("Delete task?", "This cannot be undone.", [
          { text: "Cancel", style: "cancel" },
          { text: "Delete", style: "destructive", onPress: () => deleteTask.mutate() },
        ])
      }
      onEdit={() =>
        router.push({
          pathname: "/task/[spaceSlug]/[taskId]/edit",
          params: { spaceSlug: props.spaceSlug, taskId: props.taskId },
        })
      }
      onShare={() =>
        void Share.share({ title: query.data.title, message: shareUrl, url: shareUrl })
      }
      relationKind={relationKind}
      relationSearch={relationSearch}
      removeRelation={removeRelation}
      searchResults={search.data ?? []}
      setRelationKind={setRelationKind}
      setRelationSearch={setRelationSearch}
      task={query.data}
    />
  );
}

function TaskReadView({
  task,
  activity,
  relationSearch,
  relationKind,
  searchResults,
  setRelationSearch,
  setRelationKind,
  addRelation,
  removeRelation,
  deleteTask,
  notice,
  onEdit,
  onShare,
  onDelete,
}: {
  task: Task;
  activity: components["schemas"]["ActivityLogEntry"][];
  relationSearch: string;
  relationKind: TaskRelationKind;
  searchResults: components["schemas"]["TaskSearchResult"][];
  setRelationSearch: (value: string) => void;
  setRelationKind: (value: TaskRelationKind) => void;
  addRelation: UseMutationResult<
    components["schemas"]["TaskRelation"],
    Error,
    string,
    unknown
  >;
  removeRelation: UseMutationResult<
    void,
    Error,
    { kind: TaskRelationKind; relatedTaskId: string },
    unknown
  >;
  deleteTask: UseMutationResult<void, Error, void, unknown>;
  notice: string | null;
  onEdit: () => void;
  onShare: () => void;
  onDelete: () => void;
}) {
  return (
    <SafeAreaView edges={["left", "right", "bottom"]} style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.hero}>
          <Text style={styles.space}>{task.spaceSlug.toUpperCase()}</Text>
          <Text accessibilityRole="header" style={styles.title}>
            {task.title}
          </Text>
          <Text style={styles.status}>{task.status}</Text>
          <View style={styles.actions}>
            <Pressable accessibilityRole="button" onPress={onEdit} style={styles.secondaryButton}>
              <Text style={styles.secondaryButtonText}>Edit</Text>
            </Pressable>
            <Pressable accessibilityRole="button" onPress={onShare} style={styles.linkButton}>
              <Text style={styles.linkButtonText}>Share</Text>
            </Pressable>
            <Pressable
              accessibilityRole="button"
              disabled={deleteTask.isPending}
              onPress={onDelete}
              style={styles.deleteButton}
            >
              <Text style={styles.deleteButtonText}>Delete</Text>
            </Pressable>
          </View>
        </View>
        {task.description ? <Section label="Description" value={task.description} /> : null}
        <View style={styles.grid}>
          <Property label="Due" value={task.due?.at ?? "No due date"} />
          <Property label="Priority" value={task.priority ?? "None"} />
          <Property label="Effort" value={task.effort ?? "None"} />
          <Property label="Recurrence" value={formatRecurrence(task)} />
          <Property
            label="Assignees"
            value={task.assigneeIds.length ? `${task.assigneeIds.length}` : "None"}
          />
          <Property
            label="Rotation"
            value={task.rotationPool.length ? `${task.rotationPool.length} people` : "None"}
          />
        </View>
        {task.tags.length ? <Section label="Tags" value={task.tags.join(" · ")} /> : null}
        <View style={styles.section}>
          <Text style={styles.label}>Relations</Text>
          {task.relations.map((relation) => (
            <View key={`${relation.kind}/${relation.relatedTaskId}`} style={styles.relationRow}>
              <Text style={styles.relationText}>
                {relation.kind.replaceAll("_", " ")} · {relation.relatedTaskId}
              </Text>
              <Pressable
                accessibilityLabel={`Remove relation to ${relation.relatedTaskId}`}
                accessibilityRole="button"
                onPress={() =>
                  removeRelation.mutate({
                    kind: relation.kind,
                    relatedTaskId: relation.relatedTaskId,
                  })
                }
              >
                <Text style={styles.removeText}>Remove</Text>
              </Pressable>
            </View>
          ))}
          <ChoiceChips
            label="Relation type"
            onChange={setRelationKind}
            options={RELATION_KINDS}
            value={relationKind}
          />
          <FormField
            label="Find a related task"
            onChangeText={setRelationSearch}
            placeholder="Search across spaces"
            value={relationSearch}
          />
          {searchResults.map((result) => (
            <Pressable
              accessibilityRole="button"
              key={`${result.spaceSlug}/${result.id}`}
              onPress={() => addRelation.mutate(result.id)}
              style={styles.searchResult}
            >
              <Text style={styles.searchResultTitle}>{result.title}</Text>
              <Text style={styles.searchResultMeta}>{result.spaceSlug}</Text>
            </Pressable>
          ))}
          {addRelation.error ? <Text style={styles.error}>{addRelation.error.message}</Text> : null}
        </View>
        {task.overdueActionRule ? (
          <Section
            label="When overdue"
            value={`${task.overdueActionRule.action.replaceAll("_", " ")}${task.overdueActionRule.after === null ? " immediately" : ` after ${task.overdueActionRule.after} days`}`}
          />
        ) : null}
        <Text selectable style={styles.identifier}>
          Task ID · {task.id}
        </Text>
        <View style={styles.section}>
          <Text style={styles.label}>Activity</Text>
          {activity.length ? (
            activity.map((entry) => (
              <View key={entry.id} style={styles.activityRow}>
                <Text style={styles.sectionValue}>
                  {entry.action} {entry.entityType}
                </Text>
                <Text style={styles.activityTime}>{new Date(entry.createdAt).toLocaleString()}</Text>
              </View>
            ))
          ) : (
            <Text style={styles.sectionValue}>No activity yet.</Text>
          )}
        </View>
        {notice ? <Text style={styles.notice}>{notice}</Text> : null}
      </ScrollView>
    </SafeAreaView>
  );
}

function Property({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.property}>
      <Text style={styles.label}>{label}</Text>
      <Text style={styles.value}>{value}</Text>
    </View>
  );
}

function Section({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.section}>
      <Text style={styles.label}>{label}</Text>
      <Text style={styles.sectionValue}>{value}</Text>
    </View>
  );
}

function formatRecurrence(task: Task): string {
  if (task.recurrenceType === "one_off") return "One-off";
  return task.recurrenceRule ?? task.recurrenceType.replaceAll("_", " ");
}

const RELATION_KINDS = [
  { value: "relates_to", label: "Related" },
  { value: "blocks", label: "Blocks" },
  { value: "blocked_by", label: "Blocked by" },
  { value: "parent_of", label: "Parent" },
  { value: "child_of", label: "Child" },
  { value: "duplicates", label: "Duplicates" },
  { value: "triggers", label: "Triggers" },
  { value: "spawns", label: "Spawns" },
] as const;

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  scroll: { alignSelf: "center", maxWidth: 760, padding: 20, paddingBottom: 48, width: "100%" },
  hero: { marginBottom: 22 },
  space: { color: colors.accent, fontSize: 12, fontWeight: "800", letterSpacing: 1.3 },
  title: {
    color: colors.ink,
    fontSize: 32,
    fontWeight: "700",
    letterSpacing: -0.7,
    lineHeight: 38,
    marginTop: 7,
  },
  status: {
    alignSelf: "flex-start",
    backgroundColor: colors.accentSoft,
    borderRadius: 999,
    color: colors.accent,
    fontSize: 13,
    fontWeight: "700",
    marginTop: 13,
    overflow: "hidden",
    paddingHorizontal: 11,
    paddingVertical: 6,
  },
  actions: { flexDirection: "row", gap: 9, marginTop: 15 },
  secondaryButton: {
    backgroundColor: colors.accent,
    borderRadius: 12,
    paddingHorizontal: 17,
    paddingVertical: 10,
  },
  secondaryButtonText: { color: colors.surface, fontSize: 14, fontWeight: "700" },
  linkButton: { borderRadius: 12, paddingHorizontal: 14, paddingVertical: 10 },
  linkButtonText: { color: colors.accent, fontSize: 14, fontWeight: "700" },
  deleteButton: { borderRadius: 12, paddingHorizontal: 14, paddingVertical: 10 },
  deleteButtonText: { color: colors.danger, fontSize: 14, fontWeight: "700" },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 10, marginBottom: 10 },
  property: {
    backgroundColor: colors.surface,
    borderRadius: 15,
    minWidth: 145,
    padding: 15,
    width: "48%",
  },
  label: {
    color: colors.muted,
    fontSize: 12,
    fontWeight: "700",
    letterSpacing: 0.7,
    textTransform: "uppercase",
  },
  value: {
    color: colors.ink,
    fontSize: 16,
    fontWeight: "600",
    marginTop: 6,
    textTransform: "capitalize",
  },
  section: { backgroundColor: colors.surface, borderRadius: 15, marginBottom: 10, padding: 17 },
  sectionValue: { color: colors.ink, fontSize: 16, lineHeight: 24, marginTop: 8 },
  relationRow: { alignItems: "center", flexDirection: "row", gap: 10, paddingVertical: 8 },
  relationText: { color: colors.ink, flex: 1, fontSize: 14 },
  removeText: { color: colors.danger, fontSize: 13, fontWeight: "700" },
  searchResult: {
    backgroundColor: colors.canvas,
    borderRadius: 11,
    paddingHorizontal: 12,
    paddingVertical: 10,
  },
  searchResultTitle: { color: colors.ink, fontSize: 15, fontWeight: "600" },
  searchResultMeta: { color: colors.muted, fontSize: 12, marginTop: 2 },
  activityRow: { borderBottomColor: colors.outline, borderBottomWidth: StyleSheet.hairlineWidth, paddingVertical: 6 },
  activityTime: { color: colors.muted, fontSize: 12, marginTop: 2 },
  error: { color: colors.danger, fontSize: 13, fontWeight: "600" },
  notice: { color: colors.accent, fontSize: 13, fontWeight: "600" },
  identifier: { color: colors.muted, fontSize: 12, marginTop: 18 },
});
