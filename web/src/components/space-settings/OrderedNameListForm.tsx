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
import { GripVertical, Pencil, Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";
import { EFFORT_SUGGESTED_ICONS } from "../../lib/level-icons.ts";
import { ErrorAlert } from "./ErrorAlert.tsx";
import { IconPicker } from "./IconPicker.tsx";

interface NamedItem {
  id: string;
  name: string;
  icon: string;
}

interface ServerItem {
  name: string;
  position: number;
  icon?: string;
}

function toItems(serverItems: ServerItem[], existing?: NamedItem[]): NamedItem[] {
  const nameToId = new Map(existing?.map((e) => [e.name, e.id]));
  return serverItems.map((s) => ({
    id: nameToId.get(s.name) ?? crypto.randomUUID(),
    name: s.name,
    icon: s.icon ?? "",
  }));
}

function arraysEqual(a: NamedItem[], b: ServerItem[]): boolean {
  if (a.length !== b.length) return false;
  return a.every(
    (item, i) => item.name.trim() === b[i]?.name && (item.icon || "") === (b[i]?.icon || ""),
  );
}

export function OrderedNameListForm({
  items: serverItems,
  queryKey,
  mutationFn,
  itemLabel,
  minItems = 0,
  showIcons = false,
  suggestedIcons = EFFORT_SUGGESTED_ICONS,
}: {
  items: ServerItem[];
  queryKey: readonly unknown[];
  mutationFn: (items: { name: string; icon?: string }[]) => Promise<{ items: ServerItem[] }>;
  itemLabel: string;
  minItems?: number | undefined;
  /** Show an icon picker for each item. */
  showIcons?: boolean;
  /** Which set of suggested icons to show in the icon picker. */
  suggestedIcons?: string[];
}) {
  const queryClient = useQueryClient();
  const [items, setItems] = useState(() => toItems(serverItems));
  const [validationError, setValidationError] = useState<string | null>(null);

  const saveMutation = useMutation({
    mutationFn,
    onSuccess: async (data) => {
      setItems((prev) => toItems(data.items, prev));
      await queryClient.invalidateQueries({ queryKey });
    },
  });

  function validate(itemsToValidate: NamedItem[]): string | null {
    for (const item of itemsToValidate) {
      const trimmed = item.name.trim();
      if (trimmed.length === 0) return `All ${itemLabel.toLowerCase()}s must have a name.`;
      if (trimmed.length > 100) return `${itemLabel} names must be 100 characters or fewer.`;
    }
    const names = itemsToValidate.map((i) => i.name.trim().toLowerCase());
    const uniqueNames = new Set(names);
    if (uniqueNames.size !== names.length) return `${itemLabel} names must be unique.`;
    if (itemsToValidate.length < minItems)
      return `There must be at least ${minItems} ${itemLabel.toLowerCase()}${minItems === 1 ? "" : "s"}.`;
    return null;
  }

  function saveIfChanged(nextItems: NamedItem[]) {
    const error = validate(nextItems);
    setValidationError(error);
    if (error) return;
    if (arraysEqual(nextItems, serverItems)) return;
    saveMutation.mutate(
      nextItems.map((item) => ({
        name: item.name.trim(),
        ...(showIcons ? { icon: item.icon || "" } : {}),
      })),
    );
  }

  function clearErrors() {
    setValidationError(null);
    saveMutation.reset();
  }

  const pending = saveMutation.isPending;

  return (
    <div className="flex flex-col gap-4">
      <SortableNameList
        items={items}
        setItems={(value) => {
          clearErrors();
          setItems(value);
        }}
        onSave={saveIfChanged}
        disabled={pending}
        itemLabel={itemLabel}
        minItems={minItems}
        showIcons={showIcons}
        suggestedIcons={suggestedIcons}
      />

      {validationError && <ErrorAlert key={validationError} message={validationError} />}
      {saveMutation.error && <ErrorAlert message={saveMutation.error.message} />}
    </div>
  );
}

function SortableNameList({
  items,
  setItems,
  onSave,
  disabled,
  itemLabel,
  minItems,
  showIcons,
  suggestedIcons,
}: {
  items: NamedItem[];
  setItems: (value: NamedItem[]) => void;
  onSave: (items: NamedItem[]) => void;
  disabled: boolean;
  itemLabel: string;
  minItems: number;
  showIcons: boolean;
  suggestedIcons: string[];
}) {
  const [editingId, setEditingId] = useState<string | null>(null);

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
      const oldIndex = items.findIndex((i) => i.id === active.id);
      const newIndex = items.findIndex((i) => i.id === over.id);
      const reordered = arrayMove(items, oldIndex, newIndex);
      setItems(reordered);
      onSave(reordered);
    }
  }

  function handleAdd() {
    const newId = crypto.randomUUID();
    setItems([...items, { id: newId, name: "", icon: "" }]);
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

  function handleIconChange(id: string, icon: string) {
    const next = items.map((i) => (i.id === id ? { ...i, icon } : i));
    setItems(next);
    onSave(next);
  }

  function handleEndEdit() {
    setEditingId(null);
    // Remove items with empty names (cancelled adds)
    const cleaned = items.filter((i) => i.name.trim());
    if (cleaned.length !== items.length) {
      setItems(cleaned);
    }
    onSave(cleaned);
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
                isEditing={editingId === item.id}
                onStartEdit={() => setEditingId(item.id)}
                onEndEdit={handleEndEdit}
                onRename={(name) => handleRename(item.id, name)}
                onIconChange={
                  showIcons ? (icon: string) => handleIconChange(item.id, icon) : undefined
                }
                suggestedIcons={suggestedIcons}
                onRemove={canRemoveItem ? () => handleRemove(item.id) : undefined}
                disabled={disabled}
                draggable={items.length > 1}
                itemLabel={itemLabel}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>

      {!(editingId && items.find((i) => i.id === editingId && !i.name.trim())) && (
        <button
          type="button"
          onClick={handleAdd}
          disabled={disabled}
          className="btn btn-sm preset-outlined-surface-200-800 self-start text-xs"
        >
          <Plus className="size-3.5" aria-hidden="true" />
          Add {itemLabel.toLowerCase()}
        </button>
      )}
    </div>
  );
}

