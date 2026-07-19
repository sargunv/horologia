import { Button, FieldGroup, Host, Text } from "@expo/ui";
import { Redirect } from "expo-router";
import { useState } from "react";

import { useSession } from "@/auth/session-context";
import { FormField } from "@/components/forms";
import { ScreenState } from "@/components/screen-state";

export default function OnboardingScreen() {
  const session = useSession();
  const [serverUrl, setServerUrl] = useState("");

  if (session.status === "restoring") {
    return <ScreenState loading title="Restoring session" />;
  }
  if (session.status === "signed-in") return <Redirect href="/(tabs)/tasks" />;

  const busy = session.status === "authorizing";
  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <FieldGroup>
        <FieldGroup.Section title="Horologia">
          <Text>Your household, close at hand.</Text>
          <Text>
            Connect securely to your self-hosted server. Your address stays on this device.
          </Text>
        </FieldGroup.Section>
        {session.profile ? (
          <FieldGroup.Section title="Ready to connect">
            <Text>{session.profile.displayName}</Text>
            <Button
              disabled={busy}
              label={busy ? "Opening sign-in…" : "Continue to sign in"}
              onPress={() => void session.signIn()}
              variant="filled"
            />
          </FieldGroup.Section>
        ) : (
          <FieldGroup.Section title="Server">
            <FormField
              autoCapitalize="none"
              autoCorrect={false}
              keyboardType="url"
              label="Server address"
              onChangeText={setServerUrl}
              placeholder="https://home.example.com"
              returnKeyType="go"
              value={serverUrl}
            />
            <Button
              disabled={busy || !serverUrl.trim()}
              label={busy ? "Checking server…" : "Connect"}
              onPress={() => void session.connect(serverUrl)}
              variant="filled"
            />
          </FieldGroup.Section>
        )}
        {busy ? <Text>Working…</Text> : null}
        {session.detail ? (
          <FieldGroup.Section title="Connection">
            <Text>{session.detail}</Text>
          </FieldGroup.Section>
        ) : null}
        {session.status === "error" ? (
          <Button label="Try again" onPress={() => void session.recover()} variant="filled" />
        ) : null}
      </FieldGroup>
    </Host>
  );
}
