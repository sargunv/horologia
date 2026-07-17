import {
  createQueries,
  type HorologiaClient,
  type ServerProfile,
} from "@horologia/client-core";
import { useQuery } from "@tanstack/react-query";
import { useLocalSearchParams } from "expo-router";
import { useMemo } from "react";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";
import { TaskEditor } from "@/components/task-editor";

export default function EditTaskScreen() {
  const { spaceSlug, taskId } = useLocalSearchParams<{ spaceSlug: string; taskId: string }>();
  const session = useSession();
  if (!session.profile || !session.accountId || !session.client || !spaceSlug || !taskId) {
    return <ScreenState loading title="Opening editor" />;
  }
  return (
    <AuthenticatedEditor
      accountId={session.accountId}
      client={session.client}
      profile={session.profile}
      spaceSlug={spaceSlug}
      taskId={taskId}
    />
  );
}

function AuthenticatedEditor({
  accountId,
  client,
  profile,
  spaceSlug,
  taskId,
}: {
  accountId: string;
  client: HorologiaClient;
  profile: ServerProfile;
  spaceSlug: string;
  taskId: string;
}) {
  const queries = useMemo(
    () => createQueries({ serverId: profile.id, apiClient: client, appClient: client }),
    [client, profile.id],
  );
  const taskQuery = useQuery(queries.spaceTaskQueryOptions(spaceSlug, taskId));
  if (taskQuery.isPending) return <ScreenState loading title="Opening editor" />;
  if (taskQuery.isError || !taskQuery.data) {
    return <ScreenState detail="The task could not be loaded." title="Task unavailable" />;
  }
  return (
    <TaskEditor
      accountId={accountId}
      client={client}
      profile={profile}
      task={taskQuery.data}
    />
  );
}
