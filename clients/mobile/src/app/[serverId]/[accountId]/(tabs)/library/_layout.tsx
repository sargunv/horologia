import { Stack } from "expo-router";

export default function LibraryStackLayout() {
  return (
    <Stack screenOptions={{ headerLargeTitle: true }}>
      <Stack.Screen name="index" options={{ title: "Library" }} />
      <Stack.Screen name="spaces/index" options={{ title: "Spaces" }} />
      <Stack.Screen name="spaces/[spaceSlug]/index" options={{ title: "" }} />
      <Stack.Screen name="spaces/[spaceSlug]/tasks/index" options={{ title: "" }} />
      <Stack.Screen name="spaces/[spaceSlug]/tasks/[taskId]" options={{ title: "" }} />
      <Stack.Screen name="recipes/index" options={{ title: "Recipes" }} />
      <Stack.Screen name="recipes/[spaceSlug]/[recipeId]" options={{ title: "" }} />
      <Stack.Screen name="spaces/[spaceSlug]/recipes/index" options={{ title: "" }} />
      <Stack.Screen name="spaces/[spaceSlug]/recipes/[recipeId]" options={{ title: "" }} />
    </Stack>
  );
}
