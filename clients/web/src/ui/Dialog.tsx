/**
 * Dialog primitive — thin wrappers over Radix `Dialog` with the shared
 * overlay surface + motion. `DialogHeader` + `DialogFooter` consolidate
 * the repeated title/close and footer-actions boilerplate; pass
 * `title` to `DialogHeader` and action buttons as children to
 * `DialogFooter`.
 */
import { X as XIcon } from "lucide-react";
import { Dialog as RxDialog } from "radix-ui";
import type { ComponentProps, ReactNode } from "react";
import { cx } from "./cx.ts";
import { SURFACE, SURFACE_MOTION } from "./surface.ts";

export const DialogRoot = RxDialog.Root;
export const DialogTrigger = RxDialog.Trigger;
export const DialogClose = RxDialog.Close;

export function DialogContent({
  className,
  children,
  ...rest
}: ComponentProps<typeof RxDialog.Content>) {
  return (
    <RxDialog.Portal>
      <RxDialog.Overlay
        className={cx(
          "fixed inset-0 z-40 bg-black/40 backdrop-blur-sm",
          "data-[state=open]:animate-in data-[state=closed]:animate-out",
          "data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0",
        )}
      />
      <RxDialog.Content
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
      </RxDialog.Content>
    </RxDialog.Portal>
  );
}

export function DialogTitle({ className, ...rest }: ComponentProps<typeof RxDialog.Title>) {
  return (
    <RxDialog.Title className={cx("text-lg font-semibold leading-tight", className)} {...rest} />
  );
}

export function DialogDescription({
  className,
  ...rest
}: ComponentProps<typeof RxDialog.Description>) {
  return (
    <RxDialog.Description
      className={cx("mt-1 text-sm text-base-content/70", className)}
      {...rest}
    />
  );
}

/**
 * Header with title + close button. Pass `title` (string or node) and
 * optionally hide the close button with `closeButton={false}` (for the
 * token-reveal flow, where the dialog should only close via an explicit
 * "Done" action).
 */
export function DialogHeader({
  title,
  closeButton = true,
  closeDisabled,
}: {
  title: ReactNode;
  closeButton?: boolean;
  closeDisabled?: boolean;
}) {
  return (
    <header className="flex items-start justify-between gap-2">
      <DialogTitle>{title}</DialogTitle>
      {closeButton && (
        <DialogClose
          className="btn btn-ghost btn-square btn-sm"
          aria-label="Close dialog"
          disabled={closeDisabled}
        >
          <XIcon className="size-4" aria-hidden="true" />
        </DialogClose>
      )}
    </header>
  );
}

/** Right-aligned action bar. Typical: Cancel + primary action. */
export function DialogFooter({ children }: { children: ReactNode }) {
  return <footer className="flex justify-end gap-2">{children}</footer>;
}
