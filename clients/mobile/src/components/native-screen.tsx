import { FieldGroup, Host } from "@expo/ui";
import { imePadding } from "@expo/ui/jetpack-compose/modifiers";
import { scrollDismissesKeyboard } from "@expo/ui/swift-ui/modifiers";
import type { PropsWithChildren } from "react";

export function NativeFormScreen({ children }: PropsWithChildren) {
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <FieldGroup modifiers={[imePadding(), scrollDismissesKeyboard("immediately")]}>
        {children}
      </FieldGroup>
    </Host>
  );
}
