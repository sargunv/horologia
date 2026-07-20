import { useLocalSearchParams } from "expo-router";

import { SpaceWorkspaceScreen } from "@/components/library-screens";

export default function SpaceScreen() {
  const { spaceSlug } = useLocalSearchParams<{ spaceSlug: string }>();
  return <SpaceWorkspaceScreen spaceSlug={spaceSlug} />;
}
