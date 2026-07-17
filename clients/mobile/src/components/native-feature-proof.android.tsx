import { AssistChip, Host, Text } from "@expo/ui/jetpack-compose";

export function NativeFeatureProof() {
  return (
    <Host matchContents={{ vertical: true }} style={{ marginHorizontal: 20 }}>
      <AssistChip onClick={() => undefined}>
        <AssistChip.Label>
          <Text>Material 3 Compose active</Text>
        </AssistChip.Label>
      </AssistChip>
    </Host>
  );
}
