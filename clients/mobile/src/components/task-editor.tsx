import {
  buildRRule,
  createQueries,
  createTaskCommands,
  describeRule,
  parseRRule,
  taskCreateFromDraft,
  taskDraftFromTask,
  taskRecurrenceUsesRule,
  taskUpdateFromDraft,
  type HorologiaClient,
  type ServerProfile,
  type TaskDraft,
} from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ActionButton, ChoiceChips, FormField, FormSection } from "@/components/forms";
import { colors } from "@/design/tokens";
import { refreshMyTasksWidget } from "@/widgets/refreshTaskWidget";

type Task = components["schemas"]["Task"];

interface TaskEditorProps {
  client: HorologiaClient;
  profile: ServerProfile;
  accountId: string;
  task?: Task;
  initialSpaceSlug?: string;
}

export function TaskEditor({
  accountId,
  client,
  profile,
  task,
  initialSpaceSlug,
}: TaskEditorProps) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const queries = useMemo(
    () => createQueries({ serverId: profile.id, apiClient: client, appClient: client }),
    [client, profile.id],
  );
  const spacesQuery = useQuery(queries.spacesQueryOptions);
  const defaultSpace = task?.spaceSlug ?? initialSpaceSlug ?? spacesQuery.data?.[0]?.slug ?? "";
  const [spaceSlug, setSpaceSlug] = useState(defaultSpace);
  const activeSpace = spaceSlug || defaultSpace;
  const statusesQuery = useQuery({
    ...queries.spaceTaskStatusesQueryOptions(activeSpace),
    enabled: activeSpace.length > 0,
  });
  const effortQuery = useQuery({
    ...queries.spaceEffortLevelsQueryOptions(activeSpace),
    enabled: activeSpace.length > 0,
  });
  const priorityQuery = useQuery({
    ...queries.spacePriorityLevelsQueryOptions(activeSpace),
    enabled: activeSpace.length > 0,
  });
  const [draft, setDraft] = useState(() =>
    taskDraftFromTask(task, Intl.DateTimeFormat().resolvedOptions().timeZone ?? "UTC"),
  );
  const [cacheWarning, setCacheWarning] = useState<string | null>(null);
  const commands = createTaskCommands({
    serverId: profile.id,
    apiClient: client,
    queryClient,
    onCacheError() {
      setCacheWarning("Saved, but some lists may be stale. Pull to refresh.");
    },
  });
  const mutation = useMutation({
    mutationFn: async () => {
      if (!activeSpace) throw new Error("Choose a space");
      const body = taskUpdateFromDraft(draft);
      return task
        ? commands.update(activeSpace, task.id, body)
        : commands.create(activeSpace, taskCreateFromDraft(draft));
    },
    async onSuccess(saved) {
      try {
        await refreshMyTasksWidget({ profile, accountId, client, queryClient });
      } catch {
        setCacheWarning("Task saved. The widget will refresh when My Tasks opens.");
      }
      router.replace({
        pathname: "/task/[spaceSlug]/[taskId]",
        params: { spaceSlug: saved.spaceSlug, taskId: saved.id },
      });
    },
  });
  const recurrence = parseRRule(draft.recurrenceRule || null);

  function set<K extends keyof TaskDraft>(key: K, value: TaskDraft[K]) {
    setDraft((current) => ({ ...current, [key]: value }));
  }

  function chooseFrequency(freq: "DAILY" | "WEEKLY" | "MONTHLY" | "YEARLY") {
    set("recurrenceRule", buildRRule({ ...recurrence, freq, parseError: false }));
  }

  return (
    <SafeAreaView edges={["left", "right", "bottom"]} style={styles.safeArea}>
      <ScrollView
        automaticallyAdjustKeyboardInsets
        contentContainerStyle={styles.scroll}
        keyboardShouldPersistTaps="handled"
      >
        <View>
          <Text accessibilityRole="header" style={styles.heading}>
            {task ? "Edit task" : "New task"}
          </Text>
          <Text style={styles.subheading}>Every task option, in one straightforward form.</Text>
        </View>

        {!task && spacesQuery.data ? (
          <FormSection title="Space">
            <ChoiceChips
              label="Space"
              onChange={setSpaceSlug}
              options={spacesQuery.data.map((space) => ({ label: space.name, value: space.slug }))}
              value={activeSpace}
            />
          </FormSection>
        ) : null}

        <FormSection title="Basics">
          <FormField
            autoCapitalize="sentences"
            label="Title"
            maxLength={500}
            onChangeText={(value) => set("title", value)}
            placeholder="What needs to be done?"
            testID="task-title-input"
            value={draft.title}
          />
          <FormField
            label="Description"
            maxLength={10_000}
            multiline
            onChangeText={(value) => set("description", value)}
            placeholder="Notes, context, or a checklist"
            value={draft.description}
          />
          <ChoiceChips
            label="Status"
            onChange={(value) => set("status", value)}
            options={(statusesQuery.data ?? []).map((status) => ({
              label: status.name,
              value: status.name,
            }))}
            value={draft.status}
          />
        </FormSection>

        <FormSection title="Planning">
          <ChoiceChips
            label="Effort"
            onChange={(value) => set("effort", value)}
            options={[
              { label: "None", value: "" },
              ...(effortQuery.data ?? []).map((level) => ({
                label: level.name,
                value: level.name,
              })),
            ]}
            value={draft.effort}
          />
          <ChoiceChips
            label="Priority"
            onChange={(value) => set("priority", value)}
            options={[
              { label: "None", value: "" },
              ...(priorityQuery.data ?? []).map((level) => ({
                label: level.name,
                value: level.name,
              })),
            ]}
            value={draft.priority}
          />
          <FormField
            autoCapitalize="none"
            label="Tags (comma separated)"
            onChangeText={(value) => set("tags", value)}
            value={draft.tags}
          />
          <FormField
            autoCapitalize="none"
            label="Assignee IDs (comma separated)"
            onChangeText={(value) => set("assigneeIds", value)}
            value={draft.assigneeIds}
          />
          <FormField
            autoCapitalize="none"
            label="Rotation pool IDs (comma separated)"
            onChangeText={(value) => set("rotationPool", value)}
            value={draft.rotationPool}
          />
        </FormSection>

        <FormSection title="Due date">
          <FormField
            autoCapitalize="none"
            label="Date (YYYY-MM-DD)"
            onChangeText={(value) => set("dueDate", value)}
            placeholder="2026-07-18"
            value={draft.dueDate}
          />
          <FormField
            autoCapitalize="none"
            label="Timezone"
            onChangeText={(value) => set("timezone", value)}
            value={draft.timezone}
          />
        </FormSection>

        <FormSection title="Recurrence">
          <ChoiceChips
            label="Behavior"
            onChange={(value) => {
              set("recurrenceType", value);
              if (!taskRecurrenceUsesRule(value)) set("recurrenceRule", "");
            }}
            options={RECURRENCE_TYPES}
            value={draft.recurrenceType}
          />
          {taskRecurrenceUsesRule(draft.recurrenceType) ? (
            <>
              <ChoiceChips
                label="Frequency"
                onChange={chooseFrequency}
                options={FREQUENCIES}
                value={recurrence.freq}
              />
              <FormField
                keyboardType="number-pad"
                label="Every (interval)"
                onChangeText={(value) => {
                  const interval = Math.max(1, Number(value) || 1);
                  set("recurrenceRule", buildRRule({ ...recurrence, interval, parseError: false }));
                }}
                value={String(recurrence.interval)}
              />
              <FormField
                autoCapitalize="characters"
                label="RRULE"
                maxLength={500}
                onChangeText={(value) => set("recurrenceRule", value)}
                placeholder="FREQ=WEEKLY;BYDAY=MO,WE"
                value={draft.recurrenceRule}
              />
              <Text style={recurrence.parseError ? styles.error : styles.hint}>
                {recurrence.parseError ? "This RRULE is not valid." : describeRule(recurrence)}
              </Text>
            </>
          ) : null}
        </FormSection>

        <FormSection title="When overdue">
          <ChoiceChips
            label="Action"
            onChange={(value) => set("overdueAction", value)}
            options={OVERDUE_ACTIONS}
            value={draft.overdueAction}
          />
          {draft.overdueAction ? (
            <>
              <FormField
                keyboardType="number-pad"
                label="Grace period in days (blank is immediate)"
                onChangeText={(value) => set("overdueAfter", value)}
                value={draft.overdueAfter}
              />
              {draft.overdueAction === "set_status" ? (
                <ChoiceChips
                  label="New status"
                  onChange={(value) => set("overdueStatus", value)}
                  options={(statusesQuery.data ?? []).map((status) => ({
                    label: status.name,
                    value: status.name,
                  }))}
                  value={draft.overdueStatus}
                />
              ) : null}
            </>
          ) : null}
        </FormSection>

        {mutation.error ? (
          <Text accessibilityRole="alert" style={styles.error}>
            {mutation.error.message}
          </Text>
        ) : null}
        {cacheWarning ? <Text style={styles.warning}>{cacheWarning}</Text> : null}
        <ActionButton
          disabled={mutation.isPending || !draft.title.trim() || !activeSpace}
          label={mutation.isPending ? "Saving…" : task ? "Save task" : "Create task"}
          onPress={() => mutation.mutate()}
        />
      </ScrollView>
    </SafeAreaView>
  );
}

