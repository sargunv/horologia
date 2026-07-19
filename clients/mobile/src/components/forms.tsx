import { Column, FieldGroup, Picker, Text, TextInput, type TextInputRef } from "@expo/ui";
import { useEffect, useRef, useState } from "react";

type NativeTextInputProps = Parameters<typeof TextInput>[0];

export const FormSection = FieldGroup.Section;

export function FormField({
  label,
  value,
  onChangeText,
  ...props
}: Omit<NativeTextInputProps, "value" | "defaultValue"> & {
  label: string;
  value: string;
  onChangeText: (value: string) => void;
}) {
  const input = useRef<TextInputRef>(null);
  const nativeValue = useRef(value);
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    if (value === nativeValue.current) return;
    nativeValue.current = value;
    setRevision((current) => current + 1);
  }, [value]);

  return (
    <TextInput
      {...props}
      defaultValue={value}
      key={revision}
      onChangeText={(nextValue) => {
        nativeValue.current = nextValue;
        onChangeText(nextValue);
      }}
      placeholder={props.placeholder ?? label}
      ref={input}
      {...(!props.multiline
        ? {
            onSubmitEditing: (nextValue: string) => {
              input.current?.blur();
              props.onSubmitEditing?.(nextValue);
            },
            returnKeyType: props.returnKeyType ?? "done",
          }
        : {})}
    />
  );
}

export function FormPicker<T extends string>({
  label,
  options,
  value,
  onChange,
}: {
  label: string;
  options: readonly { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
}) {
  return (
    <Column spacing={4}>
      <Text>{label}</Text>
      <Picker onValueChange={onChange} selectedValue={value}>
        {options.map((option) => (
          <Picker.Item key={option.value} label={option.label} value={option.value} />
        ))}
      </Picker>
    </Column>
  );
}
