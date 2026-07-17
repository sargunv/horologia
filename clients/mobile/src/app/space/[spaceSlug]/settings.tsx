import {
  createLibraryCommands,
  createQueries,
  createSettingsCommands,
  type HorologiaClient,
  type ServerProfile,
} from "@horologia/client-core";
import type { components } from "@horologia/client-core/schema";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Alert, Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useSession } from "@/auth/session-context";
import { ActionButton, ChoiceChips, FormField, FormSection } from "@/components/forms";
import { ScreenState } from "@/components/screen-state";
import { colors } from "@/design/tokens";
import { parseLevels, parseStatuses } from "@/features/settings/ordered-input";

type Schema = components["schemas"];
type SpaceRole = Schema["SpaceRole"];
type SettingsCommands = ReturnType<typeof createSettingsCommands>;

const ROLES = [
  { value: "admin", label: "Admin" },
  { value: "member", label: "Member" },
  { value: "viewer", label: "Viewer" },
] as const;

export default function SpaceSettingsScreen() {
  const { spaceSlug } = useLocalSearchParams<{ spaceSlug: string }>();
  const session = useSession();
  if (!session.profile || !session.client || !spaceSlug) {
    return <ScreenState loading title="Opening settings" />;
  }
  return <SpaceSettings client={session.client} profile={session.profile} spaceSlug={spaceSlug} />;
}

function SpaceSettings({
  client,
  profile,
  spaceSlug,
}: {
  client: HorologiaClient;
  profile: ServerProfile;
  spaceSlug: string;
}) {
  const queries = useMemo(
    () => createQueries({ serverId: profile.id, apiClient: client, appClient: client }),
    [client, profile.id],
  );
  const space = useQuery(queries.spaceQueryOptions(spaceSlug));
  const members = useQuery(queries.spaceMembersQueryOptions(spaceSlug));
  const currentUser = useQuery(queries.currentUserQueryOptions);
  const statuses = useQuery(queries.spaceTaskStatusesQueryOptions(spaceSlug));
  const effort = useQuery(queries.spaceEffortLevelsQueryOptions(spaceSlug));
  const priority = useQuery(queries.spacePriorityLevelsQueryOptions(spaceSlug));
  const tags = useQuery(queries.spaceTagsQueryOptions(spaceSlug));
  const isAdmin =
    members.data?.some(
      (member) => member.userId === currentUser.data?.id && member.role === "admin",
    ) ?? false;
  const users = useQuery({ ...queries.usersQueryOptions, enabled: isAdmin });
  if (
    space.isPending ||
    members.isPending ||
    currentUser.isPending ||
    statuses.isPending ||
    effort.isPending ||
    priority.isPending ||
    tags.isPending
  ) {
    return <ScreenState loading title="Loading space settings" />;
  }
  if (
    space.isError ||
    members.isError ||
    currentUser.isError ||
    statuses.isError ||
    effort.isError ||
    priority.isError ||
    tags.isError
  ) {
    return <ScreenState detail="Try again when your server is reachable." title="Settings unavailable" />;
  }
  return (
    <SettingsContent
      client={client}
      currentUserId={currentUser.data.id}
      effort={effort.data}
      isAdmin={isAdmin}
      members={members.data}
      priority={priority.data}
      profile={profile}
      space={space.data}
      statuses={statuses.data}
      tags={tags.data}
      users={users.data ?? []}
    />
  );
}

