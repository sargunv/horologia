/**
 * DropdownMenu primitive — thin wrappers over Radix `DropdownMenu` using the
 * shared overlay surface (`SURFACE` + `SURFACE_MOTION`) and tight menu-item
 * density. Compose with a searchable variant via
 * `components/SearchableMenuContent.tsx` + `lib/useMenuSearch.ts`.
 */
import { Check } from "lucide-react";
import { DropdownMenu as RxMenu } from "radix-ui";
import type { ComponentProps } from "react";
import { cx } from "./cx.ts";
import { MENU_ITEM, SURFACE, SURFACE_MOTION } from "./surface.ts";

export const DropdownMenuRoot = RxMenu.Root;
export const DropdownMenuTrigger = RxMenu.Trigger;
export const DropdownMenuSub = RxMenu.Sub;
export const DropdownMenuRadioGroup = RxMenu.RadioGroup;

export function DropdownMenuRadioItem({
  className,
  children,
  ...rest
}: ComponentProps<typeof RxMenu.RadioItem>) {
  return (
    <RxMenu.RadioItem className={cx(MENU_ITEM, "pl-7 relative", className)} {...rest}>
      <span className="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
        <RxMenu.ItemIndicator>
          <Check className="size-3.5" aria-hidden="true" />
        </RxMenu.ItemIndicator>
      </span>
      {children}
    </RxMenu.RadioItem>
  );
}

export function DropdownMenuContent({
  className,
  sideOffset = 6,
  collisionPadding = 8,
  ...rest
}: ComponentProps<typeof RxMenu.Content>) {
  return (
    <RxMenu.Portal>
      <RxMenu.Content
        sideOffset={sideOffset}
        collisionPadding={collisionPadding}
        className={cx("z-50 min-w-[10rem] p-1", SURFACE, SURFACE_MOTION, className)}
        {...rest}
      />
    </RxMenu.Portal>
  );
}

export function DropdownMenuItem({ className, ...rest }: ComponentProps<typeof RxMenu.Item>) {
  return <RxMenu.Item className={cx(MENU_ITEM, className)} {...rest} />;
}

export function DropdownMenuSeparator({
  className,
  ...rest
}: ComponentProps<typeof RxMenu.Separator>) {
  return <RxMenu.Separator className={cx("-mx-1 my-1 h-px bg-base-300", className)} {...rest} />;
}

export function DropdownMenuSubTrigger({
  className,
  ...rest
}: ComponentProps<typeof RxMenu.SubTrigger>) {
  return (
    <RxMenu.SubTrigger
      className={cx(MENU_ITEM, "data-[state=open]:bg-base-200", className)}
      {...rest}
    />
  );
}

export function DropdownMenuSubContent({
  className,
  sideOffset = 4,
  ...rest
}: ComponentProps<typeof RxMenu.SubContent>) {
  return (
    <RxMenu.Portal>
      <RxMenu.SubContent
        sideOffset={sideOffset}
        className={cx("z-50 min-w-[10rem] p-1", SURFACE, SURFACE_MOTION, className)}
        {...rest}
      />
    </RxMenu.Portal>
  );
}
