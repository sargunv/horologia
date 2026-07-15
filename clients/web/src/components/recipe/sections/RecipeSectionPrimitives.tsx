import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { sortableKeyboardCoordinates, useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVertical, Plus, Trash2 } from "lucide-react";
import type { FocusEvent, ReactNode } from "react";
import type { SortableData } from "./recipeSectionDnd.ts";

export function focusStayedInside(event: FocusEvent<HTMLElement>): boolean {
  return event.relatedTarget instanceof Node && event.currentTarget.contains(event.relatedTarget);
}

export function AddRow({
  label,
  onClick,
  disabled,
  variant = "empty",
}: {
  label: string;
  onClick: () => void;
  disabled: boolean;
  variant?: "empty" | "inline";
}) {
  return (
    <button
      type="button"
      className={
        variant === "inline"
          ? "flex w-full items-center justify-center gap-2 border-t border-base-300 px-3 py-2 text-sm text-base-content/50 transition-colors hover:bg-base-200 hover:text-base-content/80 disabled:opacity-50"
          : "flex w-full items-center justify-center gap-2 rounded-box border-2 border-dashed border-base-300 p-3 text-sm text-base-content/60 transition-colors hover:border-base-content/40 hover:text-base-content/80 disabled:opacity-50"
      }
      onClick={onClick}
      disabled={disabled}
    >
      <Plus className="size-4" aria-hidden="true" />
      {label}
    </button>
  );
}

export function AddSectionButton({
  onClick,
  disabled,
}: {
  onClick: () => void;
  disabled: boolean;
}) {
  return (
    <button
      type="button"
      className="btn btn-ghost btn-sm gap-1.5 px-2 font-normal text-base-content/60"
      onClick={onClick}
      disabled={disabled}
    >
      <Plus className="size-3.5" aria-hidden="true" />
      Add section
    </button>
  );
}

export function DeleteButton({
  label,
  pending,
  onDelete,
  visible = false,
}: {
  label: string;
  pending: boolean;
  onDelete: () => void;
  visible?: boolean;
}) {
  return (
    <button
      type="button"
      className={`btn btn-ghost btn-square btn-xs shrink-0 text-base-content/35 hover:text-error ${
        visible ? "opacity-100" : "opacity-0 group-hover:opacity-100 focus:opacity-100"
      }`}
      aria-label={`Delete ${label}`}
      disabled={pending}
      onMouseDown={(event) => event.preventDefault()}
      onClick={onDelete}
    >
      <Trash2 className="size-3.5" aria-hidden="true" />
    </button>
  );
}

export function SortableRoot({
  onDragEnd,
  children,
}: {
  onDragEnd: (event: DragEndEvent) => void;
  children: ReactNode;
}) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
      {children}
    </DndContext>
  );
}

function SortableItem({
  id,
  label,
  disabled,
  data,
  canMoveBetweenSections = false,
  reserveHandleSpace = false,
  children,
}: {
  id: string;
  label: string;
  disabled: boolean;
  data: SortableData;
  canMoveBetweenSections?: boolean;
  reserveHandleSpace?: boolean;
  children: (props: {
    setNodeRef: (node: HTMLElement | null) => void;
    style: React.CSSProperties;
    className: string;
    handle: ReactNode;
  }) => ReactNode;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging, isOver } =
    useSortable({
      id,
      data,
      disabled: { draggable: disabled, droppable: false },
    });
  return children({
    setNodeRef,
    style: { transform: CSS.Transform.toString(transform), transition },
    className: isDragging
      ? "relative z-10 opacity-60"
      : isOver
        ? "outline-2 outline-offset-2 outline-primary/40"
        : "",
    handle: disabled ? (
      reserveHandleSpace ? (
        <span className="size-6 shrink-0" aria-hidden="true" />
      ) : null
    ) : (
      <button
        type="button"
        className="btn btn-ghost btn-square btn-xs shrink-0 cursor-grab touch-none text-base-content/45 hover:text-base-content/75 focus:text-base-content/75 active:cursor-grabbing"
        aria-label={
          canMoveBetweenSections
            ? `Drag to reorder or move ${label} between sections`
            : `Drag to reorder ${label}`
        }
        title={
          canMoveBetweenSections ? "Drag to reorder or move between sections" : "Drag to reorder"
        }
        {...attributes}
        {...listeners}
      >
        <GripVertical className="size-3.5" aria-hidden="true" />
      </button>
    ),
  });
}

