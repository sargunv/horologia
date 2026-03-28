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
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { GripVertical, ListChecks, Plus, Trash2 } from "lucide-react";
import { type FormEvent, type SetStateAction, useMemo, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { ErrorAlert } from "./ErrorAlert.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

type TaskStatus = components["schemas"]["TaskStatus"];
type TaskStatusCategory = components["schemas"]["TaskStatusCategory"];

interface StatusItem {
  id: string;
  name: string;
  category: TaskStatusCategory;
}

function toItems(statuses: TaskStatus[], category: TaskStatusCategory): StatusItem[] {
  return statuses
    .filter((s) => s.category === category)
    .map((s) => ({ id: crypto.randomUUID(), name: s.name, category }));
}

function arraysEqual(a: StatusItem[], b: TaskStatus[], category: TaskStatusCategory): boolean {
  const filtered = b.filter((s) => s.category === category);
  if (a.length !== filtered.length) return false;
  return a.every((item, i) => item.name === filtered[i]?.name);
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
        key={taskStatuses.map((s) => `${s.name}:${s.category}`).join(",")}
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

  const [initialItems, setInitialItems] = useState(() => toItems(taskStatuses, "initial"));
  const [intermediateItems, setIntermediateItems] = useState(() =>
    toItems(taskStatuses, "intermediate"),
  );
  const [completionItems, setCompletionItems] = useState(() => toItems(taskStatuses, "completion"));

  const hasChanges =
    !arraysEqual(initialItems, taskStatuses, "initial") ||
    !arraysEqual(intermediateItems, taskStatuses, "intermediate") ||
    !arraysEqual(completionItems, taskStatuses, "completion");

  const saveMutation = useMutation({
    mutationFn: async (items: { name: string; category: TaskStatusCategory }[]) => {
      const { data, error } = await apiClient.PUT("/spaces/{spaceSlug}/task-statuses", {
        params: { path: { spaceSlug } },
        body: { items },
      });
      if (error)
        throw new Error(
          (error as { message?: string }).message ?? "Failed to update task statuses",
        );
      return data;
    },
    onSuccess: async (data) => {
      setInitialItems(toItems(data.items, "initial"));
      setIntermediateItems(toItems(data.items, "intermediate"));
      setCompletionItems(toItems(data.items, "completion"));
      try {
        await queryClient.invalidateQueries({ queryKey: ["spaces", spaceSlug, "taskStatuses"] });
      } catch (err) {
        console.error("Failed to refresh after task statuses update:", err);
      }
    },
  });

  // Fix #3: Clear mutation error when user edits the form
  function wrapSetter<T>(setter: React.Dispatch<SetStateAction<T>>) {
    return (value: SetStateAction<T>) => {
      saveMutation.reset();
      setter(value);
    };
  }

  function validate(): string | null {
    const allItems = [...initialItems, ...intermediateItems, ...completionItems];
    for (const item of allItems) {
      const trimmed = item.name.trim();
      if (trimmed.length === 0) return "All statuses must have a name.";
      if (trimmed.length > 100) return "Status names must be 100 characters or fewer.";
    }
    const names = allItems.map((i) => i.name.trim().toLowerCase());
    const uniqueNames = new Set(names);
    if (uniqueNames.size !== names.length) return "Status names must be unique.";
    if (initialItems.length !== 1) return "There must be exactly one initial status.";
    if (completionItems.length < 1) return "There must be at least one completion status.";
    return null;
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const validationError = validate();
    if (validationError) return;
    const items = [...initialItems, ...intermediateItems, ...completionItems].map((item) => ({
      name: item.name.trim(),
      category: item.category,
    }));
    saveMutation.mutate(items);
  }

  const validationError = validate();
  const pending = saveMutation.isPending;

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-6">
      <CategoryGroup
        category="initial"
        items={initialItems}
        setItems={wrapSetter(setInitialItems)}
        canAdd={false}
        minItems={1}
        disabled={pending}
      />

      <CategoryGroup
        category="intermediate"
        items={intermediateItems}
        setItems={wrapSetter(setIntermediateItems)}
        canAdd={true}
        minItems={0}
        disabled={pending}
      />

      <CategoryGroup
        category="completion"
        items={completionItems}
        setItems={wrapSetter(setCompletionItems)}
        canAdd={true}
        minItems={1}
        disabled={pending}
      />

      {validationError && hasChanges && (
        <ErrorAlert key={validationError} message={validationError} />
      )}

      {saveMutation.error && <ErrorAlert message={saveMutation.error.message} />}

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={pending || !hasChanges || validationError !== null}
          className="btn preset-filled-primary-500"
        >
          {pending ? "Saving..." : "Save changes"}
        </button>
      </div>
    </form>
  );
}

