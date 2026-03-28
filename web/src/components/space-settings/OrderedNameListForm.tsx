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
import { GripVertical, Plus, Trash2 } from "lucide-react";
import { type FormEvent, type SetStateAction, useMemo, useState } from "react";
import { ErrorAlert } from "./ErrorAlert.tsx";

interface NamedItem {
  id: string;
  name: string;
}

interface ServerItem {
  name: string;
  position: number;
}

function toItems(serverItems: ServerItem[]): NamedItem[] {
  return serverItems.map((s) => ({ id: crypto.randomUUID(), name: s.name }));
}

function arraysEqual(a: NamedItem[], b: ServerItem[]): boolean {
  if (a.length !== b.length) return false;
  return a.every((item, i) => item.name === b[i]?.name);
}

export function OrderedNameListForm({
  items: serverItems,
  queryKey,
  mutationFn,
  itemLabel,
  minItems = 0,
}: {
  items: ServerItem[];
  queryKey: readonly unknown[];
  mutationFn: (names: { name: string }[]) => Promise<{ items: ServerItem[] }>;
  itemLabel: string;
  minItems?: number | undefined;
}) {
  const queryClient = useQueryClient();
  const [items, setItems] = useState(() => toItems(serverItems));

  const hasChanges = !arraysEqual(items, serverItems);

  const saveMutation = useMutation({
    mutationFn: async (body: { name: string }[]) => {
      return mutationFn(body);
    },
    onSuccess: async (data) => {
      setItems(toItems(data.items));
      try {
        await queryClient.invalidateQueries({ queryKey: [...queryKey] });
      } catch (err) {
        console.error("Failed to refresh after update:", err);
      }
    },
  });

  function wrapSetter(setter: React.Dispatch<SetStateAction<NamedItem[]>>) {
    return (value: SetStateAction<NamedItem[]>) => {
      saveMutation.reset();
      setter(value);
    };
  }

  function validate(): string | null {
    for (const item of items) {
      const trimmed = item.name.trim();
      if (trimmed.length === 0) return `All ${itemLabel.toLowerCase()}s must have a name.`;
      if (trimmed.length > 100) return `${itemLabel} names must be 100 characters or fewer.`;
    }
    const names = items.map((i) => i.name.trim().toLowerCase());
    const uniqueNames = new Set(names);
    if (uniqueNames.size !== names.length) return `${itemLabel} names must be unique.`;
    if (items.length < minItems)
      return `There must be at least ${minItems} ${itemLabel.toLowerCase()}${minItems === 1 ? "" : "s"}.`;
    return null;
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const validationError = validate();
    if (validationError) return;
    saveMutation.mutate(items.map((item) => ({ name: item.name.trim() })));
  }

  const validationError = validate();
  const pending = saveMutation.isPending;
  const wrappedSetItems = wrapSetter(setItems);

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <SortableNameList
        items={items}
        setItems={wrappedSetItems}
        disabled={pending}
        itemLabel={itemLabel}
        minItems={minItems}
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

function SortableNameList({
  items,
  setItems,
  disabled,
  itemLabel,
  minItems,
}: {
  items: NamedItem[];
  setItems: (value: SetStateAction<NamedItem[]>) => void;
  disabled: boolean;
  itemLabel: string;
  minItems: number;
}) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const announcements = useMemo(() => {
    const getName = (id: UniqueIdentifier) =>
      items.find((i) => i.id === id)?.name || itemLabel.toLowerCase();
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
  }, [items, itemLabel]);

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
    setItems((prev) => [...prev, { id: crypto.randomUUID(), name: "" }]);
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
                onRename={(name) => handleRename(item.id, name)}
                onRemove={canRemoveItem ? () => handleRemove(item.id) : undefined}
                disabled={disabled}
                draggable={items.length > 1}
                itemLabel={itemLabel}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>

      <button
        type="button"
        onClick={handleAdd}
        disabled={disabled}
        className="btn btn-sm preset-outlined-surface-200-800 self-start text-xs"
      >
        <Plus className="size-3.5" aria-hidden="true" />
        Add {itemLabel.toLowerCase()}
      </button>
    </div>
  );
}

function SortableNameRow({
  item,
  index,
  onRename,
  onRemove,
  disabled,
  draggable,
  itemLabel,
}: {
  item: NamedItem;
  index: number;
  onRename: (name: string) => void;
  onRemove?: (() => void) | undefined;
  disabled: boolean;
  draggable: boolean;
  itemLabel: string;
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
        aria-label={`Drag to reorder ${item.name || itemLabel.toLowerCase()}`}
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
        placeholder={`${itemLabel} name`}
        maxLength={100}
        required
        disabled={disabled}
        aria-label={`${itemLabel} name ${index + 1}`}
      />

      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          disabled={disabled}
          className="btn-icon btn-icon-sm preset-outlined-surface-200-800 shrink-0"
          aria-label={`Remove ${item.name || itemLabel.toLowerCase()}`}
        >
          <Trash2 className="size-3.5" aria-hidden="true" />
        </button>
      )}
    </div>
  );
}
