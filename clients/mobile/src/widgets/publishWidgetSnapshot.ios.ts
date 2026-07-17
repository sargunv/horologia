import type { WidgetSnapshotV1 } from "@horologia/client-core";

import MyTasksWidget from "../../widgets/MyTasksWidget";

export async function publishWidgetSnapshot(snapshot: WidgetSnapshotV1): Promise<void> {
  const nextTask = snapshot.tasks[0];
  MyTasksWidget.updateSnapshot({
    count: snapshot.tasks.length,
    generatedAt: new Date(snapshot.generatedAt).toLocaleTimeString(undefined, {
      hour: "numeric",
      minute: "2-digit",
    }),
    nextTaskId: nextTask?.id ?? "",
    nextTaskTitle: nextTask?.title ?? "You're all caught up",
    secondTaskTitle: snapshot.tasks[1]?.title ?? "",
    signedIn: true,
    spaceSlug: nextTask?.spaceSlug ?? "",
    thirdTaskTitle: snapshot.tasks[2]?.title ?? "",
  });
}

export async function clearWidgetSnapshot(): Promise<void> {
  MyTasksWidget.updateSnapshot({
    count: 0,
    generatedAt: "",
    nextTaskId: "",
    nextTaskTitle: "Sign in to see your tasks",
    secondTaskTitle: "",
    signedIn: false,
    spaceSlug: "",
    thirdTaskTitle: "",
  });
}
