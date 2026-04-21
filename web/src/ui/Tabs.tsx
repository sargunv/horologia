/**
 * Tabs primitive — Radix `Tabs` with a daisyUI-like bottom-border look
 * (active tab gets `border-primary`). We don't use daisyUI's `tabs` class
 * directly because Radix Tabs' DOM doesn't match daisyUI's structure.
 */
import { Tabs as RxTabs } from "radix-ui";
import type { ComponentProps } from "react";
import { cx } from "./cx.ts";

export const TabsRoot = RxTabs.Root;

export function TabsList({ className, ...rest }: ComponentProps<typeof RxTabs.List>) {
  return <RxTabs.List className={cx("flex border-b border-base-300", className)} {...rest} />;
}

export function TabsTrigger({ className, ...rest }: ComponentProps<typeof RxTabs.Trigger>) {
  return (
    <RxTabs.Trigger
      className={cx(
        "-mb-px border-b-2 border-transparent px-4 py-2 text-sm font-medium",
        "text-base-content/70 transition-colors hover:text-base-content",
        "data-[state=active]:border-primary data-[state=active]:text-base-content",
        "disabled:pointer-events-none disabled:opacity-40",
        className,
      )}
      {...rest}
    />
  );
}

export function TabsContent({ className, ...rest }: ComponentProps<typeof RxTabs.Content>) {
  return <RxTabs.Content className={cx("focus:outline-none", className)} {...rest} />;
}
