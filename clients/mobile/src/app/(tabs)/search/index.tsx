import { createQueries } from "@horologia/client-core";
import { useQuery } from "@tanstack/react-query";
import { FieldGroup, Host, ListItem, Text } from "@expo/ui";
import { useIsFocused, useRouter } from "expo-router";
import { useMemo, useState } from "react";

import { useSession } from "@/auth/session-context";
import { FormField } from "@/components/forms";
import { ScreenState } from "@/components/screen-state";

export default function SearchScreen() {
  const isFocused = useIsFocused();
  const session = useSession();
  if (!isFocused) return null;
  if (!session.profile || !session.client) return <ScreenState loading title="Opening search" />;
  return <AuthenticatedSearch client={session.client} serverId={session.profile.id} />;
}

function AuthenticatedSearch({
  client,
  serverId,
}: {
  client: NonNullable<ReturnType<typeof useSession>["client"]>;
  serverId: string;
}) {
  const router = useRouter();
  const [search, setSearch] = useState("");
  const queries = useMemo(() => createQueries({ serverId, apiClient: client }), [client, serverId]);
  const enabled = search.trim().length >= 2;
  const tasks = useQuery({ ...queries.taskSearchQueryOptions({ query: search }), enabled });
  const recipes = useQuery({ ...queries.recipeSearchQueryOptions({ query: search }), enabled });

  return (
    <Host style={{ flex: 1 }} useViewportSizeMeasurement>
      <FieldGroup>
        <FieldGroup.Section title="Search">
          <FormField
            autoCapitalize="none"
            autoCorrect={false}
            onChangeText={setSearch}
            label="Tasks and recipes across every space"
            value={search}
          />
          {!enabled ? <Text>Type at least two characters.</Text> : null}
        </FieldGroup.Section>
        {enabled ? (
          <>
            <ResultSection title="Tasks">
              {(tasks.data ?? []).map((task) => (
                <Result
                  key={`${task.spaceSlug}/${task.id}`}
                  meta={`${task.spaceSlug} · ${task.status}`}
                  onPress={() =>
                    router.push({
                      pathname: "/task/[spaceSlug]/[taskId]",
                      params: { spaceSlug: task.spaceSlug, taskId: task.id },
                    })
                  }
                  title={task.title}
                />
              ))}
              {!tasks.isPending && tasks.data?.length === 0 ? (
                <Text>No matching tasks.</Text>
              ) : null}
            </ResultSection>
            <ResultSection title="Recipes">
              {(recipes.data ?? []).map((recipe) => (
                <Result
                  key={`${recipe.spaceSlug}/${recipe.id}`}
                  meta={recipe.spaceSlug}
                  onPress={() =>
                    router.push({
                      pathname: "/recipe/[spaceSlug]/[recipeId]",
                      params: { spaceSlug: recipe.spaceSlug, recipeId: recipe.id },
                    })
                  }
                  title={recipe.name}
                />
              ))}
              {!recipes.isPending && recipes.data?.length === 0 ? (
                <Text>No matching recipes.</Text>
              ) : null}
            </ResultSection>
          </>
        ) : null}
      </FieldGroup>
    </Host>
  );
}

function ResultSection({ title, children }: { title: string; children: React.ReactNode }) {
  return <FieldGroup.Section title={title}>{children}</FieldGroup.Section>;
}

function Result({ title, meta, onPress }: { title: string; meta: string; onPress: () => void }) {
  return (
    <ListItem onPress={onPress} supportingText={meta}>
      <Text>{title}</Text>
    </ListItem>
  );
}