function SettingsContent(props: {
  client: HorologiaClient;
  currentUserId: string;
  effort: Schema["TaskEffortLevel"][];
  isAdmin: boolean;
  members: Schema["SpaceMember"][];
  priority: Schema["TaskPriorityLevel"][];
  profile: ServerProfile;
  space: Schema["Space"];
  statuses: Schema["TaskStatus"][];
  tags: Schema["Tag"][];
  users: Schema["User"][];
}) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [notice, setNotice] = useState<string | null>(null);
  const commandContext = {
    serverId: props.profile.id,
    apiClient: props.client,
    queryClient,
    onCacheError: () => setNotice("Saved. Refresh if a list still looks stale."),
  };
  const settings = createSettingsCommands(commandContext);
  const library = createLibraryCommands(commandContext);
  const deletion = useMutation({
    mutationFn: () => library.deleteSpace(props.space.slug),
    onSuccess: () => router.replace("/(tabs)/library"),
  });
  return (
    <SafeAreaView edges={["left", "right", "bottom"]} style={styles.safeArea}>
      <ScrollView
        automaticallyAdjustKeyboardInsets
        contentContainerStyle={styles.scroll}
        keyboardDismissMode="on-drag"
        keyboardShouldPersistTaps="handled"
      >
        <Text accessibilityRole="header" style={styles.heading}>Space settings</Text>
        <Text style={styles.detail}>{props.space.name} · permissions are enforced by the server.</Text>
        <GeneralEditor
          key={props.space.updatedAt}
          commands={library}
          isAdmin={props.isAdmin}
          onRenamed={(slug) =>
            router.replace({ pathname: "/space/[spaceSlug]/settings", params: { spaceSlug: slug } })
          }
          space={props.space}
        />
        <MembersEditor
          commands={settings}
          currentUserId={props.currentUserId}
          isAdmin={props.isAdmin}
          members={props.members}
          spaceSlug={props.space.slug}
          users={props.users}
        />
        {props.isAdmin ? (
          <>
            <OrderedEditor
              hint="name | initial, intermediate, or completion | optional icon"
              initialValue={props.statuses.map((item) => `${item.name} | ${item.category} | ${item.icon}`).join("\n")}
              label="Statuses"
              save={(value) => settings.replaceTaskStatuses(props.space.slug, parseStatuses(value))}
            />
            <OrderedEditor
              hint="name | optional icon"
              initialValue={props.effort.map((item) => `${item.name} | ${item.icon}`).join("\n")}
              label="Effort levels"
              save={(value) => settings.replaceEffortLevels(props.space.slug, parseLevels(value))}
            />
            <OrderedEditor
              hint="name | optional icon"
              initialValue={props.priority.map((item) => `${item.name} | ${item.icon}`).join("\n")}
              label="Priority levels"
              save={(value) => settings.replacePriorityLevels(props.space.slug, parseLevels(value))}
            />
            <TagsEditor commands={settings} spaceSlug={props.space.slug} tags={props.tags} />
            <FormSection title="Danger zone">
              <Text style={styles.detail}>Delete this space and all of its current content.</Text>
              {deletion.error ? <ErrorText message={deletion.error.message} /> : null}
              <ActionButton
                destructive
                disabled={deletion.isPending}
                label={deletion.isPending ? "Deleting…" : "Delete space"}
                onPress={() =>
                  Alert.alert("Delete space?", "This cannot be undone.", [
                    { text: "Cancel", style: "cancel" },
                    { text: "Delete", style: "destructive", onPress: () => deletion.mutate() },
                  ])
                }
              />
            </FormSection>
          </>
        ) : (
          <Text style={styles.detail}>An administrator can change workflow settings and tags.</Text>
        )}
        {notice ? <Text style={styles.notice}>{notice}</Text> : null}
      </ScrollView>
    </SafeAreaView>
  );
}

function GeneralEditor({
  commands,
  isAdmin,
  onRenamed,
  space,
}: {
  commands: ReturnType<typeof createLibraryCommands>;
  isAdmin: boolean;
  onRenamed: (slug: string) => void;
  space: Schema["Space"];
}) {
  const [name, setName] = useState(space.name);
  const [slug, setSlug] = useState(space.slug);
  const [description, setDescription] = useState(space.description);
  const mutation = useMutation({
    mutationFn: () => commands.updateSpace(space.slug, { name, slug, description }),
    onSuccess: (saved) => onRenamed(saved.slug),
  });
  return (
    <FormSection title="General">
      <FormField editable={isAdmin} label="Name" onChangeText={setName} value={name} />
      <FormField autoCapitalize="none" editable={isAdmin} label="Slug" onChangeText={setSlug} value={slug} />
      <FormField editable={isAdmin} label="Description" multiline onChangeText={setDescription} value={description} />
      {mutation.error ? <ErrorText message={mutation.error.message} /> : null}
      {isAdmin ? (
        <ActionButton
          disabled={!name.trim() || !slug.trim() || mutation.isPending}
          label={mutation.isPending ? "Saving…" : "Save general settings"}
          onPress={() => mutation.mutate()}
        />
      ) : null}
    </FormSection>
  );
}

