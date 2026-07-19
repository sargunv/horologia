import { Button, FieldGroup, Host, Text } from "@expo/ui";

export function ScreenState({
  title,
  detail,
  loading = false,
  onAction,
  actionLabel = "Try again",
}: {
  title: string;
  detail?: string;
  loading?: boolean;
  onAction?: () => void;
  actionLabel?: string;
}) {
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <FieldGroup>
        <FieldGroup.Section title={loading ? "Loading" : "Status"}>
          <Text>{title}</Text>
          {detail ? <Text>{detail}</Text> : null}
          {onAction ? <Button label={actionLabel} onPress={onAction} variant="filled" /> : null}
        </FieldGroup.Section>
      </FieldGroup>
    </Host>
  );
}
