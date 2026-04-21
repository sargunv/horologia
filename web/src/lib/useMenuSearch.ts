import {
  useCallback,
  useRef,
  useState,
  type ChangeEvent,
  type KeyboardEvent,
  type RefObject,
} from "react";

export interface MenuSearchInputProps {
  ref: RefObject<HTMLInputElement | null>;
  value: string;
  onChange: (e: ChangeEvent<HTMLInputElement>) => void;
  onKeyDown: (e: KeyboardEvent<HTMLInputElement>) => void;
}

export interface MenuSearchResult {
  query: string;
  setQuery: (query: string) => void;
  inputRef: RefObject<HTMLInputElement | null>;
  inputProps: MenuSearchInputProps;
  /** Props to spread on a Radix DropdownMenuRoot or DropdownMenuSub. */
  menuProps: {
    onOpenChange: (open: boolean) => void;
  };
}

const TYPEAHEAD_KEY_RE = /^[\S\s]$/u; // any single printable character

/**
 * Hook for coordinating a search input inside a Radix DropdownMenu.
 *
 * Manages:
 * - Search query state
 * - Query reset when the enclosing menu closes (via `menuProps.onOpenChange`)
 * - ArrowDown / Enter from the input → first menu item
 * - Swallows printable single-character keys so Radix's typeahead doesn't
 *   intercept text input; leaves navigation keys (Arrow*, Home, End, Tab,
 *   Escape) alone so Radix's menu keyboard model still works.
 *
 * Focus is not managed explicitly — place the search input as the first
 * tabbable inside the `DropdownMenu.Content` and Radix's FocusScope lands
 * on it automatically when the menu opens.
 *
 * Does NOT manage filtering — consumers filter based on `query` themselves.
 */
export function useMenuSearch(): MenuSearchResult {
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const handleOpenChange = useCallback((open: boolean) => {
    if (!open) setQuery("");
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
    // Block Radix typeahead for printable characters so they reach the input
    // instead of jumping focus to a menuitem whose label starts with that key.
    if (TYPEAHEAD_KEY_RE.test(e.key)) {
      e.stopPropagation();
    }

    if (e.key !== "Enter" && e.key !== "ArrowDown") return;

    const menu = inputRef.current?.closest("[role='menu']");
    if (!menu) return;

    const firstItem = menu.querySelector(
      [
        "[role='menuitem']:not([aria-disabled='true'])",
        "[role='menuitemradio']:not([aria-disabled='true'])",
        "[role='menuitemcheckbox']:not([aria-disabled='true'])",
      ].join(", "),
    );

    if (e.key === "Enter") {
      e.preventDefault();
      if (firstItem instanceof HTMLElement) firstItem.click();
      return;
    }

    // ArrowDown
    e.preventDefault();
    if (firstItem instanceof HTMLElement) firstItem.focus();
  }, []);

  const handleChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setQuery(e.target.value);
  }, []);

  return {
    query,
    setQuery,
    inputRef,
    inputProps: {
      ref: inputRef,
      value: query,
      onChange: handleChange,
      onKeyDown: handleKeyDown,
    },
    menuProps: { onOpenChange: handleOpenChange },
  };
}
