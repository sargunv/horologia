/**
 * AlertDialog primitive — thin wrappers over Radix `AlertDialog` for
 * destructive confirmations (delete, revoke, remove, etc.). Uses the same
 * overlay surface + motion as `Dialog`.
 *
 * Key differences from `Dialog`:
 * - Role `alertdialog` (applied automatically by Radix).
 * - No close-X button; `AlertDialogCancel` is the close affordance, and
 *   `AlertDialogAction` is the confirm affordance.
 * - Focus lands on `AlertDialogCancel` by default (Radix behavior), which
 *   is the safe choice for destructive flows.
 */
import { AlertDialog as RxAlertDialog } from "radix-ui";
import type { ComponentProps, ReactNode } from "react";
import { cx } from "./cx.ts";
import { SURFACE, SURFACE_MOTION } from "./surface.ts";

export const AlertDialogRoot = RxAlertDialog.Root;
export const AlertDialogTrigger = RxAlertDialog.Trigger;
export const AlertDialogCancel = RxAlertDialog.Cancel;
export const AlertDialogAction = RxAlertDialog.Action;

export function AlertDialogContent({
  className,
  children,
  ...rest
}: ComponentProps<typeof RxAlertDialog.Content>) {
  return (
    <RxAlertDialog.Portal>
      <RxAlertDialog.Overlay
        className={cx(
          "fixed inset-0 z-40 bg-black/40 backdrop-blur-sm",
          "data-[state=open]:animate-in data-[state=closed]:animate-out",
          "data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0",
        )}
      />
      <RxAlertDialog.Content
        className={cx(
          "fixed left-1/2 top-1/2 z-50 w-[calc(100%-2rem)] max-w-lg",
          "-translate-x-1/2 -translate-y-1/2 p-6",
          SURFACE,
          SURFACE_MOTION,
          className,
        )}
        {...rest}
      >
        {children}
      </RxAlertDialog.Content>
    </RxAlertDialog.Portal>
  );
}

export function AlertDialogTitle({
  className,
  ...rest
}: ComponentProps<typeof RxAlertDialog.Title>) {
  return (
    <RxAlertDialog.Title
      className={cx("text-lg font-semibold leading-tight", className)}
      {...rest}
    />
  );
}

export function AlertDialogDescription({
  className,
  ...rest
}: ComponentProps<typeof RxAlertDialog.Description>) {
  return (
    <RxAlertDialog.Description
      className={cx("mt-1 text-sm text-base-content/70", className)}
      {...rest}
    />
  );
}

/**
 * Header with title. Unlike `DialogHeader`, no close-X is rendered —
 * destructive confirmations should close via explicit Cancel/Action.
 */
export function AlertDialogHeader({ title }: { title: ReactNode }) {
  return (
    <header>
      <AlertDialogTitle>{title}</AlertDialogTitle>
    </header>
  );
}

/** Right-aligned action bar. Typical: Cancel + destructive action. */
export function AlertDialogFooter({ children }: { children: ReactNode }) {
  return <footer className="flex justify-end gap-2">{children}</footer>;
}
