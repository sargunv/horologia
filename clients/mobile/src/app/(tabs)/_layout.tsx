import { Redirect, useSegments } from "expo-router";
import { NativeTabs } from "expo-router/unstable-native-tabs";
import { View } from "react-native";

import { useSession } from "@/auth/session-context";

export default function TabLayout() {
  const session = useSession();
  const segments = useSegments();
  if (session.status !== "signed-in") return <Redirect href="/" />;
  const isTabRoute = segments[0] === "(tabs)";

  return (
    <View
      accessibilityElementsHidden={!isTabRoute}
      importantForAccessibility={isTabRoute ? "auto" : "no-hide-descendants"}
      style={{ flex: 1 }}
    >
      <NativeTabs>
        <NativeTabs.Trigger name="tasks">
          <NativeTabs.Trigger.Label>Tasks</NativeTabs.Trigger.Label>
          <NativeTabs.Trigger.Icon sf="checkmark.circle" md="check_circle" />
        </NativeTabs.Trigger>
        <NativeTabs.Trigger name="library">
          <NativeTabs.Trigger.Label>Library</NativeTabs.Trigger.Label>
          <NativeTabs.Trigger.Icon sf="square.grid.2x2" md="grid_view" />
        </NativeTabs.Trigger>
        <NativeTabs.Trigger name="search" role="search">
          <NativeTabs.Trigger.Label>Search</NativeTabs.Trigger.Label>
          <NativeTabs.Trigger.Icon sf="magnifyingglass" md="search" />
        </NativeTabs.Trigger>
        <NativeTabs.Trigger name="account">
          <NativeTabs.Trigger.Label>Account</NativeTabs.Trigger.Label>
          <NativeTabs.Trigger.Icon sf="person.crop.circle" md="account_circle" />
        </NativeTabs.Trigger>
      </NativeTabs>
    </View>
  );
}
