import { Button, Column, Host, Text } from "@expo/ui";
import { useLocalSearchParams, useRouter } from "expo-router";
import { StyleSheet } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

export default function TaskDetailScreen() {
  const { spaceSlug, taskId } = useLocalSearchParams<{
    spaceSlug: string;
    taskId: string;
  }>();
  const router = useRouter();

  return (
    <SafeAreaView style={styles.safeArea}>
      <Host style={styles.host}>
        <Column spacing={16}>
          <Text textStyle={{ fontSize: 30, fontWeight: "bold" }}>{`Task ${taskId}`}</Text>
          <Text textStyle={{ color: "#65716A" }}>{`Space: ${spaceSlug}`}</Text>
          <Text>
            This route proves server-safe widget and application deep links before feature data is
            connected.
          </Text>
          <Button label="Back to My Tasks" onPress={() => router.back()} />
        </Column>
      </Host>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1 },
  host: { flex: 1, padding: 24 },
});