const RECURRENCE_TYPES = [
  { value: "one_off", label: "One-off" },
  { value: "completion_based", label: "On completion" },
  { value: "fixed_non_accumulating", label: "Fixed" },
  { value: "fixed_accumulating", label: "Accumulating" },
  { value: "on_dependency", label: "On dependency" },
] as const;

const FREQUENCIES = [
  { value: "DAILY", label: "Daily" },
  { value: "WEEKLY", label: "Weekly" },
  { value: "MONTHLY", label: "Monthly" },
  { value: "YEARLY", label: "Yearly" },
] as const;

const OVERDUE_ACTIONS = [
  { value: "", label: "None" },
  { value: "advance_recurrence", label: "Advance" },
  { value: "set_status", label: "Set status" },
  { value: "clear_due_date", label: "Clear due" },
] as const;

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  scroll: {
    alignSelf: "center",
    gap: 14,
    maxWidth: 760,
    padding: 18,
    paddingBottom: 50,
    width: "100%",
  },
  heading: { color: colors.ink, fontSize: 30, fontWeight: "700", letterSpacing: -0.6 },
  subheading: { color: colors.muted, fontSize: 14, marginTop: 4 },
  hint: { color: colors.muted, fontSize: 13, lineHeight: 19 },
  error: { color: colors.danger, fontSize: 14, fontWeight: "600", lineHeight: 20 },
  warning: { color: colors.accent, fontSize: 14, fontWeight: "600" },
});
