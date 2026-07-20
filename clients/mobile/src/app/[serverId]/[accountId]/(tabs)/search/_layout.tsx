import { Stack } from "expo-router";

export default function SearchStackLayout() {
  return (
    <Stack>
      <Stack.Screen name="index" options={{ title: "Search" }} />
    </Stack>
  );
}
