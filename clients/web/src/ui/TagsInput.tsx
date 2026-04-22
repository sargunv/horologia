/**
 * TagsInput primitive — wraps Ark UI's headless TagsInput with daisyUI
 * styling (control rendered as `input`, each tag as `badge badge-soft`).
 * Ark is pulled in only for this component; everywhere else we use Radix.
 * Callers get a simple `value: string[]` / `onValueChange(value)` contract.
 */
import {
  TagsInputContext,
  TagsInputControl,
  TagsInputHiddenInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDeleteTrigger,
  TagsInputItemInput,
  TagsInputItemPreview,
  TagsInputItemText,
  TagsInputRoot,
} from "@ark-ui/react";
import { X } from "lucide-react";
import { cx } from "./cx.ts";

export interface TagsInputProps {
  value: string[];
  onValueChange: (value: string[]) => void;
  placeholder?: string;
  disabled?: boolean;
  readOnly?: boolean;
  max?: number;
  editable?: boolean;
  className?: string;
  name?: string;
  /**
   * Accessible label for the underlying input. Forwarded to `aria-label` on
   * the `TagsInputInput` so screen readers announce the widget's purpose.
   */
  label?: string;
}

export function TagsInput({
  value,
  onValueChange,
  placeholder,
  disabled,
  readOnly,
  max,
  editable = true,
  className,
  name,
  label,
}: TagsInputProps) {
  return (
    <TagsInputRoot
      value={value}
      onValueChange={(d) => {
        onValueChange(d.value);
      }}
      disabled={disabled}
      readOnly={readOnly}
      max={max}
      editable={editable}
      name={name}
      className={cx("w-full", className)}
    >
      <TagsInputContext>
        {(api) => (
          <TagsInputControl
            className={cx(
              "input flex h-auto min-h-10 w-full flex-wrap items-center gap-1 py-1",
              "has-[input:focus]:outline-2 has-[input:focus]:outline-primary/60",
              "data-[disabled]:opacity-60",
            )}
          >
            {api.value.map((item: string, index: number) => (
              <TagsInputItem key={`${item}-${index}`} index={index} value={item}>
                <TagsInputItemPreview className="badge badge-soft gap-1 data-[highlighted]:ring-1 data-[highlighted]:ring-primary/40">
                  <TagsInputItemText>{item}</TagsInputItemText>
                  <TagsInputItemDeleteTrigger
                    aria-label={`Remove ${item}`}
                    className="-mr-1 inline-flex items-center justify-center rounded-full p-0.5 hover:bg-base-content/10"
                  >
                    <X className="h-3 w-3" />
                  </TagsInputItemDeleteTrigger>
                </TagsInputItemPreview>
                <TagsInputItemInput className="bg-transparent outline-none" />
              </TagsInputItem>
            ))}
            <TagsInputInput
              placeholder={placeholder}
              aria-label={label}
              className="min-w-[8ch] flex-1 bg-transparent px-1 outline-none placeholder:text-base-content/50"
            />
          </TagsInputControl>
        )}
      </TagsInputContext>
      <TagsInputHiddenInput />
    </TagsInputRoot>
  );
}
