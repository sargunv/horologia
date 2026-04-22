/**
 * Card — a plain surface container. `rounded-box bg-base-100 border border-base-300`
 * is the shared card look used on auth pages, list empty states, the 404 page,
 * and space tiles. For interactive cards (dialogs, popovers, menus), use the
 * primitives from `ui/Dialog.tsx` / `ui/DropdownMenu.tsx` instead.
 */
import type { ComponentProps } from "react";
import { cx } from "./cx.ts";

export function Card({ className, ...rest }: ComponentProps<"div">) {
  return (
    <div className={cx("rounded-box border border-base-300 bg-base-100", className)} {...rest} />
  );
}
