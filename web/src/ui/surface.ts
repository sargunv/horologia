/**
 * Shared class constants for overlay primitives (Dialog, DropdownMenu,
 * Popover when we add one).
 *
 * `SURFACE_MOTION` and `MENU_ITEM` both rely on Radix's `data-[state]` /
 * `data-[highlighted]` / `data-[side]` attributes, which Radix applies to
 * `Content` / `Item` elements. They won't work on non-Radix overlays
 * (cmdk items use `data-[selected=true]` instead — see `Combobox` patterns).
 */

/** Overlay card background + border + radius + shadow. */
export const SURFACE = "bg-base-100 text-base-content border border-base-300 rounded-box shadow-lg";

/** Enter/exit motion for Radix overlays with `data-state` + `data-side`. */
export const SURFACE_MOTION =
  "data-[state=open]:animate-in data-[state=closed]:animate-out " +
  "data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0 " +
  "data-[state=open]:zoom-in-95 data-[state=closed]:zoom-out-95 " +
  "data-[side=top]:slide-in-from-bottom-1 " +
  "data-[side=bottom]:slide-in-from-top-1 " +
  "data-[side=left]:slide-in-from-right-1 " +
  "data-[side=right]:slide-in-from-left-1";

/** Compact menu row used by Radix DropdownMenu items and sub-triggers. */
export const MENU_ITEM =
  "flex cursor-default select-none items-center gap-2 rounded-field px-2 py-1.5 text-sm " +
  "outline-none " +
  "data-[highlighted]:bg-base-200 " +
  "data-[disabled]:pointer-events-none data-[disabled]:opacity-40";
