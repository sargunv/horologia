import { Stack } from "expo-router";

export default function AccountStackLayout() {
  return (
    <Stack>
      <Stack.Screen name="index" options={{ title: "Account" }} />
    </Stack>
  );
}
