import { useLocalSearchParams } from "expo-router";

import { TaskDetailController } from "@/components/task-screens";

export default function SpaceTaskDetailScreen() {
  const { spaceSlug, taskId } = useLocalSearchParams<{ spaceSlug: string; taskId: string }>();
  return <TaskDetailController spaceSlug={spaceSlug} taskId={taskId} controlsHeader />;
}
