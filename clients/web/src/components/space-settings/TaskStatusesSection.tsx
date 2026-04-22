import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
  type UniqueIdentifier,
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ListChecks, Plus } from "lucide-react";
import { useMemo, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { STATUS_SUGGESTED_ICONS } from "../../lib/level-icons.ts";
import { spaceTaskStatusesQueryOptions } from "../../lib/queries.ts";
import { notifyStaleData } from "../../lib/toaster.ts";
import { SortableNameRow } from "./OrderedNameListForm.tsx";
import { ErrorAlert } from "./ErrorAlert.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

type TaskStatus = components["schemas"]["TaskStatus"];
type TaskStatusCategory = components["schemas"]["TaskStatusCategory"];

interface StatusItem {
  id: string;
  name: string;
  category: TaskStatusCategory;
  icon: string;
}

function toItems(
  statuses: TaskStatus[],
  category: TaskStatusCategory,
  existing?: StatusItem[],
): StatusItem[] {
  const nameToId = new Map(existing?.map((e) => [e.name, e.id]));
  return statuses
    .filter((s) => s.category === category)
    .map((s) => ({
      id: nameToId.get(s.name) ?? crypto.randomUUID(),
      name: s.name,
      category,
      icon: s.icon ?? "",
    }));
}

function allEqual(
  initial: StatusItem[],
  intermediate: StatusItem[],
  completion: StatusItem[],
  serverStatuses: TaskStatus[],
): boolean {
  const merged = [...initial, ...intermediate, ...completion];
  if (merged.length !== serverStatuses.length) return false;
  return merged.every(
    (item, i) =>
      item.name.trim() === serverStatuses[i]?.name &&
      item.category === serverStatuses[i]?.category &&
      (item.icon || "") === (serverStatuses[i]?.icon || ""),
  );
}

const CATEGORY_LABELS: Record<TaskStatusCategory, string> = {
  initial: "Initial",
  intermediate: "Intermediate",
  completion: "Completion",
};

const CATEGORY_DESCRIPTIONS: Record<TaskStatusCategory, string> = {
  initial: "New tasks start here. Exactly one status.",
  intermediate: "Workflow states with no special semantics.",
  completion: "Triggers recurrence reset, staleness reset, and rotation advance.",
};

export function TaskStatusesSection({
  spaceSlug,
  taskStatuses,
}: {
  spaceSlug: string;
  taskStatuses: TaskStatus[];
}) {
  return (
    <SettingsSection
      icon={<ListChecks className="size-5" />}
      title="Task Statuses"
      description="Configure the workflow statuses for tasks in this space."
    >
      <TaskStatusesForm
        key={taskStatuses.map((s) => `${s.name}:${s.category}:${s.icon}`).join(",")}
        spaceSlug={spaceSlug}
        taskStatuses={taskStatuses}
      />
    </SettingsSection>
  );
}

function TaskStatusesForm({
  spaceSlug,
  taskStatuses,
}: {
  spaceSlug: string;
  taskStatuses: TaskStatus[];
}) {
  const queryClient = useQueryClient();
  const queryKey = spaceTaskStatusesQueryOptions(spaceSlug).queryKey;

  const [initialItems, setInitialItems] = useState(() => toItems(taskStatuses, "initial"));
  const [intermediateItems, setIntermediateItems] = useState(() =>
    toItems(taskStatuses, "intermediate"),
  );
  const [completionItems, setCompletionItems] = useState(() => toItems(taskStatuses, "completion"));
  const [validationError, setValidationError] = useState<string | null>(null);

  const saveMutation = useMutation({
    mutationFn: async (items: { name: string; category: TaskStatusCategory; icon?: string }[]) => {
      const { data, error } = await apiClient.PUT("/spaces/{spaceSlug}/task-statuses", {
        params: { path: { spaceSlug } },
        body: { items },
      });
      if (error) throw new Error(error.message ?? "Failed to update task statuses");
      return data;
    },
    onSuccess: async (data) => {
      setInitialItems((prev) => toItems(data.items, "initial", prev));
      setIntermediateItems((prev) => toItems(data.items, "intermediate", prev));
      setCompletionItems((prev) => toItems(data.items, "completion", prev));
      try {
        await queryClient.invalidateQueries({ queryKey });
      } catch (err) {
        console.error("Cache invalidation failed after mutation:", err);
        notifyStaleData();
      }
    },
  });

  function validate(
    initial: StatusItem[],
    intermediate: StatusItem[],
    completion: StatusItem[],
  ): string | null {
    const allItems = [...initial, ...intermediate, ...completion];
    for (const item of allItems) {
      const trimmed = item.name.trim();
      if (trimmed.length === 0) return "All statuses must have a name.";
      if (trimmed.length > 100) return "Status names must be 100 characters or fewer.";
    }
    const names = allItems.map((i) => i.name.trim().toLowerCase());
    const uniqueNames = new Set(names);
    if (uniqueNames.size !== names.length) return "Status names must be unique.";
    if (initial.length !== 1) return "There must be exactly one initial status.";
    if (completion.length < 1) return "There must be at least one completion status.";
    return null;
  }

  function handleCategorySave(category: TaskStatusCategory, updatedItems: StatusItem[]) {
    const initial = category === "initial" ? updatedItems : initialItems;
    const intermediate = category === "intermediate" ? updatedItems : intermediateItems;
    const completion = category === "completion" ? updatedItems : completionItems;

    const error = validate(initial, intermediate, completion);
    setValidationError(error);
    if (error) return;
    if (allEqual(initial, intermediate, completion, taskStatuses)) return;

    saveMutation.mutate(
      [...initial, ...intermediate, ...completion].map((item) => ({
        name: item.name.trim(),
        category: item.category,
        icon: item.icon || "",
      })),
    );
  }

  function clearErrors() {
    setValidationError(null);
    saveMutation.reset();
  }

  const pending = saveMutation.isPending;

  return (
    <div className="flex flex-col gap-6">
      <CategoryGroup
        category="initial"
        items={initialItems}
        setItems={(value) => {
          clearErrors();
          setInitialItems(value);
        }}
        onSave={(items) => handleCategorySave("initial", items)}
        canAdd={false}
        minItems={1}
        disabled={pending}
      />

      <CategoryGroup
        category="intermediate"
        items={intermediateItems}
        setItems={(value) => {
          clearErrors();
          setIntermediateItems(value);
        }}
        onSave={(items) => handleCategorySave("intermediate", items)}
        canAdd={true}
        minItems={0}
        disabled={pending}
      />

      <CategoryGroup
        category="completion"
        items={completionItems}
        setItems={(value) => {
          clearErrors();
          setCompletionItems(value);
        }}
        onSave={(items) => handleCategorySave("completion", items)}
        canAdd={true}
        minItems={1}
        disabled={pending}
      />

      {validationError && <ErrorAlert key={validationError} message={validationError} />}
      {saveMutation.error && <ErrorAlert message={saveMutation.error.message} />}
    </div>
  );
}

function CategoryGroup({
  category,
  items,
  setItems,
  onSave,
  canAdd,
  minItems,
  disabled,
}: {
  category: TaskStatusCategory;
  items: StatusItem[];
  setItems: (value: StatusItem[]) => void;
  onSave: (items: StatusItem[]) => void;
  canAdd: boolean;
  minItems: number;
  disabled: boolean;
}) {
  const [editingId, setEditingId] = useState<string | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const announcements = useMemo(() => {
    const getName = (id: UniqueIdentifier) => items.find((i) => i.id === id)?.name || "status";
    return {
      onDragStart({ active }: { active: { id: UniqueIdentifier } }) {
        return `Picked up ${getName(active.id)}.`;
      },
      onDragOver({
        active,
        over,
      }: {
        active: { id: UniqueIdentifier };
        over: { id: UniqueIdentifier } | null;
      }) {
        if (over) return `${getName(active.id)} is now over ${getName(over.id)}.`;
        return `${getName(active.id)} is no longer over a drop target.`;
      },
      onDragEnd({
        active,
        over,
      }: {
        active: { id: UniqueIdentifier };
        over: { id: UniqueIdentifier } | null;
      }) {
        if (over)
          return `${getName(active.id)} was dropped at the position of ${getName(over.id)}.`;
        return `${getName(active.id)} was dropped.`;
      },
      onDragCancel({ active }: { active: { id: UniqueIdentifier } }) {
        return `Reordering cancelled. ${getName(active.id)} was returned to its original position.`;
      },
    };
  }, [items]);

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (over && active.id !== over.id) {
      const oldIndex = items.findIndex((i) => i.id === active.id);
      const newIndex = items.findIndex((i) => i.id === over.id);
      const reordered = arrayMove(items, oldIndex, newIndex);
      setItems(reordered);
      onSave(reordered);
    }
  }

  function handleAdd() {
    const newId = crypto.randomUUID();
    setItems([...items, { id: newId, name: "", category, icon: "" }]);
    setEditingId(newId);
  }

  function handleIconChange(id: string, icon: string) {
    const next = items.map((i) => (i.id === id ? { ...i, icon } : i));
    setItems(next);
    onSave(next);
  }

  function handleRemove(id: string) {
    if (editingId === id) setEditingId(null);
    const next = items.filter((i) => i.id !== id);
    setItems(next);
    onSave(next);
  }

  function handleRename(id: string, name: string) {
    setItems(items.map((i) => (i.id === id ? { ...i, name } : i)));
  }

  function handleEndEdit() {
    setEditingId(null);
    const cleaned = items.filter((i) => i.name.trim());
    if (cleaned.length !== items.length) {
      setItems(cleaned);
    }
    onSave(cleaned);
  }

  const canRemoveItem = items.length > minItems;

  return (
    <div className="flex flex-col gap-2">
      <div>
        <h3 className="text-sm font-medium">{CATEGORY_LABELS[category]}</h3>
        <p className="text-base-content/70 text-xs">{CATEGORY_DESCRIPTIONS[category]}</p>
      </div>

      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
        accessibility={{ announcements }}
      >
        <SortableContext items={items.map((i) => i.id)} strategy={verticalListSortingStrategy}>
          <div className="flex flex-col gap-1">
            {items.map((item, index) => (
              <SortableNameRow
                key={item.id}
                item={item}
                index={index}
                isEditing={editingId === item.id}
                onStartEdit={() => setEditingId(item.id)}
                onEndEdit={handleEndEdit}
                onRename={(name) => handleRename(item.id, name)}
                onIconChange={(icon: string) => handleIconChange(item.id, icon)}
                suggestedIcons={STATUS_SUGGESTED_ICONS}
                onRemove={canRemoveItem ? () => handleRemove(item.id) : undefined}
                disabled={disabled}
                draggable={items.length > 1}
                itemLabel="Status"
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>

      {canAdd && !(editingId && items.find((i) => i.id === editingId && !i.name.trim())) && (
        <button
          type="button"
          onClick={handleAdd}
          disabled={disabled}
          className="btn btn-sm btn-soft self-start text-xs"
        >
          <Plus className="size-3.5" aria-hidden="true" />
          Add status
        </button>
      )}
    </div>
  );
}
