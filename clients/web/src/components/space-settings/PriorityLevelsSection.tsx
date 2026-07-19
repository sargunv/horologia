import { SignalHigh } from "lucide-react";
import type { components } from "@horologia/client-core/schema";
import { PRIORITY_SUGGESTED_ICONS } from "../../lib/level-icons.ts";
import { useSettingsCommands } from "../../lib/mutations.ts";
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
  const commands = useSettingsCommands();
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
        mutationFn={(items) => commands.replacePriorityLevels(spaceSlug, items)}
        itemLabel="Priority level"
        showIcons
        suggestedIcons={PRIORITY_SUGGESTED_ICONS}
      />
    </SettingsSection>
  );
}
