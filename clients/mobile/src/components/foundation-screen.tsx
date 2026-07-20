import { Button, Column, Host, Text } from "@expo/ui";
import { ActivityIndicator, StyleSheet, useWindowDimensions, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

interface FoundationScreenProps {
  title: string;
  detail: string;
  loading?: boolean;
  actionLabel?: string;
  onAction?: () => void;
}

export function FoundationScreen({
  title,
  detail,
  loading = false,
  actionLabel,
  onAction,
}: FoundationScreenProps) {
  const { width } = useWindowDimensions();
  const nativeContentWidth = Math.max(0, Math.min(480, width - 64));

  return (
    <SafeAreaView style={styles.safeArea}>
      <View style={styles.content}>
        {loading ? <ActivityIndicator size="large" /> : null}
        <Host matchContents>
          <Column alignment="center" spacing={16} style={{ width: nativeContentWidth }}>
            <Text textStyle={styles.title}>{title}</Text>
            <Text style={styles.detail} textStyle={styles.detailText}>
              {detail}
            </Text>
            {actionLabel && onAction ? (
              <Button label={actionLabel} variant="filled" onPress={onAction} />
            ) : null}
          </Column>
        </Host>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1 },
  content: {
    alignItems: "center",
    flex: 1,
    gap: 16,
    justifyContent: "center",
    padding: 32,
  },
  title: {
    fontSize: 28,
    fontWeight: "700",
    textAlign: "center",
  },
  detail: {
    opacity: 0.7,
  },
  detailText: {
    fontSize: 16,
    textAlign: "center",
  },
});
