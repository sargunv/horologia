import { createQueries, createSettingsCommands } from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { Alert, Appearance, Linking, ScrollView, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { ActionButton, ChoiceChips, FormField, FormSection } from "@/components/forms";
import { ScreenState } from "@/components/screen-state";
import { colors } from "@/design/tokens";

type Schema = components["schemas"];
type AppearanceMode = Schema["AppearanceMode"];
type SettingsCommands = ReturnType<typeof createSettingsCommands>;

const APPEARANCE_MODES = [
  { value: "system", label: "System" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
] as const;

export default function AccountScreen() {
  const session = useSession();
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
    () => createQueries({ serverId: profile.id, apiClient: client, appClient: client }),
    [client, profile.id],
  );
  const user = useQuery(queries.currentUserQueryOptions);
  const authConfig = useQuery(queries.authConfigQueryOptions);
  const serverInfo = useQuery(queries.serverInfoQueryOptions);
  useEffect(() => {
    if (user.data) applyAppearance(user.data.appearanceMode);
  }, [user.data]);
  if (user.isPending || authConfig.isPending || serverInfo.isPending) {
    return <ScreenState loading title="Loading account settings" />;
  }
  if (user.isError || authConfig.isError || serverInfo.isError) {
    const error = [user.error, authConfig.error, serverInfo.error].find(Boolean);
    const source = user.isError
      ? "profile"
      : authConfig.isError
        ? "authentication settings"
        : "server information";
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
    <SafeAreaView edges={["left", "right"]} style={styles.safeArea}>
      <ScrollView
        automaticallyAdjustKeyboardInsets
        contentContainerStyle={styles.scroll}
        keyboardDismissMode="on-drag"
        keyboardShouldPersistTaps="handled"
      >
        <Text accessibilityRole="header" style={styles.heading}>
          Account settings
        </Text>
        <Text style={styles.detail}>Profile, security, appearance, and this server.</Text>
        <ProfileEditor commands={commands} key={user.data.updatedAt} user={user.data} />
        <AppearanceEditor commands={commands} user={user.data} />
        {authConfig.data.password.enabled ? (
          <PasswordEditor commands={commands} user={user.data} />
        ) : null}
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
        <ActionButton label="Sign out securely" onPress={() => void onSignOut()} />
        {notice ? <Text style={styles.notice}>{notice}</Text> : null}
        <Text style={styles.exclusion}>Global owner administration stays in the web app.</Text>
      </ScrollView>
    </SafeAreaView>
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
      <FormField label="Name" onChangeText={setName} testID="profile-name-input" value={name} />
      <FormField
        autoCapitalize="none"
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        value={email}
      />
      {mutation.error ? <ErrorText message={mutation.error.message} /> : null}
      <ActionButton
        disabled={!name.trim() || !email.trim() || mutation.isPending}
        label={mutation.isPending ? "Saving…" : "Save profile"}
        onPress={() => mutation.mutate()}
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
      <Text style={styles.detail}>
        Use the system appearance or force a light or dark native shell.
      </Text>
      <ChoiceChips
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
      <Text style={styles.detail}>
        {user.hasPassword ? "Password is set." : "No password is set."}
      </Text>
      <FormField
        label="New password"
        maxLength={72}
        onChangeText={setPassword}
        secureTextEntry
        value={password}
      />
      {mutation.error ? <ErrorText message={mutation.error.message} /> : null}
      <ActionButton
        disabled={password.length < 8 || mutation.isPending}
        label={
          mutation.isPending ? "Saving…" : user.hasPassword ? "Change password" : "Set password"
        }
        onPress={() => mutation.mutate()}
      />
    </FormSection>
  );
}

function WebTokenHandoff({ serverBaseUrl }: { serverBaseUrl: string }) {
  const settingsUrl = new URL("settings", `${serverBaseUrl.replace(/\/+$/u, "")}/`).toString();
  return (
    <FormSection title="API tokens">
      <Text style={styles.detail}>
        Personal API tokens are full-trust credentials. The app uses a scoped OAuth session, so
        token management stays in the signed-in web app rather than allowing that session to elevate
        its own access.
      </Text>
      <ActionButton
        label="Manage API tokens on web"
        onPress={() => void Linking.openURL(settingsUrl)}
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
      <Text style={styles.detail}>
        Permanently delete this account, memberships, assignments, and tokens.
      </Text>
      {deletion.error ? <ErrorText message={deletion.error.message} /> : null}
      <ActionButton
        destructive
        disabled={deletion.isPending}
        label={deletion.isPending ? "Deleting…" : "Delete account"}
        onPress={() =>
          Alert.alert("Delete account?", `Permanently delete ${user.email}?`, [
            { text: "Cancel", style: "cancel" },
            { text: "Delete", style: "destructive", onPress: () => deletion.mutate() },
          ])
        }
      />
    </FormSection>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.info}>
      <Text style={styles.meta}>{label}</Text>
      <Text selectable style={styles.infoValue}>
        {value}
      </Text>
    </View>
  );
}

function ErrorText({ message }: { message: string }) {
  return (
    <Text accessibilityRole="alert" style={styles.error}>
      {message}
    </Text>
  );
}

function applyAppearance(mode: AppearanceMode) {
  Appearance.setColorScheme(mode === "system" ? "unspecified" : mode);
}

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  scroll: {
    alignSelf: "center",
    gap: 14,
    maxWidth: 760,
    padding: 18,
    paddingBottom: 54,
    width: "100%",
  },
  heading: { color: colors.ink, fontSize: 30, fontWeight: "700" },
  detail: { color: colors.muted, fontSize: 14, lineHeight: 20 },
  meta: { color: colors.muted, fontSize: 12, marginTop: 2 },
  info: {
    borderBottomColor: colors.outline,
    borderBottomWidth: StyleSheet.hairlineWidth,
    paddingBottom: 10,
  },
  infoValue: { color: colors.ink, fontSize: 15, marginTop: 4 },
  error: { color: colors.danger, fontSize: 13, fontWeight: "600" },
  notice: { color: colors.accent, fontSize: 13, fontWeight: "600" },
  exclusion: { color: colors.muted, fontSize: 12, textAlign: "center" },
});
