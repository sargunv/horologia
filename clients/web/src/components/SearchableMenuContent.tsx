import type { ComponentProps, ReactNode } from "react";
import type { MenuSearchResult } from "../lib/useMenuSearch.ts";
import {
  DropdownMenuContent,
  DropdownMenuSeparator,
  DropdownMenuSubContent,
} from "../ui/DropdownMenu.tsx";

type TopContentProps = {
  search: MenuSearchResult;
  placeholder?: string;
  inputLabel?: string;
  children: ReactNode;
} & Omit<ComponentProps<typeof DropdownMenuContent>, "children">;

/**
 * Wraps `DropdownMenuContent` with a search input + separator. Pass the
 * `useMenuSearch()` result; the input is the first tabbable so Radix's
 * FocusScope lands on it automatically when the menu opens.
 *
 * The caller must spread `search.menuProps` onto the enclosing
 * `DropdownMenuRoot` so the query resets on close.
 */
export function SearchableMenuContent({
  search,
  placeholder = "Search...",
  inputLabel,
  children,
  ...rest
}: TopContentProps) {
  return (
    <DropdownMenuContent {...rest}>
      <SearchInput search={search} placeholder={placeholder} inputLabel={inputLabel} />
      <DropdownMenuSeparator />
      {children}
    </DropdownMenuContent>
  );
}

type SubContentProps = {
  search: MenuSearchResult;
  placeholder?: string;
  inputLabel?: string;
  children: ReactNode;
} & Omit<ComponentProps<typeof DropdownMenuSubContent>, "children">;

/**
 * Sub-menu variant. Also spread `search.menuProps` onto the enclosing
 * `DropdownMenuSub` so the query resets when the sub-menu closes.
 */
export function SearchableSubMenuContent({
  search,
  placeholder = "Search...",
  inputLabel,
  children,
  ...rest
}: SubContentProps) {
  return (
    <DropdownMenuSubContent {...rest}>
      <SearchInput search={search} placeholder={placeholder} inputLabel={inputLabel} />
      <DropdownMenuSeparator />
      {children}
    </DropdownMenuSubContent>
  );
}

function SearchInput({
  search,
  placeholder,
  inputLabel,
}: {
  search: MenuSearchResult;
  placeholder: string;
  inputLabel: string | undefined;
}) {
  return (
    <div role="none" className="px-2 pt-1.5 pb-1">
      <input
        {...search.inputProps}
        type="text"
        placeholder={placeholder}
        aria-label={inputLabel ?? placeholder}
        className="w-full bg-transparent text-sm outline-none placeholder:text-base-content/50"
      />
    </div>
  );
}