function SortableNameRow({
  item,
  index,
  isEditing,
  onStartEdit,
  onEndEdit,
  onRename,
  onIconChange,
  suggestedIcons,
  onRemove,
  disabled,
  draggable,
  itemLabel,
}: {
  item: NamedItem;
  index: number;
  isEditing: boolean;
  onStartEdit: () => void;
  onEndEdit: () => void;
  onRename: (name: string) => void;
  onIconChange?: ((icon: string) => void) | undefined;
  suggestedIcons: string[];
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
      className={`flex items-center gap-1 rounded-base ${isDragging ? "opacity-50" : ""}`}
    >
      <button
        type="button"
        className={`btn-icon btn-icon-sm shrink-0 ${draggable ? "preset-tonal-surface cursor-grab" : "cursor-default opacity-50"}`}
        disabled={!draggable || disabled}
        aria-label={`Drag to reorder ${item.name || `${itemLabel.toLowerCase()} ${index + 1}`}`}
        {...(draggable && !disabled ? { ...attributes, ...listeners } : {})}
      >
        <GripVertical className="size-4" aria-hidden="true" />
      </button>

      {onIconChange && (
        <IconPicker
          value={item.icon || undefined}
          onChange={onIconChange}
          disabled={disabled}
          label={`Icon for ${item.name || `${itemLabel.toLowerCase()} ${index + 1}`}`}
          suggestedIcons={suggestedIcons}
        />
      )}

      {isEditing ? (
        <input
          type="text"
          value={item.name}
          onChange={(e) => onRename(e.target.value)}
          onBlur={onEndEdit}
          onKeyDown={handleKeyDown}
          className="input preset-outlined-surface-200-800 flex-1"
          placeholder={`${itemLabel} name`}
          maxLength={100}
          required
          disabled={disabled}
          aria-label={`${itemLabel} name ${index + 1}`}
          autoFocus
        />
      ) : (
        <button
          type="button"
          onClick={onStartEdit}
          disabled={disabled}
          className="flex flex-1 items-center gap-2 truncate rounded-base px-3 py-2 text-left text-sm hover:bg-surface-200-800"
          aria-label={`Edit ${item.name || itemLabel.toLowerCase()}`}
        >
          <span className="flex-1 truncate">{item.name || `${itemLabel} name`}</span>
          <Pencil className="text-surface-600-400 size-3.5 shrink-0" aria-hidden="true" />
        </button>
      )}

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
