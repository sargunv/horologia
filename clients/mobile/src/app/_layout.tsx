import { createQueryClient } from "@horologia/client-core/runtime";
import { QueryClientProvider } from "@tanstack/react-query";
import { DarkTheme, DefaultTheme, Stack, ThemeProvider } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { useState } from "react";
import { useColorScheme } from "react-native";

import { SessionProvider } from "@/auth/session-context";

export default function RootLayout() {
  const colorScheme = useColorScheme();
  const [queryClient] = useState(createQueryClient);

  return (
    <QueryClientProvider client={queryClient}>
      <SessionProvider>
        <ThemeProvider value={colorScheme === "dark" ? DarkTheme : DefaultTheme}>
          <Stack>
            <Stack.Screen name="index" options={{ headerShown: false }} />
            <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
            <Stack.Screen name="task/[spaceSlug]/[taskId]" options={{ title: "Task" }} />
            <Stack.Screen name="oauth/callback" options={{ title: "Sign in" }} />
          </Stack>
          <StatusBar style="auto" />
        </ThemeProvider>
      </SessionProvider>
    </QueryClientProvider>
  );
}
