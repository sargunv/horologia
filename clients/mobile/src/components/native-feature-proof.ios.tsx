import { Host, Label } from "@expo/ui/swift-ui";
import { padding } from "@expo/ui/swift-ui/modifiers";

export function NativeFeatureProof() {
  return (
    <Host matchContents={{ vertical: true }} style={{ marginHorizontal: 20 }}>
      <Label
        title="SwiftUI foundation active"
        systemImage="sparkles"
        modifiers={[padding({ vertical: 8 })]}
      />
    </Host>
  );
}
