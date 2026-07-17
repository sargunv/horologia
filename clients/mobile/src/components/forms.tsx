import type { PropsWithChildren } from "react";
import { Pressable, StyleSheet, Text, TextInput, type TextInputProps, View } from "react-native";

import { colors } from "@/design/tokens";

export function FormSection({ title, children }: PropsWithChildren<{ title: string }>) {
  return (
    <View style={styles.section}>
      <Text accessibilityRole="header" style={styles.sectionTitle}>
        {title}
      </Text>
      {children}
    </View>
  );
}

export function FormField({ label, ...props }: TextInputProps & { label: string }) {
  return (
    <View style={styles.field}>
      <Text style={styles.label}>{label}</Text>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={colors.muted}
        style={[styles.input, props.multiline && styles.multiline]}
        {...props}
      />
    </View>
  );
}

export function ChoiceChips<T extends string>({
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
    <View style={styles.field}>
      <Text style={styles.label}>{label}</Text>
      <View accessibilityLabel={label} accessibilityRole="radiogroup" style={styles.chips}>
        {options.map((option) => {
          const selected = option.value === value;
          return (
            <Pressable
              accessibilityRole="radio"
              accessibilityState={{ selected }}
              key={option.value}
              onPress={() => onChange(option.value)}
              style={[styles.chip, selected && styles.chipSelected]}
            >
              <Text style={[styles.chipText, selected && styles.chipTextSelected]}>
                {option.label}
              </Text>
            </Pressable>
          );
        })}
      </View>
    </View>
  );
}

export function ActionButton({
  label,
  onPress,
  disabled = false,
  destructive = false,
}: {
  label: string;
  onPress: () => void;
  disabled?: boolean;
  destructive?: boolean;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      style={[
        styles.button,
        destructive && styles.destructiveButton,
        disabled && styles.buttonDisabled,
      ]}
    >
      <Text style={styles.buttonText}>{label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  section: {
    backgroundColor: colors.surface,
    borderColor: colors.outline,
    borderRadius: 18,
    borderWidth: StyleSheet.hairlineWidth,
    gap: 14,
    padding: 17,
  },
  sectionTitle: { color: colors.ink, fontSize: 19, fontWeight: "700" },
  field: { gap: 7 },
  label: { color: colors.muted, fontSize: 13, fontWeight: "700" },
  input: {
    backgroundColor: colors.canvas,
    borderColor: colors.outline,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    color: colors.ink,
    fontSize: 16,
    minHeight: 48,
    paddingHorizontal: 13,
    paddingVertical: 11,
  },
  multiline: { minHeight: 110, textAlignVertical: "top" },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 7 },
  chip: {
    backgroundColor: colors.canvas,
    borderColor: colors.outline,
    borderRadius: 999,
    borderWidth: StyleSheet.hairlineWidth,
    minHeight: 38,
    paddingHorizontal: 13,
    paddingVertical: 9,
  },
  chipSelected: { backgroundColor: colors.accent, borderColor: colors.accent },
  chipText: { color: colors.ink, fontSize: 14, fontWeight: "600" },
  chipTextSelected: { color: colors.surface },
  button: {
    alignItems: "center",
    backgroundColor: colors.accent,
    borderRadius: 14,
    justifyContent: "center",
    minHeight: 50,
    paddingHorizontal: 18,
  },
  destructiveButton: { backgroundColor: colors.danger },
  buttonDisabled: { opacity: 0.45 },
  buttonText: { color: colors.surface, fontSize: 16, fontWeight: "700" },
});
