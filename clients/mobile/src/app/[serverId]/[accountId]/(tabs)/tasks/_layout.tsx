import { Stack } from "expo-router";

export default function TasksStackLayout() {
  return (
    <Stack screenOptions={{ headerLargeTitle: true }}>
      <Stack.Screen name="index" options={{ title: "My Tasks" }} />
      <Stack.Screen name="[spaceSlug]/[taskId]" options={{ title: "" }} />
    </Stack>
  );
}
