import { completeMobileAuthorization } from "@/auth/oauth";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

type CompletionState = "authorizing" | "authorized" | "error";

export default function OAuthCallbackScreen() {
  const router = useRouter();
  const { code, state } = useLocalSearchParams<{ code?: string; state?: string }>();
  const [completionState, setCompletionState] = useState<CompletionState>("authorizing");
  const [detail, setDetail] = useState("Completing secure sign-in…");

  useEffect(() => {
    if (!code) {
      setCompletionState("error");
      setDetail("The authorization server did not return a code.");
      return;
    }

    void completeMobileAuthorization(code, state)
      .then((response) => {
        setCompletionState("authorized");
        setDetail(`OAuth ready · ${response.scope?.split(" ").length ?? 0} user-facing scopes`);
      })
      .catch((error: unknown) => {
        setCompletionState("error");
        setDetail(error instanceof Error ? error.message : "Authorization failed");
      });
  }, [code, state]);

  return (
    <SafeAreaView style={styles.safeArea}>
      <View style={styles.content}>
        {completionState === "authorizing" ? <ActivityIndicator size="large" /> : null}
        <Text accessibilityRole="header" style={styles.title}>
          {completionState === "authorized"
            ? "Authorized"
            : completionState === "error"
              ? "Sign-in failed"
              : "Signing in"}
        </Text>
        <Text accessibilityLiveRegion="polite" style={styles.detail}>
          {detail}
        </Text>
        {completionState !== "authorizing" ? (
          <Pressable
            accessibilityRole="button"
            onPress={() => router.replace("/")}
            style={({ pressed }) => [styles.button, pressed && styles.buttonPressed]}
          >
            <Text style={styles.buttonLabel}>Back to My Tasks</Text>
          </Pressable>
        ) : null}
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1 },
  content: {
    alignItems: "center",
    flex: 1,
    gap: 14,
    justifyContent: "center",
    padding: 32,
  },
  title: { fontSize: 30, fontWeight: "700" },
  detail: { fontSize: 16, maxWidth: 420, opacity: 0.7, textAlign: "center" },
  button: {
    backgroundColor: "#2F6D4B",
    borderRadius: 999,
    marginTop: 12,
    paddingHorizontal: 20,
    paddingVertical: 12,
  },
  buttonPressed: { opacity: 0.75 },
  buttonLabel: { color: "white", fontSize: 16, fontWeight: "600" },
});
