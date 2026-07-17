import { Gauge } from "lucide-react";
import type { components } from "../../api/schema.d.ts";
import { EFFORT_SUGGESTED_ICONS } from "../../lib/level-icons.ts";
import { useSettingsCommands } from "../../lib/mutations.ts";
import { spaceEffortLevelsQueryOptions } from "../../lib/queries.ts";
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
  const commands = useSettingsCommands();
  return (
    <SettingsSection
      icon={<Gauge className="size-5" />}
      title="Effort Levels"
      description="Define effort levels for estimating task complexity."
    >
      <OrderedNameListForm
        key={effortLevels.map((l) => `${l.name}:${l.icon ?? ""}`).join(",")}
        items={effortLevels}
        queryKey={spaceEffortLevelsQueryOptions(spaceSlug).queryKey}
        mutationFn={(items) => commands.replaceEffortLevels(spaceSlug, items)}
        itemLabel="Effort level"
        showIcons
        suggestedIcons={EFFORT_SUGGESTED_ICONS}
      />
    </SettingsSection>
  );
}