function MembersEditor({
  commands,
  currentUserId,
  isAdmin,
  members,
  spaceSlug,
  users,
}: {
  commands: SettingsCommands;
  currentUserId: string;
  isAdmin: boolean;
  members: Schema["SpaceMember"][];
  spaceSlug: string;
  users: Schema["User"][];
}) {
  const available = users.filter((user) => !members.some((member) => member.userId === user.id));
  const [userId, setUserId] = useState(available[0]?.id ?? "");
  const selectedUserId = available.some((user) => user.id === userId)
    ? userId
    : available[0]?.id ?? "";
  const [role, setRole] = useState<SpaceRole>("member");
  const add = useMutation({
    mutationFn: () => commands.createMember(spaceSlug, { userId: selectedUserId, role }),
  });
  return (
    <FormSection title="Members">
      {members.map((member) => (
        <MemberRow
          commands={commands}
          isAdmin={isAdmin}
          isSelf={member.userId === currentUserId}
          key={member.userId}
          member={member}
          spaceSlug={spaceSlug}
        />
      ))}
      {isAdmin && available.length ? (
        <View style={styles.subsection}>
          <ChoiceChips
            label="Person"
            onChange={setUserId}
            options={available.map((user) => ({ value: user.id, label: user.name }))}
            value={selectedUserId}
          />
          <ChoiceChips label="Role" onChange={setRole} options={ROLES} value={role} />
          {add.error ? <ErrorText message={add.error.message} /> : null}
          <ActionButton disabled={!selectedUserId || add.isPending} label={add.isPending ? "Adding…" : "Add member"} onPress={() => add.mutate()} />
        </View>
      ) : null}
    </FormSection>
  );
}

function MemberRow({
  commands,
  isAdmin,
  isSelf,
  member,
  spaceSlug,
}: {
  commands: SettingsCommands;
  isAdmin: boolean;
  isSelf: boolean;
  member: Schema["SpaceMember"];
  spaceSlug: string;
}) {
  const update = useMutation({
    mutationFn: (role: SpaceRole) => commands.updateMember(spaceSlug, member.userId, { role }),
  });
  const remove = useMutation({ mutationFn: () => commands.deleteMember(spaceSlug, member.userId) });
  return (
    <View style={styles.memberRow}>
      <Text style={styles.rowTitle}>{member.userName}{isSelf ? " (you)" : ""}</Text>
      <Text style={styles.meta}>{member.userEmail}</Text>
      {isAdmin ? (
        <>
          <ChoiceChips label={`Role for ${member.userName}`} onChange={(value) => update.mutate(value)} options={ROLES} value={member.role} />
          <Pressable
            accessibilityRole="button"
            disabled={remove.isPending}
            onPress={() => Alert.alert("Remove member?", member.userName, [
              { text: "Cancel", style: "cancel" },
              { text: "Remove", style: "destructive", onPress: () => remove.mutate() },
            ])}
          >
            <Text style={styles.dangerLink}>Remove</Text>
          </Pressable>
        </>
      ) : <Text style={styles.meta}>{member.role}</Text>}
      {update.error || remove.error ? <ErrorText message={(update.error ?? remove.error)?.message ?? "Member update failed"} /> : null}
    </View>
  );
}

function OrderedEditor({
  hint,
  initialValue,
  label,
  save,
}: {
  hint: string;
  initialValue: string;
  label: string;
  save: (value: string) => Promise<unknown>;
}) {
  const [value, setValue] = useState(initialValue);
  const mutation = useMutation({ mutationFn: () => save(value) });
  return (
    <FormSection title={label}>
      <Text style={styles.detail}>One per line in display order: {hint}.</Text>
      <FormField label={label} multiline onChangeText={setValue} value={value} />
      {mutation.error ? <ErrorText message={mutation.error.message} /> : null}
      <ActionButton disabled={mutation.isPending} label={mutation.isPending ? "Saving…" : `Save ${label.toLowerCase()}`} onPress={() => mutation.mutate()} />
    </FormSection>
  );
}

