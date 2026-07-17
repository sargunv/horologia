import {
  createLibraryCommands,
  slugifySpaceName,
  type HorologiaClient,
  type ServerProfile,
} from "@horologia/client-core";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { useState } from "react";
import { ScrollView, StyleSheet, Text } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { ActionButton, FormField, FormSection } from "@/components/forms";
import { ScreenState } from "@/components/screen-state";
import { colors } from "@/design/tokens";

export default function NewSpaceScreen() {
  const session = useSession();
  if (!session.profile || !session.client) return <ScreenState loading title="Opening editor" />;
  return <NewSpace client={session.client} profile={session.profile} />;
}

function NewSpace({ client, profile }: { client: HorologiaClient; profile: ServerProfile }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [slugEdited, setSlugEdited] = useState(false);
  const commands = createLibraryCommands({
    serverId: profile.id,
    apiClient: client,
    queryClient,
  });
  const mutation = useMutation({
    mutationFn: () =>
      commands.createSpace({ name: name.trim(), slug, ...(description ? { description } : {}) }),
    onSuccess(space) {
      router.replace({ pathname: "/space/[spaceSlug]", params: { spaceSlug: space.slug } });
    },
  });
  return (
    <SafeAreaView edges={["left", "right", "bottom"]} style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <Text accessibilityRole="header" style={styles.heading}>
          New space
        </Text>
        <Text style={styles.detail}>A calm home for related tasks, recipes, and people.</Text>
        <FormSection title="Details">
          <FormField
            label="Name"
            maxLength={200}
            onChangeText={(value) => {
              setName(value);
              if (!slugEdited) setSlug(slugifySpaceName(value));
            }}
            value={name}
          />
          <FormField
            autoCapitalize="none"
            label="Slug"
            maxLength={100}
            onChangeText={(value) => {
              setSlugEdited(true);
              setSlug(value);
            }}
            value={slug}
          />
          <FormField
            label="Description"
            maxLength={2_000}
            multiline
            onChangeText={setDescription}
            value={description}
          />
        </FormSection>
        {mutation.error ? <Text style={styles.error}>{mutation.error.message}</Text> : null}
        <ActionButton
          disabled={!name.trim() || !slug || mutation.isPending}
          label={mutation.isPending ? "Creating…" : "Create space"}
          onPress={() => mutation.mutate()}
        />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  scroll: { alignSelf: "center", gap: 14, maxWidth: 700, padding: 18, width: "100%" },
  heading: { color: colors.ink, fontSize: 30, fontWeight: "700" },
  detail: { color: colors.muted, fontSize: 14 },
  error: { color: colors.danger, fontSize: 14, fontWeight: "600" },
});
