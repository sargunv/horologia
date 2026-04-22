import { useMemo } from "react";
import {
  EFFORT_SUGGESTED_ICONS,
  FALLBACK_ICON,
  getIcon,
  searchIcons,
} from "../../lib/level-icons.ts";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";
import {
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuRoot,
  DropdownMenuTrigger,
} from "../../ui/DropdownMenu.tsx";
import { SearchableMenuContent } from "../SearchableMenuContent.tsx";

/**
 * A searchable menu-based icon picker for selecting a Lucide icon.
 * Shows suggested icons by default; typing searches across all ~1700 Lucide icons.
 */
export function IconPicker({
  value,
  onChange,
  disabled,
  label,
  suggestedIcons = EFFORT_SUGGESTED_ICONS,
}: {
  value: string | undefined;
  onChange: (icon: string) => void;
  disabled?: boolean;
  label?: string;
  suggestedIcons?: string[];
}) {
  const search = useMenuSearch();
  const CurrentIcon = value ? getIcon(value) : FALLBACK_ICON;

  const results = useMemo(
    () => searchIcons(search.query, suggestedIcons),
    [search.query, suggestedIcons],
  );

  return (
    <DropdownMenuRoot {...search.menuProps}>
      <DropdownMenuTrigger
        className="btn btn-soft btn-square btn-sm shrink-0"
        disabled={disabled}
        aria-label={label ?? "Pick icon"}
      >
        <CurrentIcon className="size-4" aria-hidden="true" />
      </DropdownMenuTrigger>
      <SearchableMenuContent
        search={search}
        placeholder="Search icons..."
        inputLabel="Search icons"
      >
        {results.length === 0 ? (
          <div className="px-3 py-2 text-sm text-base-content/60">No matching icons</div>
        ) : (
          <DropdownMenuRadioGroup
            value={value ?? ""}
            onValueChange={(v) => {
              if (v) onChange(v);
            }}
          >
            {results.map((name) => {
              const Icon = getIcon(name);
              return (
                <DropdownMenuRadioItem key={name} value={name}>
                  <Icon className="size-4" aria-hidden="true" />
                  <span>{name}</span>
                </DropdownMenuRadioItem>
              );
            })}
          </DropdownMenuRadioGroup>
        )}
      </SearchableMenuContent>
    </DropdownMenuRoot>
  );
}
