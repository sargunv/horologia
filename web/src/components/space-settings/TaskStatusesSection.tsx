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
import { GripVertical, ListChecks, Pencil, Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";
import { apiClient } from "../../api/client.ts";
import type { components } from "../../api/schema.d.ts";
import { spaceTaskStatusesQueryOptions } from "../../lib/queries.ts";
import { ErrorAlert } from "./ErrorAlert.tsx";
import { SettingsSection } from "./SettingsSection.tsx";

type TaskStatus = components["schemas"]["TaskStatus"];
type TaskStatusCategory = components["schemas"]["TaskStatusCategory"];

interface StatusItem {
  id: string;
  name: string;
  category: TaskStatusCategory;
}

function toItems(
  statuses: TaskStatus[],
  category: TaskStatusCategory,
  existing?: StatusItem[],
): StatusItem[] {
  return statuses
    .filter((s) => s.category === category)
    .map((s, i) => ({ id: existing?.[i]?.id ?? crypto.randomUUID(), name: s.name, category }));
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
      item.name.trim() === serverStatuses[i]?.name && item.category === serverStatuses[i]?.category,
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
  const queryKey = spaceTaskStatusesQueryOptions(spaceSlug).queryKey;

  const [initialItems, setInitialItems] = useState(() => toItems(taskStatuses, "initial"));
  const [intermediateItems, setIntermediateItems] = useState(() =>
    toItems(taskStatuses, "intermediate"),
  );
  const [completionItems, setCompletionItems] = useState(() => toItems(taskStatuses, "completion"));
  const [validationError, setValidationError] = useState<string | null>(null);

  const saveMutation = useMutation({
    mutationFn: async (items: { name: string; category: TaskStatusCategory }[]) => {
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
      await queryClient.invalidateQueries({ queryKey });
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
    setItems([...items, { id: newId, name: "", category }]);
    setEditingId(newId);
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
                isEditing={editingId === item.id}
                onStartEdit={() => setEditingId(item.id)}
                onEndEdit={handleEndEdit}
                onRename={(name) => handleRename(item.id, name)}
                onRemove={canRemoveItem ? () => handleRemove(item.id) : undefined}
                disabled={disabled}
                draggable={items.length > 1}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>

      {canAdd && !editingId && (
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
  isEditing,
  onStartEdit,
  onEndEdit,
  onRename,
  onRemove,
  disabled,
  draggable,
}: {
  item: StatusItem;
  index: number;
  isEditing: boolean;
  onStartEdit: () => void;
  onEndEdit: () => void;
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

  const { role: _role, ...handleAttributes } = attributes;

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" || e.key === "Escape") {
      e.preventDefault();
      e.currentTarget.blur();
    }
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`flex items-center gap-2 rounded-base ${isDragging ? "opacity-50" : ""}`}
    >
      <button
        type="button"
        className={`btn-icon btn-icon-sm shrink-0 ${draggable ? "preset-tonal-surface cursor-grab" : "cursor-default opacity-50"}`}
        disabled={!draggable || disabled}
        aria-label={`Drag to reorder ${item.name || `status ${index + 1}`}`}
        {...(draggable && !disabled ? { ...handleAttributes, ...listeners } : {})}
      >
        <GripVertical className="size-4" aria-hidden="true" />
      </button>

      {isEditing ? (
        <input
          type="text"
          value={item.name}
          onChange={(e) => onRename(e.target.value)}
          onBlur={onEndEdit}
          onKeyDown={handleKeyDown}
          className="input preset-outlined-surface-200-800 flex-1"
          placeholder="Status name"
          maxLength={100}
          required
          disabled={disabled}
          aria-label={`${CATEGORY_LABELS[item.category]} status name ${index + 1}`}
          autoFocus
        />
      ) : (
        <button
          type="button"
          onClick={onStartEdit}
          disabled={disabled}
          className="flex flex-1 items-center gap-2 truncate rounded-base px-3 py-2 text-left text-sm hover:bg-surface-200-800"
          aria-label={`Edit ${item.name || "status"}`}
        >
          <span className="flex-1 truncate">{item.name || "Status name"}</span>
          <Pencil className="text-surface-600-400 size-3.5 shrink-0" aria-hidden="true" />
        </button>
      )}

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
