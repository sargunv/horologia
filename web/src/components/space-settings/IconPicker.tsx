import { Menu, Portal } from "@skeletonlabs/skeleton-react";
import { Check } from "lucide-react";
import { useMemo } from "react";
import {
  EFFORT_SUGGESTED_ICONS,
  FALLBACK_ICON,
  getIcon,
  searchIcons,
} from "../../lib/level-icons.ts";
import { useMenuSearch } from "../../lib/useMenuSearch.ts";
import { MENU_ITEM_CLASS, SearchableMenuContent } from "../SearchableMenuContent.tsx";

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
  /** Current icon name (kebab-case). */
  value: string | undefined;
  /** Called when the user picks an icon. */
  onChange: (icon: string) => void;
  disabled?: boolean;
  label?: string;
  /** Which set of suggested icons to show when not searching. */
  suggestedIcons?: string[];
}) {
  const search = useMenuSearch();
  const CurrentIcon = value ? getIcon(value) : FALLBACK_ICON;

  const results = useMemo(
    () => searchIcons(search.query, suggestedIcons),
    [search.query, suggestedIcons],
  );

  return (
    <Menu {...search.menuProps} closeOnSelect={false}>
      <Menu.Trigger
        className="btn-icon btn-icon-sm shrink-0 preset-tonal-surface"
        disabled={disabled}
        aria-label={label ?? "Pick icon"}
      >
        <CurrentIcon className="size-4" aria-hidden="true" />
      </Menu.Trigger>
      <Portal>
        <Menu.Positioner>
          <SearchableMenuContent inputProps={search.inputProps} placeholder="Search icons...">
            {results.length === 0 ? (
              <div className="text-surface-500 px-3 py-2 text-sm">No matching icons</div>
            ) : (
              results.map((name) => {
                const Icon = getIcon(name);
                const isSelected = name === value;
                return (
                  <Menu.OptionItem
                    key={name}
                    type="radio"
                    checked={isSelected}
                    value={name}
                    onCheckedChange={(checked) => {
                      if (checked) onChange(name);
                    }}
                    className={MENU_ITEM_CLASS}
                  >
                    <Menu.ItemIndicator className="invisible data-[state=checked]:visible">
                      <Check className="size-4" />
                    </Menu.ItemIndicator>
                    <Icon className="size-4" aria-hidden="true" />
                    <Menu.ItemText>{name}</Menu.ItemText>
                  </Menu.OptionItem>
                );
              })
            )}
          </SearchableMenuContent>
        </Menu.Positioner>
      </Portal>
    </Menu>
  );
}