export function SortableSection({
  id,
  label,
  data,
  disabled,
  reserveHandleSpace,
  children,
}: {
  id: string;
  label: string;
  data: Extract<SortableData, { type: "ingredient-section" | "instruction-section" }>;
  disabled: boolean;
  reserveHandleSpace: boolean;
  children: (handle: ReactNode) => ReactNode;
}) {
  return (
    <SortableItem
      id={id}
      label={label}
      data={data}
      disabled={disabled}
      reserveHandleSpace={reserveHandleSpace}
    >
      {({ setNodeRef, style, className, handle }) => (
        <div ref={setNodeRef} style={style} className={`space-y-2 ${className}`}>
          {children(handle)}
        </div>
      )}
    </SortableItem>
  );
}

export function SortableRow({
  id,
  label,
  data,
  disabled,
  className,
  children,
}: {
  id: string;
  label: string;
  data: Extract<SortableData, { type: "ingredient" | "step" }>;
  disabled: boolean;
  className: string;
  children: (handle: ReactNode) => ReactNode;
}) {
  return (
    <SortableItem
      id={id}
      label={label}
      data={data}
      disabled={disabled}
      canMoveBetweenSections
      reserveHandleSpace
    >
      {({ setNodeRef, style, className: sortableClassName, handle }) => (
        <li
          ref={setNodeRef}
          style={style}
          className={`group flex gap-1 px-2 ${className} ${sortableClassName}`}
        >
          {children(handle)}
        </li>
      )}
    </SortableItem>
  );
}

export function SectionHeader({
  kind,
  title,
  editing,
  handle,
  pending,
  onTitleChange,
  onSave,
  onCancel,
  onBeginEditing,
  onDelete,
}: {
  kind: "ingredient" | "instruction";
  title: string;
  editing: boolean;
  handle: ReactNode;
  pending: boolean;
  onTitleChange: (title: string) => void;
  onSave: () => void;
  onCancel: () => void;
  onBeginEditing: () => void;
  onDelete: () => void;
}) {
  if (!editing && !title) return null;
  const label = `${kind} section`;
  return (
    <div className="group flex items-center gap-1">
      {handle}
      {editing ? (
        <input
          className="min-w-0 flex-1 border-b-2 border-primary bg-transparent px-1 py-1 text-sm font-semibold outline-none"
          aria-label={`${kind === "ingredient" ? "Ingredient" : "Instruction"} section title`}
          placeholder="Section title"
          value={title}
          maxLength={200}
          autoFocus
          onChange={(event) => onTitleChange(event.target.value)}
          onBlur={() => {
            if (title.trim()) onSave();
            else onCancel();
          }}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              event.preventDefault();
              event.stopPropagation();
              onCancel();
            }
            if (event.key === "Enter") event.currentTarget.blur();
          }}
        />
      ) : (
        <button
          type="button"
          className="min-w-0 flex-1 rounded-field px-1 text-left text-sm font-semibold text-base-content/70 transition-colors hover:bg-base-200 hover:text-base-content"
          onClick={onBeginEditing}
        >
          {title}
        </button>
      )}
      <DeleteButton
        label={title || label}
        pending={pending}
        visible={editing}
        onDelete={onDelete}
      />
    </div>
  );
}

export function SectionBody({
  hasItems,
  addItemLabel,
  controlsDisabled,
  onAddItem,
  children,
}: {
  hasItems: boolean;
  addItemLabel: string;
  controlsDisabled: boolean;
  onAddItem: () => void;
  children: ReactNode;
}) {
  return hasItems ? (
    <div className="overflow-hidden rounded-box border border-base-300">
      {children}
      <AddRow
        label={addItemLabel}
        onClick={onAddItem}
        disabled={controlsDisabled}
        variant="inline"
      />
    </div>
  ) : (
    <AddRow label={addItemLabel} onClick={onAddItem} disabled={controlsDisabled} />
  );
}
