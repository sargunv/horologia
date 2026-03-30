import { Combobox, Portal, useListCollection } from "@skeletonlabs/skeleton-react";
import { Check, ChevronDown } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

const ALL_TIMEZONES: string[] = Intl.supportedValuesOf("timeZone");

export function TimezoneCombobox({
  value,
  onChange,
  onOpenChange,
  disabled,
  "aria-label": ariaLabel = "Timezone",
}: {
  value: string;
  onChange: (value: string) => void;
  onOpenChange?: ((open: boolean) => void) | undefined;
  disabled?: boolean | undefined;
  "aria-label"?: string | undefined;
}) {
  const [inputValue, setInputValue] = useState(value);

  useEffect(() => {
    setInputValue(value);
  }, [value]);

  const filteredTimezones = useMemo(
    () =>
      inputValue
        ? ALL_TIMEZONES.filter((tz) => tz.toLowerCase().includes(inputValue.toLowerCase()))
        : ALL_TIMEZONES,
    [inputValue],
  );

  const collection = useListCollection({
    items: filteredTimezones,
    itemToString: (tz) => tz,
    itemToValue: (tz) => tz,
  });

  return (
    <Combobox
      collection={collection}
      value={[value]}
      inputValue={inputValue}
      onInputValueChange={(e) => setInputValue(e.inputValue)}
      onValueChange={(e) => {
        if (e.value[0]) onChange(e.value[0]);
      }}
      onOpenChange={(e) => onOpenChange?.(e.open)}
      loopFocus
      openOnClick
      disabled={disabled}
    >
      <Combobox.Control className="input-group preset-outlined-surface-200-800 grid grid-cols-[1fr_auto]">
        <Combobox.Input
          placeholder="Search timezones..."
          className="ig-input"
          aria-label={ariaLabel}
        />
        <Combobox.Trigger className="ig-btn preset-tonal-surface">
          <ChevronDown className="size-4" aria-hidden="true" />
        </Combobox.Trigger>
      </Combobox.Control>
      <Portal>
        <Combobox.Positioner className="z-50">
          <Combobox.Content className="card preset-outlined-surface-200-800 bg-surface-100-900 max-h-60 overflow-auto p-1">
            {filteredTimezones.length === 0 ? (
              <div className="text-surface-600-400 px-3 py-2 text-sm">No timezones found</div>
            ) : (
              filteredTimezones.map((tz) => (
                <Combobox.Item
                  key={tz}
                  item={tz}
                  className="flex cursor-pointer items-center gap-2 rounded px-3 py-2 text-sm data-[highlighted]:bg-surface-200-800"
                >
                  <Combobox.ItemIndicator>
                    <Check className="size-4" />
                  </Combobox.ItemIndicator>
                  <Combobox.ItemText>{tz}</Combobox.ItemText>
                </Combobox.Item>
              ))
            )}
          </Combobox.Content>
        </Combobox.Positioner>
      </Portal>
    </Combobox>
  );
}
