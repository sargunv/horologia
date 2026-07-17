import { Redirect, useRouter } from "expo-router";
import { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { colors } from "@/design/tokens";

export default function OnboardingScreen() {
  const router = useRouter();
  const session = useSession();
  const [serverUrl, setServerUrl] = useState("http://localhost:8080");

  if (session.status === "signed-in") return <Redirect href="/(tabs)/tasks" />;

  const busy = session.status === "restoring" || session.status === "authorizing";
  return (
    <SafeAreaView style={styles.safeArea}>
      <View style={styles.content}>
        <Text accessibilityRole="header" style={styles.eyebrow}>
          HOROLOGIA
        </Text>
        <Text style={styles.title}>Your household, close at hand.</Text>
        <Text style={styles.copy}>
          Connect securely to your self-hosted server. Your address stays on this device.
        </Text>
        {session.profile ? (
          <View style={styles.serverCard}>
            <Text style={styles.serverLabel}>Ready to connect</Text>
            <Text style={styles.serverName}>{session.profile.displayName}</Text>
            <PrimaryButton
              disabled={busy}
              label={busy ? "Opening sign-in…" : "Continue to sign in"}
              onPress={() => void session.signIn().then(() => router.replace("/(tabs)/tasks"))}
            />
          </View>
        ) : (
          <View style={styles.form}>
            <Text style={styles.label}>Server address</Text>
            <TextInput
              accessibilityLabel="Server address"
              autoCapitalize="none"
              autoCorrect={false}
              keyboardType="url"
              onChangeText={setServerUrl}
              placeholder="https://home.example.com"
              returnKeyType="go"
              style={styles.input}
              value={serverUrl}
            />
            <PrimaryButton
              disabled={busy}
              label={busy ? "Checking server…" : "Connect"}
              onPress={() => void session.connect(serverUrl)}
            />
          </View>
        )}
        {busy ? <ActivityIndicator accessibilityLabel="Working" /> : null}
        {session.detail ? (
          <Text accessibilityLiveRegion="polite" style={styles.error}>
            {session.detail}
          </Text>
        ) : null}
        {session.status === "error" ? (
          <Pressable accessibilityRole="button" onPress={() => void session.recover()}>
            <Text style={styles.retry}>Try again</Text>
          </Pressable>
        ) : null}
      </View>
    </SafeAreaView>
  );
}

function PrimaryButton(props: { disabled: boolean; label: string; onPress: () => void }) {
  return (
    <Pressable
      accessibilityRole="button"
      disabled={props.disabled}
      onPress={() => props.onPress()}
      style={({ pressed }) => [
        styles.button,
        pressed && styles.pressed,
        props.disabled && styles.disabled,
      ]}
    >
      <Text style={styles.buttonLabel}>{props.label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  content: {
    alignSelf: "center",
    flex: 1,
    justifyContent: "center",
    maxWidth: 520,
    padding: 28,
    width: "100%",
  },
  eyebrow: {
    color: colors.accent,
    fontSize: 13,
    fontWeight: "800",
    letterSpacing: 2,
    marginBottom: 12,
  },
  title: {
    color: colors.ink,
    fontSize: 38,
    fontWeight: "700",
    letterSpacing: -1.1,
    lineHeight: 43,
  },
  copy: { color: colors.muted, fontSize: 17, lineHeight: 25, marginBottom: 30, marginTop: 14 },
  form: { gap: 12 },
  label: { color: colors.ink, fontSize: 14, fontWeight: "600" },
  input: {
    backgroundColor: colors.surface,
    borderColor: colors.outline,
    borderRadius: 14,
    borderWidth: StyleSheet.hairlineWidth,
    color: colors.ink,
    fontSize: 17,
    minHeight: 52,
    paddingHorizontal: 16,
  },
  button: {
    alignItems: "center",
    backgroundColor: colors.accent,
    borderRadius: 14,
    justifyContent: "center",
    minHeight: 52,
    paddingHorizontal: 18,
  },
  buttonLabel: { color: "#FFFFFF", fontSize: 17, fontWeight: "700" },
  pressed: { opacity: 0.76 },
  disabled: { opacity: 0.55 },
  serverCard: { backgroundColor: colors.surface, borderRadius: 20, gap: 12, padding: 20 },
  serverLabel: { color: colors.muted, fontSize: 13, fontWeight: "600" },
  serverName: { color: colors.ink, fontSize: 22, fontWeight: "700" },
  error: { color: colors.danger, fontSize: 14, lineHeight: 20, marginTop: 18, textAlign: "center" },
  retry: {
    color: colors.accent,
    fontSize: 16,
    fontWeight: "600",
    padding: 12,
    textAlign: "center",
  },
});
