import { Button, Column, Host, Row, Spacer, Text } from "@expo/ui";
import { createWidgetSnapshotV1 } from "@horologia/client-core";
import { useRouter } from "expo-router";
import { useEffect } from "react";
import {
  FlatList,
  Pressable,
  StyleSheet,
  Text as NativeText,
  useColorScheme,
  useWindowDimensions,
  View,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { NativeFeatureProof } from "@/components/native-feature-proof";
import { OAuthSpikeCard } from "@/components/oauth-spike-card";
import { publishWidgetSnapshot } from "@/widgets/publishWidgetSnapshot";

const DEMO_TASKS = [
  { id: "1", spaceSlug: "home", title: "Water the herbs", detail: "Today · Home" },
  { id: "2", spaceSlug: "home", title: "Change the air filter", detail: "Overdue · Home" },
  { id: "3", spaceSlug: "kitchen", title: "Plan next week's meals", detail: "Kitchen" },
] as const;

export default function MyTasksScreen() {
  const router = useRouter();
  const colorScheme = useColorScheme();
  const { width } = useWindowDimensions();
  const expanded = width >= 700;
  const primaryText = colorScheme === "dark" ? "#F3F5F4" : "#17201B";
  const secondaryText = colorScheme === "dark" ? "#B9C2BC" : "#59655E";

  useEffect(() => {
    void publishWidgetSnapshot(
      createWidgetSnapshotV1({
        serverId: "local-spike",
        accountId: "demo",
        generatedAt: new Date().toISOString(),
        tasks: DEMO_TASKS.map((task) => ({
          id: task.id,
          spaceSlug: task.spaceSlug,
          title: task.title,
          due: null,
          status: "open",
        })),
      }),
    ).catch((error: unknown) => {
      console.error("Unable to publish the widget snapshot", error);
    });
  }, []);

  return (
    <SafeAreaView style={styles.safeArea} edges={["bottom"]}>
      <View style={[styles.layout, expanded && styles.expandedLayout]}>
        <View style={styles.listPane}>
          <FlatList
            data={DEMO_TASKS}
            keyExtractor={(task) => task.id}
            contentContainerStyle={styles.list}
            ListHeaderComponent={
              <View style={styles.listHeader}>
                <View style={styles.nativeHeader}>
                  <Host matchContents={{ vertical: true }}>
                    <Column spacing={8}>
                      <Row alignment="center" spacing={10}>
                        <Text textStyle={{ color: primaryText, fontSize: 28, fontWeight: "bold" }}>
                          My Tasks
                        </Text>
                        <Spacer />
                        <Button label="Refresh" variant="text" onPress={() => undefined} />
                      </Row>
                      <Text textStyle={{ color: secondaryText }}>
                        A native Expo UI foundation shared with the web client core.
                      </Text>
                    </Column>
                  </Host>
                </View>
                <NativeFeatureProof />
                <OAuthSpikeCard />
              </View>
            }
            renderItem={({ item }) => (
              <Pressable
                accessibilityLabel={`${item.title}, ${item.detail}`}
                accessibilityRole="button"
                style={({ pressed }) => [styles.taskButton, pressed && styles.taskButtonPressed]}
                onPress={() =>
                  router.push({
                    pathname: "/task/[spaceSlug]/[taskId]",
                    params: { spaceSlug: item.spaceSlug, taskId: item.id },
                  })
                }
              >
                <View style={styles.taskRow}>
                  <View style={styles.taskText}>
                    <NativeText style={[styles.taskTitle, { color: primaryText }]}>
                      {item.title}
                    </NativeText>
                    <NativeText style={[styles.taskDetail, { color: secondaryText }]}>
                      {item.detail}
                    </NativeText>
                  </View>
                  <NativeText style={[styles.chevron, { color: secondaryText }]}>›</NativeText>
                </View>
              </Pressable>
            )}
          />
        </View>
        {expanded ? (
          <View style={styles.detailPane}>
            <NativeText style={[styles.detailTitle, { color: primaryText }]}>
              Choose a task
            </NativeText>
            <NativeText style={[styles.detailCopy, { color: secondaryText }]}>
              The expanded layout keeps the list visible while showing task details.
            </NativeText>
          </View>
        ) : null}
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1 },
  layout: { flex: 1 },
  expandedLayout: { flexDirection: "row" },
  listPane: { flex: 1, minWidth: 320 },
  listHeader: { gap: 8, marginBottom: 12 },
  nativeHeader: { paddingHorizontal: 20, paddingVertical: 16 },
  list: { gap: 8, padding: 12, paddingBottom: 32 },
  taskButton: { borderRadius: 14 },
  taskButtonPressed: { backgroundColor: "rgba(90,110,98,0.12)" },
  taskRow: {
    alignItems: "center",
    flexDirection: "row",
    minHeight: 64,
    paddingHorizontal: 16,
    paddingVertical: 10,
  },
  taskText: { flex: 1, gap: 4 },
  taskTitle: { fontSize: 17, fontWeight: "600" },
  taskDetail: { fontSize: 14 },
  chevron: { fontSize: 28, opacity: 0.65 },
  detailPane: {
    alignItems: "center",
    borderLeftColor: "rgba(128,128,128,0.25)",
    borderLeftWidth: StyleSheet.hairlineWidth,
    flex: 1.25,
    justifyContent: "center",
    padding: 32,
  },
  detailTitle: { fontSize: 28, fontWeight: "700", marginBottom: 8 },
  detailCopy: { fontSize: 16, maxWidth: 360, textAlign: "center" },
});
