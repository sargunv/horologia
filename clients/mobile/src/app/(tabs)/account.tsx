import { Pressable, StyleSheet, Text, View } from "react-native";

import { useSession } from "@/auth/session-context";
import { colors } from "@/design/tokens";

export default function AccountScreen() {
  const session = useSession();
  return (
    <View style={styles.content}>
      <Text style={styles.label}>CONNECTED SERVER</Text>
      <Text style={styles.value}>{session.profile?.displayName}</Text>
      <Text style={styles.address}>{session.profile?.baseUrl}</Text>
      <Pressable
        accessibilityRole="button"
        onPress={() => void session.signOut()}
        style={styles.logout}
      >
        <Text style={styles.logoutLabel}>Sign out securely</Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  content: { flex: 1, padding: 24 },
  label: { color: colors.muted, fontSize: 12, fontWeight: "700", letterSpacing: 1.2 },
  value: { color: colors.ink, fontSize: 24, fontWeight: "700", marginTop: 8 },
  address: { color: colors.muted, fontSize: 15, marginTop: 4 },
  logout: {
    borderColor: colors.danger,
    borderRadius: 14,
    borderWidth: StyleSheet.hairlineWidth,
    marginTop: 32,
    minHeight: 50,
    padding: 14,
  },
  logoutLabel: { color: colors.danger, fontSize: 16, fontWeight: "600", textAlign: "center" },
});
