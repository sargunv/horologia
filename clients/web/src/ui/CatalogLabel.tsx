/**
 * CatalogLabel — specimen / ledger field label.
 * Letterspaced caps that read as herbarium annotation hand.
 * Under Florilegium this is the primary “inscription” signal.
 */
import type { ComponentProps } from "react";
import { cx } from "./cx.ts";

export function CatalogLabel({ className, ...rest }: ComponentProps<"span">) {
  return (
    <span
      className={cx(
        "catalog-label text-3xs font-semibold uppercase tracking-caps text-base-content/65",
        className,
      )}
      {...rest}
    />
  );
}
