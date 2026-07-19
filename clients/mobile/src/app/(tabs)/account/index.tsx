import { createQueries, createSettingsCommands } from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, ListItem, Text } from "@expo/ui";
import { useIsFocused } from "expo-router";
import { useEffect, useMemo, useState } from "react";
import { Alert, Appearance, Linking } from "react-native";

import { useSession } from "@/auth/session-context";
import { FormField, FormPicker, FormSection } from "@/components/forms";
import { NativeFormScreen } from "@/components/native-screen";
import { ScreenState } from "@/components/screen-state";

type Schema = components["schemas"];
type AppearanceMode = Schema["AppearanceMode"];
type SettingsCommands = ReturnType<typeof createSettingsCommands>;

const APPEARANCE_MODES = [
  { value: "system", label: "System" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
] as const;

export default function AccountScreen() {
  const isFocused = useIsFocused();
  const session = useSession();
  if (!isFocused) return null;
  if (!session.profile || !session.client) {
    return <ScreenState loading title="Opening account" />;
  }
  return (
    <AccountSettings
      client={session.client}
      onSignOut={() => session.signOut()}
      profile={session.profile}
    />
  );
}

function AccountSettings({
  client,
  onSignOut,
  profile,
}: {
  client: NonNullable<ReturnType<typeof useSession>["client"]>;
  onSignOut: () => Promise<void>;
  profile: NonNullable<ReturnType<typeof useSession>["profile"]>;
}) {
  const queryClient = useQueryClient();
  const [notice, setNotice] = useState<string | null>(null);
  const queries = useMemo(
    () => createQueries({ serverId: profile.id, apiClient: client }),
    [client, profile.id],
  );
  const user = useQuery(queries.currentUserQueryOptions);
  const serverInfo = useQuery(queries.serverInfoQueryOptions);
  useEffect(() => {
    if (user.data) applyAppearance(user.data.appearanceMode);
  }, [user.data]);
  if (user.isPending || serverInfo.isPending) {
    return <ScreenState loading title="Loading account settings" />;
  }
  if (user.isError || serverInfo.isError) {
    const error = user.error ?? serverInfo.error;
    const source = user.isError ? "profile" : "server information";
    return (
      <ScreenState
        detail={`${source}: ${error instanceof Error ? error.message : "request failed"}`}
        title="Account unavailable"
      />
    );
  }
  const commands = createSettingsCommands({
    serverId: profile.id,
    apiClient: client,
    queryClient,
    onCacheError: () => setNotice("Saved. Refresh if a value still looks stale."),
  });
  return (
    <NativeFormScreen>
      <FormSection title="Account settings">
        <Text>Profile, security, appearance, and this server.</Text>
      </FormSection>
      <ProfileEditor commands={commands} key={user.data.updatedAt} user={user.data} />
      <AppearanceEditor commands={commands} user={user.data} />
      <PasswordEditor commands={commands} user={user.data} />
      <WebTokenHandoff serverBaseUrl={profile.baseUrl} />
      <FormSection title="Server information">
        <Info label="Name" value={profile.displayName} />
        <Info label="Address" value={profile.baseUrl} />
        <Info label="API version" value={`${serverInfo.data.apiVersion}`} />
        <Info
          label="Capabilities"
          value={serverInfo.data.capabilities.join(", ") || "Standard API"}
        />
      </FormSection>
      <AccountDanger commands={commands} onDeleted={onSignOut} user={user.data} />
      <Button label="Sign out securely" onPress={() => void onSignOut()} variant="filled" />
      {notice ? <Text>{notice}</Text> : null}
      <Text>Global owner administration stays in the web app.</Text>
    </NativeFormScreen>
  );
}

function ProfileEditor({ commands, user }: { commands: SettingsCommands; user: Schema["User"] }) {
  const [name, setName] = useState(user.name);
  const [email, setEmail] = useState(user.email);
  const mutation = useMutation({
    mutationFn: () => commands.updateUser(user.id, { name: name.trim(), email: email.trim() }),
  });
  return (
    <FormSection title="Profile">
      <FormField label="Name" onChangeText={setName} value={name} />
      <FormField
        autoCapitalize="none"
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        value={email}
      />
      {mutation.error ? <ErrorText message={mutation.error.message} /> : null}
      <Button
        disabled={!name.trim() || !email.trim() || mutation.isPending}
        label={mutation.isPending ? "Saving…" : "Save profile"}
        onPress={() => mutation.mutate()}
        variant="filled"
      />
    </FormSection>
  );
}

function AppearanceEditor({
  commands,
  user,
}: {
  commands: SettingsCommands;
  user: Schema["User"];
}) {
  const [mode, setMode] = useState<AppearanceMode>(user.appearanceMode);
  const mutation = useMutation({
    mutationFn: (value: AppearanceMode) => commands.updateUser(user.id, { appearanceMode: value }),
  });
  function select(value: AppearanceMode) {
    const previous = mode;
    setMode(value);
    applyAppearance(value);
    mutation.mutate(value, {
      onError: () => {
        setMode(previous);
        applyAppearance(previous);
      },
    });
  }
  return (
    <FormSection title="Appearance">
      <Text>Use the system appearance or force a light or dark native shell.</Text>
      <FormPicker
        label="Appearance mode"
        onChange={select}
        options={APPEARANCE_MODES}
        value={mode}
      />
      {mutation.error ? <ErrorText message={mutation.error.message} /> : null}
    </FormSection>
  );
}

function PasswordEditor({ commands, user }: { commands: SettingsCommands; user: Schema["User"] }) {
  const [password, setPassword] = useState("");
  const mutation = useMutation({
    mutationFn: () => commands.updateUser(user.id, { setPassword: password }),
    onSuccess: () => setPassword(""),
  });
  return (
    <FormSection title="Password">
      <Text>{user.hasPassword ? "Password is set." : "No password is set."}</Text>
      <FormField
        label="New password"
        maxLength={72}
        onChangeText={setPassword}
        secureTextEntry
        value={password}
      />
      {mutation.error ? <ErrorText message={mutation.error.message} /> : null}
      <Button
        disabled={password.length < 8 || mutation.isPending}
        label={
          mutation.isPending ? "Saving…" : user.hasPassword ? "Change password" : "Set password"
        }
        onPress={() => mutation.mutate()}
        variant="filled"
      />
    </FormSection>
  );
}

function WebTokenHandoff({ serverBaseUrl }: { serverBaseUrl: string }) {
  const settingsUrl = new URL("settings", `${serverBaseUrl.replace(/\/+$/u, "")}/`).toString();
  return (
    <FormSection title="API tokens">
      <Text>
        Personal API tokens are full-trust credentials. The app uses a scoped OAuth session, so
        token management stays in the signed-in web app rather than allowing that session to elevate
        its own access.
      </Text>
      <Button
        label="Manage API tokens on web"
        onPress={() => void Linking.openURL(settingsUrl)}
        variant="filled"
      />
    </FormSection>
  );
}

function AccountDanger({
  commands,
  onDeleted,
  user,
}: {
  commands: SettingsCommands;
  onDeleted: () => Promise<void>;
  user: Schema["User"];
}) {
  const deletion = useMutation({
    mutationFn: () => commands.deleteUser(user.id),
    onSuccess: onDeleted,
  });
  return (
    <FormSection title="Danger zone">
      <Text>Permanently delete this account, memberships, assignments, and tokens.</Text>
      {deletion.error ? <ErrorText message={deletion.error.message} /> : null}
      <Button
        disabled={deletion.isPending}
        label={deletion.isPending ? "Deleting…" : "Delete account"}
        onPress={() =>
          Alert.alert("Delete account?", `Permanently delete ${user.email}?`, [
            { text: "Cancel", style: "cancel" },
            { text: "Delete", style: "destructive", onPress: () => deletion.mutate() },
          ])
        }
        variant="text"
      />
    </FormSection>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <ListItem supportingText={value}>
      <Text>{label}</Text>
    </ListItem>
  );
}

function ErrorText({ message }: { message: string }) {
  return <Text>{message}</Text>;
}

function applyAppearance(mode: AppearanceMode) {
  Appearance.setColorScheme(mode === "system" ? "unspecified" : mode);
}
