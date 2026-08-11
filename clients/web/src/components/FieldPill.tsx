/**
 * FieldPill — annotation-slip trigger for inline field editing.
 *
 * Shows a catalog key + value (specimen-label culture), not a modern
 * value-only pill. Compose inside DropdownMenuRoot / Popover asChild.
 */
import { ChevronDown } from "lucide-react";
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import { cx } from "../ui/cx.ts";
import { DropdownMenuTrigger } from "../ui/DropdownMenu.tsx";

interface FieldPillProps extends Omit<
  ButtonHTMLAttributes<HTMLButtonElement>,
  "children" | "value"
> {
  icon?: ReactNode;
  label: string;
  value?: string | null;
  /** @deprecated Label is always visible as a catalog key. */
  showLabelWithValue?: boolean;
  /**
   * Render as a plain `<button>` (ref/props forwarded) instead of the default
   * `DropdownMenuTrigger`. Use when composing with non-DropdownMenu trigger
   * primitives — wrap in the parent trigger with `asChild`.
   */
  asChild?: boolean;
}

export const FieldPill = forwardRef<HTMLButtonElement, FieldPillProps>(function FieldPill(
  { icon, label, value, showLabelWithValue: _showLabelWithValue, asChild, className, ...rest },
  ref,
) {
  const hasValue = value != null && value !== "";
  const mergedClassName = cx(
    "field-pill inline-flex cursor-pointer items-center gap-1.5 rounded-field border border-base-300 bg-base-100 px-2 py-1 text-sm text-base-content hover:bg-base-200",
    !hasValue && "opacity-70",
    className,
  );
  const ariaLabel = hasValue ? `${label}: ${value}` : label;
  const body = (
    <>
      {icon}
      <span className="inline-flex min-w-0 items-baseline gap-1.5">
        <span className="catalog-label text-3xs font-semibold uppercase tracking-caps text-base-content/60">
          {label}
        </span>
        <span className="field-pill-value whitespace-nowrap">{hasValue ? value : "—"}</span>
      </span>
      <ChevronDown className="size-3 opacity-60" aria-hidden="true" />
    </>
  );

  if (asChild) {
    return (
      <button ref={ref} type="button" className={mergedClassName} aria-label={ariaLabel} {...rest}>
        {body}
      </button>
    );
  }

  return (
    <DropdownMenuTrigger ref={ref} className={mergedClassName} aria-label={ariaLabel} {...rest}>
      {body}
    </DropdownMenuTrigger>
  );
});
