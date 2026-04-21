/**
 * Tooltip primitive — thin wrappers over Radix `Tooltip`. Mount
 * `TooltipProvider` once at the app root (see `AppShell`). Content renders
 * as a small inverted pill (`bg-base-content text-base-100`).
 */
import { Tooltip as RxTooltip } from "radix-ui";
import type { ComponentProps } from "react";
import { cx } from "./cx.ts";

export const TooltipProvider = RxTooltip.Provider;
export const TooltipRoot = RxTooltip.Root;
export const TooltipTrigger = RxTooltip.Trigger;

export function TooltipContent({
  className,
  sideOffset = 6,
  ...rest
}: ComponentProps<typeof RxTooltip.Content>) {
  return (
    <RxTooltip.Portal>
      <RxTooltip.Content
        sideOffset={sideOffset}
        className={cx(
          "z-50 rounded-field bg-base-content px-2 py-1 text-xs font-medium text-base-100 shadow-md",
          "data-[state=delayed-open]:animate-in data-[state=closed]:animate-out",
          "data-[state=delayed-open]:fade-in-0 data-[state=closed]:fade-out-0",
          "data-[state=delayed-open]:zoom-in-95 data-[state=closed]:zoom-out-95",
          className,
        )}
        {...rest}
      />
    </RxTooltip.Portal>
  );
}
