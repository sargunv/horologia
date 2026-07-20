import { useLocalSearchParams } from "expo-router";

import { SpaceTasksScreen } from "@/components/library-screens";

export default function TasksScreen() {
  const { spaceSlug } = useLocalSearchParams<{ spaceSlug: string }>();
  return <SpaceTasksScreen spaceSlug={spaceSlug} />;
}