function TagsEditor({ commands, spaceSlug, tags }: { commands: SettingsCommands; spaceSlug: string; tags: Schema["Tag"][] }) {
  const [name, setName] = useState("");
  const create = useMutation({
    mutationFn: () => commands.createTag(spaceSlug, name.trim()),
    onSuccess: () => setName(""),
  });
  return (
    <FormSection title="Tags">
      {tags.map((tag) => <TagRow commands={commands} key={tag.name} spaceSlug={spaceSlug} tag={tag} />)}
      <FormField label="New tag" onChangeText={setName} testID="new-tag-input" value={name} />
      {create.error ? <ErrorText message={create.error.message} /> : null}
      <ActionButton disabled={!name.trim() || create.isPending} label={create.isPending ? "Adding…" : "Add tag"} onPress={() => create.mutate()} />
    </FormSection>
  );
}

function TagRow({ commands, spaceSlug, tag }: { commands: SettingsCommands; spaceSlug: string; tag: Schema["Tag"] }) {
  const [name, setName] = useState(tag.name);
  const rename = useMutation({ mutationFn: () => commands.updateTag(spaceSlug, tag.name, name.trim()) });
  const remove = useMutation({ mutationFn: () => commands.deleteTag(spaceSlug, tag.name) });
  return (
    <View style={styles.tagRow}>
      <FormField label={`Tag ${tag.name}`} onChangeText={setName} value={name} />
      <View style={styles.inlineActions}>
        <Pressable
          accessibilityLabel={`Rename tag ${tag.name}`}
          accessibilityRole="button"
          onPress={() => rename.mutate()}
        ><Text style={styles.link}>Rename</Text></Pressable>
        <Pressable
          accessibilityLabel={`Delete tag ${tag.name}`}
          accessibilityRole="button"
          onPress={() => remove.mutate()}
        ><Text style={styles.dangerLink}>Delete</Text></Pressable>
      </View>
      {rename.error || remove.error ? <ErrorText message={(rename.error ?? remove.error)?.message ?? "Tag update failed"} /> : null}
    </View>
  );
}

function ErrorText({ message }: { message: string }) {
  return <Text accessibilityRole="alert" style={styles.error}>{message}</Text>;
}

const styles = StyleSheet.create({
  safeArea: { backgroundColor: colors.canvas, flex: 1 },
  scroll: { alignSelf: "center", gap: 14, maxWidth: 800, padding: 18, paddingBottom: 54, width: "100%" },
  heading: { color: colors.ink, fontSize: 30, fontWeight: "700" },
  detail: { color: colors.muted, fontSize: 14, lineHeight: 20 },
  subsection: { borderTopColor: colors.outline, borderTopWidth: StyleSheet.hairlineWidth, gap: 12, paddingTop: 14 },
  memberRow: { borderBottomColor: colors.outline, borderBottomWidth: StyleSheet.hairlineWidth, gap: 8, paddingBottom: 13 },
  rowTitle: { color: colors.ink, fontSize: 16, fontWeight: "600" },
  meta: { color: colors.muted, fontSize: 13 },
  tagRow: { borderBottomColor: colors.outline, borderBottomWidth: StyleSheet.hairlineWidth, gap: 8, paddingBottom: 12 },
  inlineActions: { flexDirection: "row", gap: 20 },
  link: { color: colors.accent, fontSize: 14, fontWeight: "700" },
  dangerLink: { color: colors.danger, fontSize: 14, fontWeight: "700", paddingVertical: 4 },
  error: { color: colors.danger, fontSize: 13, fontWeight: "600", lineHeight: 18 },
  notice: { color: colors.accent, fontSize: 13, fontWeight: "600" },
});
