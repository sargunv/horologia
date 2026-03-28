import { Gauge } from "lucide-react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { OrderedNameListForm } from "./OrderedNameListForm.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

type TaskEffortLevel = components["schemas"]["TaskEffortLevel"];

export function EffortLevelsSection({
  spaceSlug,
  effortLevels,
}: {
  spaceSlug: string;
  effortLevels: TaskEffortLevel[];
}) {
  return (
    <SettingsSection
      icon={<Gauge className="size-5" />}
      title="Effort Levels"
      description="Define effort levels for estimating task complexity."
    >
      <OrderedNameListForm
        key={effortLevels.map((l) => l.name).join(",")}
        items={effortLevels}
        queryKey={["spaces", spaceSlug, "effortLevels"]}
        mutationFn={async (items) => {
          const { data, error } = await apiClient.PUT("/spaces/{spaceSlug}/task-effort-levels", {
            params: { path: { spaceSlug } },
            body: { items },
          });
          if (error)
            throw new Error(
              (error as { message?: string }).message ?? "Failed to update effort levels",
            );
          if (!data) throw new Error("Failed to update effort levels");
          return data;
        }}
        itemLabel="Effort level"
      />
    </SettingsSection>
  );
}
