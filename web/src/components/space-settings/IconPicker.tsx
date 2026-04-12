import { Popover, Portal, ToggleGroup } from "@skeletonlabs/skeleton-react";
import { CircleOff } from "lucide-react";
import { useState } from "react";
import { FALLBACK_ICON, LEVEL_ICONS, LEVEL_ICON_NAMES } from "../../lib/level-icons.ts";

/**
 * A popover-based icon picker for selecting a Lucide icon.
 * Shows the currently selected icon as a button; clicking opens a grid of choices.
 * Uses Skeleton's ToggleGroup for roving focus and arrow-key navigation.
 */
export function IconPicker({
  value,
  onChange,
  disabled,
  label,
}: {
  /** Current icon name (kebab-case) or null for no icon. */
  value: string | null | undefined;
  /** Called when the user picks an icon or clears it. */
  onChange: (icon: string | null) => void;
  disabled?: boolean;
  label?: string;
}) {
  const [open, setOpen] = useState(false);
  const CurrentIcon = value ? (LEVEL_ICONS[value] ?? FALLBACK_ICON) : null;

  return (
    <Popover open={open} onOpenChange={(e) => setOpen(e.open)}>
      <Popover.Trigger
        className={`btn-icon btn-icon-sm shrink-0 ${value ? "preset-tonal-surface" : "preset-outlined-surface-200-800 opacity-60"}`}
        disabled={disabled}
        aria-label={label ?? "Pick icon"}
      >
        {CurrentIcon ? (
          <CurrentIcon className="size-4" aria-hidden="true" />
        ) : (
          <CircleOff className="size-3.5" aria-hidden="true" />
        )}
      </Popover.Trigger>
      <Portal>
        <Popover.Positioner>
          <Popover.Content className="bg-surface-100-900 z-50 rounded-lg border border-surface-200-800 p-2 shadow-lg">
            <div className="mb-1 flex items-center justify-between px-1">
              <Popover.Title className="text-surface-600-400 text-xs font-medium">
                Choose icon
              </Popover.Title>
              {value && (
                <button
                  type="button"
                  className="text-error-500 text-xs hover:underline"
                  aria-label="Clear icon selection"
                  onClick={() => {
                    onChange(null);
                    setOpen(false);
                  }}
                >
                  Clear
                </button>
              )}
            </div>
            <ToggleGroup
              className="grid grid-cols-8 gap-0.5"
              multiple={false}
              value={value ? [value] : []}
              onValueChange={(details) => {
                const selected = details.value[0] ?? null;
                onChange(selected);
                if (selected) setOpen(false);
              }}
            >
              {LEVEL_ICON_NAMES.map((name) => {
                const Icon = LEVEL_ICONS[name]!;
                const isSelected = name === value;
                return (
                  <ToggleGroup.Item
                    key={name}
                    value={name}
                    className={`btn-icon btn-icon-sm rounded ${isSelected ? "preset-filled-primary-500" : "hover:bg-surface-200-800"}`}
                    aria-label={name}
                    title={name}
                  >
                    <Icon className="size-4" aria-hidden="true" />
                  </ToggleGroup.Item>
                );
              })}
            </ToggleGroup>
          </Popover.Content>
        </Popover.Positioner>
      </Portal>
    </Popover>
  );
}
