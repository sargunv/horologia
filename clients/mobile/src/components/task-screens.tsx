import { useQuery } from "@tanstack/react-query";
import { Stack, useFocusEffect, useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";

import { TaskDetailView, TaskListView } from "@/components/native-views";
import { errorMessage } from "@/components/read-model";
import type { Task } from "@/components/native-views.types";
import { useListDetailGeometry } from "@/layout/window-class";
import { routes } from "@/navigation/routes";
import { useAppRuntime } from "@/runtime/app-runtime";
import { useMyTasks } from "@/runtime/use-my-tasks";

export function MyTasksScreen() {
  const runtime = useAppRuntime();
  const router = useRouter();
  const geometry = useListDetailGeometry();
  const model = useMyTasks();
  const { refresh } = model;
  const [selected, setSelected] = useState<Task | null>(null);

  useFocusEffect(
    useCallback(() => {
      void refresh();
    }, [refresh]),
  );

  useEffect(() => {
    if (geometry.presentation !== "list-detail") return;
    setSelected((current) => {
      if (
        current &&
        model.tasks.some((task) => task.id === current.id && task.spaceSlug === current.spaceSlug)
      )
        return current;
      return model.tasks[0] ?? null;
    });
  }, [geometry.presentation, model.tasks]);

  function openTask(task: Task) {
    if (geometry.presentation === "list-detail") {
      setSelected(task);
      return;
    }
    router.push(routes.taskDetail(runtime.scope, task.spaceSlug, task.id));
  }

  const list = (
    <TaskListView
      {...model}
      selectedTaskId={selected?.id}
      onSelect={openTask}
      onRefresh={model.refresh}
      onLoadMore={() => void model.loadMore()}
      onRetry={() => void model.retry()}
    />
  );

  if (geometry.presentation === "single-pane") return list;
  return (
    <View style={styles.panes}>
      <View style={[styles.listPane, { width: geometry.listPaneWidth }]}>{list}</View>
      <View style={styles.detailPane}>
        {selected ? (
          <TaskDetailController spaceSlug={selected.spaceSlug} taskId={selected.id} />
        ) : (
          <TaskDetailView
            task={null}
            isLoading={false}
            error={null}
            emptyTitle="Select a task"
            emptyDetail="Choose a task from the list to view its details."
            onRetry={() => undefined}
          />
        )}
      </View>
    </View>
  );
}

export function TaskDetailController({
  spaceSlug,
  taskId,
  controlsHeader = false,
}: {
  spaceSlug: string;
  taskId: string;
  controlsHeader?: boolean;
}) {
  const runtime = useAppRuntime();
  const query = useQuery(runtime.queries.spaceTaskQueryOptions(spaceSlug, taskId));
  return (
    <>
      {controlsHeader ? (
        <Stack.Screen
          options={{
            title: query.data?.title ?? "",
            headerLargeTitle: true,
          }}
        />
      ) : null}
      <TaskDetailView
        task={query.data ?? null}
        isLoading={query.isLoading}
        error={query.error ? errorMessage(query.error, "Could not load task.") : null}
        showTitle={!controlsHeader}
        onRetry={() => {
          void query.refetch();
        }}
      />
    </>
  );
}

const styles = StyleSheet.create({
  panes: { flex: 1, flexDirection: "row" },
  listPane: {},
  detailPane: { flex: 1 },
});
