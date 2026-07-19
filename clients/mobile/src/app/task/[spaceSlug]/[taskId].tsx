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
} from "@tanstack/react-query";
import { Button, ListItem, Text } from "@expo/ui";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Alert, Share } from "react-native";

import { useSession } from "@/auth/session-context";
import { FormField, FormPicker, FormSection } from "@/components/forms";
import { NativeFormScreen } from "@/components/native-screen";
import { ScreenState } from "@/components/screen-state";
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
  const task = query.data;
  const activityEntries = activity.data?.pages.flatMap((page) => page.items) ?? [];
  return (
    <NativeFormScreen>
        <FormSection title={task.title}>
          <Text>{`${task.spaceSlug} · ${task.status}`}</Text>
          <Button
            label="Edit"
            onPress={() =>
              router.push({
                pathname: "/task/[spaceSlug]/[taskId]/edit",
                params: { spaceSlug: props.spaceSlug, taskId: props.taskId },
              })
            }
            variant="filled"
          />
          <Button
            label="Share"
            onPress={() =>
              void Share.share({ title: task.title, message: shareUrl, url: shareUrl })
            }
            variant="filled"
          />
          <Button
            disabled={deleteTask.isPending}
            label={deleteTask.isPending ? "Deleting…" : "Delete"}
            onPress={() =>
              Alert.alert("Delete task?", "This cannot be undone.", [
                { text: "Cancel", style: "cancel" },
                { text: "Delete", style: "destructive", onPress: () => deleteTask.mutate() },
              ])
            }
            variant="text"
          />
        </FormSection>
        <FormSection title="Details">
          {task.description ? <Text>{task.description}</Text> : null}
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
          {task.tags.length ? <Property label="Tags" value={task.tags.join(" · ")} /> : null}
          {task.overdueActionRule ? (
            <Property
              label="When overdue"
              value={`${task.overdueActionRule.action.replaceAll("_", " ")}${task.overdueActionRule.after === null ? " immediately" : ` after ${task.overdueActionRule.after} days`}`}
            />
          ) : null}
          <Property label="Task ID" value={task.id} />
        </FormSection>
        <FormSection title="Relations">
          {task.relations.map((relation) => (
            <ListItem
              key={`${relation.kind}/${relation.relatedTaskId}`}
              supportingText={relation.kind.replaceAll("_", " ")}
              trailing={
                <Button
                  label="Remove"
                  onPress={() =>
                    removeRelation.mutate({
                      kind: relation.kind,
                      relatedTaskId: relation.relatedTaskId,
                    })
                  }
                  variant="text"
                />
              }
            >
              <Text>{relation.relatedTaskId}</Text>
            </ListItem>
          ))}
          <FormPicker
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
          {(search.data ?? []).map((result) => (
            <ListItem
              key={`${result.spaceSlug}/${result.id}`}
              onPress={() => addRelation.mutate(result.id)}
              supportingText={result.spaceSlug}
            >
              <Text>{result.title}</Text>
            </ListItem>
          ))}
          {addRelation.error ? <Text>{addRelation.error.message}</Text> : null}
        </FormSection>
        <FormSection title="Activity">
          {activityEntries.length ? (
            activityEntries.map((entry) => (
              <ListItem
                key={entry.id}
                supportingText={new Date(entry.createdAt).toLocaleString()}
              >
                <Text>{`${entry.action} ${entry.entityType}`}</Text>
              </ListItem>
            ))
          ) : (
            <Text>No activity yet.</Text>
          )}
        </FormSection>
        {notice ? <Text>{notice}</Text> : null}
    </NativeFormScreen>
  );
}

function Property({ label, value }: { label: string; value: string }) {
  return (
    <ListItem supportingText={value}>
      <Text>{label}</Text>
    </ListItem>
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