function CategoryGroup({
  category,
  items,
  setItems,
  canAdd,
  minItems,
  disabled,
}: {
  category: TaskStatusCategory;
  items: StatusItem[];
  setItems: (value: SetStateAction<StatusItem[]>) => void;
  canAdd: boolean;
  minItems: number;
  disabled: boolean;
}) {
  // Fix #1: Add distance constraint so PointerSensor doesn't interfere with text inputs
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  // Fix #2: Custom DnD announcements using status names instead of UUIDs
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
      setItems((prev) => {
        const oldIndex = prev.findIndex((i) => i.id === active.id);
        const newIndex = prev.findIndex((i) => i.id === over.id);
        return arrayMove(prev, oldIndex, newIndex);
      });
    }
  }

  function handleAdd() {
    setItems((prev) => [...prev, { id: crypto.randomUUID(), name: "", category }]);
  }

  function handleRemove(id: string) {
    setItems((prev) => prev.filter((i) => i.id !== id));
  }

  function handleRename(id: string, name: string) {
    setItems((prev) => prev.map((i) => (i.id === id ? { ...i, name } : i)));
  }

  const canRemoveItem = items.length > minItems;

  return (
    <div className="flex flex-col gap-2">
      <div>
        <h3 className="text-sm font-medium">{CATEGORY_LABELS[category]}</h3>
        <p className="text-surface-600-400 text-xs">{CATEGORY_DESCRIPTIONS[category]}</p>
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
              <SortableStatusRow
                key={item.id}
                item={item}
                index={index}
                onRename={(name) => handleRename(item.id, name)}
                onRemove={canRemoveItem ? () => handleRemove(item.id) : undefined}
                disabled={disabled}
                draggable={items.length > 1}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>

      {canAdd && (
        <button
          type="button"
          onClick={handleAdd}
          disabled={disabled}
          className="btn btn-sm preset-outlined-surface-200-800 self-start text-xs"
        >
          <Plus className="size-3.5" aria-hidden="true" />
          Add status
        </button>
      )}
    </div>
  );
}

function SortableStatusRow({
  item,
  index,
  onRename,
  onRemove,
  disabled,
  draggable,
}: {
  item: StatusItem;
  index: number;
  onRename: (name: string) => void;
  onRemove?: (() => void) | undefined;
  disabled: boolean;
  draggable: boolean;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: item.id,
    disabled: !draggable || disabled,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  // Fix #4b: Strip redundant role from dnd-kit attributes (already a <button>)
  const { role: _role, ...handleAttributes } = attributes;

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`flex items-center gap-2 rounded-base ${isDragging ? "opacity-50" : ""}`}
    >
      {/* Fix #4a: Use preset-tonal-surface instead of hand-rolled colors */}
      <button
        type="button"
        className={`btn-icon btn-icon-sm shrink-0 ${draggable ? "preset-tonal-surface cursor-grab" : "cursor-default opacity-50"}`}
        disabled={!draggable || disabled}
        aria-label={`Drag to reorder ${item.name || "status"}`}
        {...handleAttributes}
        {...listeners}
      >
        <GripVertical className="size-4" aria-hidden="true" />
      </button>

      <input
        type="text"
        value={item.name}
        onChange={(e) => onRename(e.target.value)}
        className="input preset-outlined-surface-200-800 flex-1"
        placeholder="Status name"
        maxLength={100}
        required
        disabled={disabled}
        aria-label={`${CATEGORY_LABELS[item.category]} status name ${index + 1}`}
      />

      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          disabled={disabled}
          className="btn-icon btn-icon-sm preset-outlined-surface-200-800 shrink-0"
          aria-label={`Remove ${item.name || "status"}`}
        >
          <Trash2 className="size-3.5" aria-hidden="true" />
        </button>
      )}
    </div>
  );
}
