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
  /** Keep the field name visible when the value alone would be ambiguous. */
  showLabelWithValue?: boolean;
  /**
   * Render as a plain `<button>` (ref/props forwarded) instead of the default
   * `DropdownMenuTrigger`. Use when composing with non-DropdownMenu trigger
   * primitives — wrap in the parent trigger with `asChild`, e.g.:
   *
   * ```tsx
   * <PopoverTrigger asChild>
   *   <FieldPill asChild icon={<Icon />} label="Due" value={text} />
   * </PopoverTrigger>
   * ```
   */
  asChild?: boolean;
}

/**
 * A styled pill trigger for field editing.
 *
 * Default (no `asChild`): renders as a `DropdownMenuTrigger` and must be used
 * inside a `<DropdownMenuRoot>` context (legacy usage preserved for backward
 * compatibility).
 *
 * With `asChild`: renders a plain `<button>` that accepts forwarded props
 * (`aria-*`, `data-state`, etc.) so it can be composed under any Radix
 * trigger's own `asChild` slot.
 */
export const FieldPill = forwardRef<HTMLButtonElement, FieldPillProps>(function FieldPill(
  { icon, label, value, showLabelWithValue, asChild, className, ...rest },
  ref,
) {
  const hasValue = value != null;
  const mergedClassName = cx(
    "inline-flex cursor-pointer items-center gap-1 rounded-full bg-base-200 px-2.5 py-1 text-sm text-base-content hover:bg-base-300",
    !hasValue && "opacity-60",
    className,
  );
  const ariaLabel = value ? `${label}: ${value}` : label;
  const displayValue = value && showLabelWithValue ? `${label} ${value}` : (value ?? label);
  const body = (
    <>
      {icon}
      <span className="whitespace-nowrap">{displayValue}</span>
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
