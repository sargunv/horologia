import { useLocalSearchParams } from "expo-router";

import { useSession } from "@/auth/session-context";
import { ScreenState } from "@/components/screen-state";
import { TaskEditor } from "@/components/task-editor";

export default function NewTaskScreen() {
  const { spaceSlug } = useLocalSearchParams<{ spaceSlug?: string }>();
  const session = useSession();
  if (!session.profile || !session.accountId || !session.client) {
    return <ScreenState loading title="Opening editor" />;
  }
  return (
    <TaskEditor
      accountId={session.accountId}
      client={session.client}
      {...(spaceSlug ? { initialSpaceSlug: spaceSlug } : {})}
      profile={session.profile}
    />
  );
}
