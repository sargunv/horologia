import { ActivityIndicator, Pressable, StyleSheet, Text, View } from "react-native";

import { colors } from "@/design/tokens";

export function ScreenState(props: {
  title: string;
  detail?: string;
  loading?: boolean;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <View style={styles.container}>
      {props.loading ? <ActivityIndicator accessibilityLabel="Loading" size="large" /> : null}
      <Text accessibilityRole="header" style={styles.title}>
        {props.title}
      </Text>
      {props.detail ? <Text style={styles.detail}>{props.detail}</Text> : null}
      {props.actionLabel && props.onAction ? (
        <Pressable
          accessibilityRole="button"
          onPress={() => props.onAction?.()}
          style={styles.action}
        >
          <Text style={styles.actionLabel}>{props.actionLabel}</Text>
        </Pressable>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { alignItems: "center", flex: 1, justifyContent: "center", padding: 32 },
  title: { color: colors.ink, fontSize: 22, fontWeight: "700", marginTop: 12, textAlign: "center" },
  detail: {
    color: colors.muted,
    fontSize: 15,
    lineHeight: 22,
    marginTop: 8,
    maxWidth: 420,
    textAlign: "center",
  },
  action: { minHeight: 44, padding: 12 },
  actionLabel: { color: colors.accent, fontSize: 16, fontWeight: "600" },
});
