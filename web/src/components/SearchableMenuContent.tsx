import { Menu } from "@skeletonlabs/skeleton-react";
import type { ReactNode } from "react";
import type { MenuSearchInputProps } from "../lib/useMenuSearch.ts";

/**
 * Default className for Menu.Item / Menu.OptionItem inside searchable menus.
 * Overrides Skeleton's justify-content: space-between with justify-start.
 */
export const MENU_ITEM_CLASS = "justify-start gap-2 text-sm";

/**
 * A Menu.Content wrapper that includes a search input at the top,
 * followed by a separator and a scrollable area for menu items.
 */
export function SearchableMenuContent({
  inputProps,
  placeholder = "Search...",
  children,
  className,
}: {
  /** Input props from useMenuSearch().inputProps */
  inputProps: MenuSearchInputProps;
  /** Placeholder text for the search input */
  placeholder?: string;
  /** Menu items to render in the scrollable area */
  children: ReactNode;
  /** Additional CSS classes for Menu.Content */
  className?: string | undefined;
}) {
  return (
    <Menu.Content className={className}>
      <input
        {...inputProps}
        type="text"
        placeholder={placeholder}
        aria-label={placeholder}
        className="w-full bg-transparent text-sm outline-none placeholder:text-surface-500"
      />
      <Menu.Separator />
      {children}
    </Menu.Content>
  );
}
