import { createQueries, type HorologiaClient, type ServerProfile } from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";
import { useQuery } from "@tanstack/react-query";
import { useLocalSearchParams } from "expo-router";
import { useMemo } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";
import { colors } from "@/design/tokens";

type Task = components["schemas"]["Task"];

export default function TaskDetailScreen() {
  const { spaceSlug, taskId } = useLocalSearchParams<{ spaceSlug: string; taskId: string }>();
  const session = useSession();
  if (!session.profile || !session.client || !spaceSlug || !taskId) {
    return <ScreenState loading title="Opening task" />;
  }
  return (
    <TaskDetail
      client={session.client}
      profile={session.profile}
      spaceSlug={spaceSlug}
      taskId={taskId}
    />
  );
}

function TaskDetail(props: {
  client: HorologiaClient;
  profile: ServerProfile;
  spaceSlug: string;
  taskId: string;
}) {
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
  return <TaskReadView task={query.data} />;
}

function TaskReadView({ task }: { task: Task }) {
  return (
    <SafeAreaView edges={["left", "right", "bottom"]} style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.hero}>
          <Text style={styles.space}>{task.spaceSlug.toUpperCase()}</Text>
          <Text accessibilityRole="header" style={styles.title}>
            {task.title}
          </Text>
          <Text style={styles.status}>{task.status}</Text>
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
        {task.relations.length ? (
          <Section label="Relations" value={`${task.relations.length} related tasks`} />
        ) : null}
        {task.overdueActionRule ? (
          <Section
            label="When overdue"
            value={`${task.overdueActionRule.action.replaceAll("_", " ")}${task.overdueActionRule.after === null ? " immediately" : ` after ${task.overdueActionRule.after} days`}`}
          />
        ) : null}
        <Text selectable style={styles.identifier}>
          Task ID · {task.id}
        </Text>
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
  identifier: { color: colors.muted, fontSize: 12, marginTop: 18 },
});
