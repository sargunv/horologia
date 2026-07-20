import { useLocalSearchParams } from "expo-router";

import { TaskDetailController } from "@/components/task-screens";

export default function MyTaskDetailScreen() {
  const { spaceSlug, taskId } = useLocalSearchParams<{ spaceSlug: string; taskId: string }>();
  return <TaskDetailController spaceSlug={spaceSlug} taskId={taskId} controlsHeader />;
}
