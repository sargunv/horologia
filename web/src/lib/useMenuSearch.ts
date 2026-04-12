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
  /** Current search query */
  query: string;
  /** Update the search query */
  setQuery: (query: string) => void;
  /** Ref to the search input element */
  inputRef: RefObject<HTMLInputElement | null>;
  /** Props to spread on the search <input> element */
  inputProps: MenuSearchInputProps;
  /**
   * Props to spread on the <Menu> root.
   * Disables typeahead (so keystrokes go to the input) and
   * handles focus/reset on open/close.
   */
  menuProps: {
    typeahead: false;
    onOpenChange: (details: { open: boolean }) => void;
  };
  /**
   * The onOpenChange handler, exposed separately for cases where
   * the consumer needs to compose it with their own onOpenChange logic.
   */
  handleOpenChange: (details: { open: boolean }) => void;
}

/**
 * Hook for coordinating a search input inside a Skeleton Menu.
 *
 * Manages:
 * - Search query state
 * - Auto-focus on the input when the menu opens
 * - Query reset when the menu closes
 * - ArrowDown from the input to the first menu item
 *
 * Does NOT manage filtering — consumers filter/transform items
 * based on `query` themselves. This keeps the hook flexible for
 * standard text filtering, custom parsing (dates), and async search.
 */
export function useMenuSearch(): MenuSearchResult {
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const handleOpenChange = useCallback((details: { open: boolean }) => {
    if (details.open) {
      requestAnimationFrame(() => {
        inputRef.current?.focus();
      });
    } else {
      setQuery("");
    }
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
    // Prevent Space from being consumed by Menu's keyboard handler
    if (e.key === " ") {
      e.stopPropagation();
      return;
    }

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
      if (firstItem instanceof HTMLElement) {
        firstItem.click();
      }
      return;
    }

    if (e.key === "ArrowDown") {
      e.preventDefault();
      if (firstItem instanceof HTMLElement) {
        firstItem.focus();
      }
    }
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
    menuProps: {
      typeahead: false,
      onOpenChange: handleOpenChange,
    },
    handleOpenChange,
  };
}
