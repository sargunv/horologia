import { Menu } from "@skeletonlabs/skeleton-react";
import type { ReactNode } from "react";
import type { MenuSearchInputProps } from "../lib/useMenuSearch.ts";

/**
 * A Menu.Content wrapper that includes a search input at the top,
 * followed by a separator and a scrollable area for menu items.
 *
 * Usage:
 * ```tsx
 * const search = useMenuSearch();
 *
 * <Menu {...search.menuProps}>
 *   <FieldPill label="Status" value={selected} />
 *   <Portal>
 *     <Menu.Positioner>
 *       <SearchableMenuContent inputProps={search.inputProps} placeholder="Search statuses...">
 *         {items.map(item => <Menu.Item ...>...</Menu.Item>)}
 *       </SearchableMenuContent>
 *     </Menu.Positioner>
 *   </Portal>
 * </Menu>
 * ```
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
        ref={inputProps.ref}
        type="text"
        value={inputProps.value}
        onChange={inputProps.onChange}
        onKeyDown={inputProps.onKeyDown}
        placeholder={placeholder}
        className="w-full bg-transparent text-sm outline-none placeholder:text-surface-500"
      />
      <Menu.Separator />
      {children}
    </Menu.Content>
  );
}
