import { authorizeMobile } from "@/auth/oauth";
import { useState } from "react";
import {
  Pressable,
  StyleSheet,
  Text,
  useColorScheme,
  useWindowDimensions,
  View,
} from "react-native";

const serverBaseUrl = "http://localhost:8080";

type AuthState = "idle" | "authorizing" | "authorized" | "error";

export function OAuthSpikeCard() {
  const colorScheme = useColorScheme();
  const { fontScale } = useWindowDimensions();
  const [state, setState] = useState<AuthState>("idle");
  const [detail, setDetail] = useState("Complete the local OAuth 2.1 + PKCE architecture gate.");
  const primaryText = colorScheme === "dark" ? "#F3F5F4" : "#17201B";
  const secondaryText = colorScheme === "dark" ? "#B9C2BC" : "#59655E";

  async function authorize() {
    setState("authorizing");
    setDetail("Waiting for the system browser…");
    try {
      const response = await authorizeMobile(serverBaseUrl);
      setState("authorized");
      setDetail(`OAuth ready · ${response.scope?.split(" ").length ?? 0} user-facing scopes`);
    } catch (error) {
      setState("error");
      setDetail(error instanceof Error ? error.message : "Authorization failed");
    }
  }

  return (
    <View style={[styles.card, fontScale >= 1.5 && styles.cardLargeText]}>
      <View style={styles.copy}>
        <Text style={[styles.title, { color: primaryText }]}>Local server sign-in</Text>
        <Text accessibilityLiveRegion="polite" style={[styles.detail, { color: secondaryText }]}>
          {detail}
        </Text>
      </View>
      <Pressable
        accessibilityRole="button"
        disabled={state === "authorizing"}
        onPress={() => void authorize()}
        style={({ pressed }) => [
          styles.button,
          fontScale >= 1.5 && styles.buttonLargeText,
          pressed && styles.buttonPressed,
          state === "authorized" && styles.buttonAuthorized,
        ]}
      >
        <Text style={styles.buttonLabel}>
          {state === "authorizing"
            ? "Opening…"
            : state === "authorized"
              ? "Authorized"
              : "Test OAuth"}
        </Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    alignItems: "center",
    borderColor: "rgba(120,130,124,0.25)",
    borderRadius: 16,
    borderWidth: StyleSheet.hairlineWidth,
    flexDirection: "row",
    gap: 12,
    marginHorizontal: 20,
    marginVertical: 8,
    padding: 14,
  },
  cardLargeText: { alignItems: "stretch", flexDirection: "column" },
  copy: { flex: 1, gap: 3 },
  title: { fontSize: 15, fontWeight: "600" },
  detail: { fontSize: 12 },
  button: {
    backgroundColor: "#2F6D4B",
    borderRadius: 999,
    minWidth: 104,
    paddingHorizontal: 16,
    paddingVertical: 10,
  },
  buttonLargeText: { alignSelf: "flex-end" },
  buttonAuthorized: { backgroundColor: "#41634D" },
  buttonPressed: { opacity: 0.75 },
  buttonLabel: { color: "white", fontSize: 14, fontWeight: "600", textAlign: "center" },
});
