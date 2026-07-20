import { GestureHandlerRootView } from "react-native-gesture-handler";

import { DarkTheme, DefaultTheme, Stack, ThemeProvider } from "expo-router";
import * as SplashScreen from "expo-splash-screen";
import { StatusBar } from "expo-status-bar";
import { useEffect } from "react";
import { StyleSheet, useColorScheme } from "react-native";
import { SafeAreaProvider } from "react-native-safe-area-context";

import { SessionProvider, useSession } from "@/auth/session-context";

void SplashScreen.preventAutoHideAsync();

export default function RootLayout() {
  return (
    <GestureHandlerRootView style={styles.root}>
      <SafeAreaProvider>
        <SessionProvider>
          <RootNavigator />
        </SessionProvider>
      </SafeAreaProvider>
    </GestureHandlerRootView>
  );
}

function RootNavigator() {
  const colorScheme = useColorScheme();
  const session = useSession();

  useEffect(() => {
    if (session.status !== "restoring") {
      void SplashScreen.hideAsync();
    }
  }, [session.status]);

  return (
    <ThemeProvider value={colorScheme === "dark" ? DarkTheme : DefaultTheme}>
      <Stack>
        <Stack.Screen name="index" options={{ headerShown: false }} />
        <Stack.Screen name="oauth/callback" options={{ headerShown: false }} />
        <Stack.Screen name="[serverId]/[accountId]" options={{ headerShown: false }} />
      </Stack>
      <StatusBar style="auto" />
    </ThemeProvider>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
});
