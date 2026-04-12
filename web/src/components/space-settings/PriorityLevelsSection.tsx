import { SignalHigh } from "lucide-react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { spacePriorityLevelsQueryOptions } from "../../lib/queries.ts";
import { OrderedNameListForm } from "./OrderedNameListForm.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

type TaskPriorityLevel = components["schemas"]["TaskPriorityLevel"];

export function PriorityLevelsSection({
  spaceSlug,
  priorityLevels,
}: {
  spaceSlug: string;
  priorityLevels: TaskPriorityLevel[];
}) {
  return (
    <SettingsSection
      icon={<SignalHigh className="size-5" />}
      title="Priority Levels"
      description="Configure priority levels for organizing tasks."
    >
      <OrderedNameListForm
        key={priorityLevels.map((l) => `${l.name}:${l.icon ?? ""}`).join(",")}
        items={priorityLevels}
        queryKey={spacePriorityLevelsQueryOptions(spaceSlug).queryKey}
        mutationFn={async (items) => {
          const { data, error } = await apiClient.PUT("/spaces/{spaceSlug}/task-priority-levels", {
            params: { path: { spaceSlug } },
            body: { items },
          });
          if (error) throw new Error(error.message ?? "Failed to update priority levels");
          if (!data) throw new Error("Failed to update priority levels");
          return data;
        }}
        itemLabel="Priority level"
        showIcons
      />
    </SettingsSection>
  );
}
