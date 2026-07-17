import type { WidgetSnapshotV1 } from "@horologia/client-core";

import MyTasksWidget from "../../widgets/MyTasksWidget";

export async function publishWidgetSnapshot(snapshot: WidgetSnapshotV1): Promise<void> {
  const nextTask = snapshot.tasks[0];
  MyTasksWidget.updateSnapshot({
    count: snapshot.tasks.length,
    nextTaskId: nextTask?.id ?? "",
    nextTaskTitle: nextTask?.title ?? "You're all caught up",
    spaceSlug: nextTask?.spaceSlug ?? "",
  });
}
