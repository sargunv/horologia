import {
  createLibraryCommands,
  slugifySpaceName,
  type HorologiaClient,
  type ServerProfile,
} from "@horologia/client-core";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button, Text } from "@expo/ui";
import { useRouter } from "expo-router";
import { useState } from "react";

import { useSession } from "@/auth/session-context";
import { FormField, FormSection } from "@/components/forms";
import { NativeFormScreen } from "@/components/native-screen";
import { ScreenState } from "@/components/screen-state";

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
    <NativeFormScreen>
      <FormSection title="New space">
        <Text>A home for related tasks, recipes, and people.</Text>
      </FormSection>
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
      {mutation.error ? <Text>{mutation.error.message}</Text> : null}
      <Button
        disabled={!name.trim() || !slug || mutation.isPending}
        label={mutation.isPending ? "Creating…" : "Create space"}
        onPress={() => mutation.mutate()}
        variant="filled"
      />
    </NativeFormScreen>
  );
}
