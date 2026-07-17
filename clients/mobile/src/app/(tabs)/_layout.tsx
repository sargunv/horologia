import { Redirect, Tabs } from "expo-router";
import { type ColorValue, Text } from "react-native";

import { useSession } from "@/auth/session-context";
import { colors } from "@/design/tokens";

const icon =
  (glyph: string) =>
  ({ color }: { color: ColorValue }) => (
    <Text accessibilityElementsHidden style={{ color, fontSize: 19 }}>
      {glyph}
    </Text>
  );

export default function TabLayout() {
  const session = useSession();
  if (session.status !== "signed-in") return <Redirect href="/" />;
  return (
    <Tabs screenOptions={{ tabBarActiveTintColor: colors.accent }}>
      <Tabs.Screen
        name="tasks"
        options={{ title: "Tasks", tabBarAccessibilityLabel: "Tasks tab", tabBarIcon: icon("✓") }}
      />
      <Tabs.Screen
        name="library"
        options={{
          title: "Library",
          tabBarAccessibilityLabel: "Library tab",
          tabBarIcon: icon("▤"),
        }}
      />
      <Tabs.Screen
        name="search"
        options={{ title: "Search", tabBarAccessibilityLabel: "Search tab", tabBarIcon: icon("⌕") }}
      />
      <Tabs.Screen
        name="account"
        options={{
          title: "Account",
          tabBarAccessibilityLabel: "Account tab",
          tabBarIcon: icon("●"),
        }}
      />
    </Tabs>
  );
}
