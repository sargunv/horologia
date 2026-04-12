import { Menu } from "@skeletonlabs/skeleton-react";
import { ChevronDown } from "lucide-react";
import type { ReactNode } from "react";

/**
 * A styled pill that serves as a Menu.Trigger for field editing.
 * Must be used inside a <Menu> context.
 *
 * Shows the current value (or the field label dimmed when empty).
 * Renders a chevron indicator to signal it's clickable.
 */
export function FieldPill({
  icon,
  label,
  value,
  children,
  className,
}: {
  /** Optional icon to show before the label/value */
  icon?: ReactNode;
  /** Field name, shown when no value is set */
  label: string;
  /** Current field value as a string */
  value?: string | null;
  /** Custom content to render instead of value/label text */
  children?: ReactNode;
  /** Additional CSS classes */
  className?: string;
}) {
  const hasValue = value != null || children != null;
  return (
    <Menu.Trigger
      className={`chip preset-tonal-surface cursor-pointer gap-1 text-sm ${!hasValue ? "opacity-60" : ""} ${className ?? ""}`}
      aria-label={value ? `${label}: ${value}` : label}
    >
      {icon}
      <span className="whitespace-nowrap">{children ?? value ?? label}</span>
      <ChevronDown className="size-3 opacity-60" aria-hidden="true" />
    </Menu.Trigger>
  );
}
