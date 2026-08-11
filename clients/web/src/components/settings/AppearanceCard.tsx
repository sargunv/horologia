import { Check, Moon, Palette, Sun } from "lucide-react";
import { useCallback, useEffect, useRef, type PointerEvent } from "react";
import { toast } from "sonner";
import type { components } from "@horologia/client-core/schema";
import { useUserPatch } from "../../lib/mutations.ts";
import {
  darkThemes,
  lightThemes,
  useTheme,
  type ThemeDefinition,
  type ThemeMode,
} from "../../lib/theme.tsx";
import { cx } from "../../ui/cx.ts";
import { SettingsSection } from "../space-settings/SettingsSection.tsx";

type UserUpdate = components["schemas"]["UserUpdate"];

const MODES: readonly { value: ThemeMode; label: string }[] = [
  { value: "system", label: "System" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

function themeLabel(name: string) {
  return name
    .split("-")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

function ThemePreview({
  theme,
  selected,
  onSelect,
  onPreview,
  onPreviewEnd,
}: {
  theme: ThemeDefinition;
  selected: boolean;
  onSelect: () => void;
  onPreview: (theme: string) => void;
  onPreviewEnd: () => void;
}) {
  return (
    <button
      type="button"
      data-theme={theme.name}
      onClick={onSelect}
      onPointerEnter={(event: PointerEvent<HTMLButtonElement>) => {
        if (event.pointerType !== "touch") onPreview(theme.name);
      }}
      onPointerLeave={(event: PointerEvent<HTMLButtonElement>) => {
        if (event.pointerType !== "touch") onPreviewEnd();
      }}
      onFocus={() => onPreview(theme.name)}
      onBlur={onPreviewEnd}
      aria-pressed={selected}
      className={cx(
        "overflow-hidden rounded-box border bg-base-100 text-left text-base-content shadow-sm transition-shadow hover:shadow-md",
        selected ? "border-primary ring-2 ring-primary" : "border-base-300",
      )}
    >
      <div className="flex h-16 items-center gap-1.5 bg-base-200 px-3">
        <span className="size-5 rounded-selector bg-primary" />
        <span className="size-5 rounded-selector bg-secondary" />
        <span className="size-5 rounded-selector bg-accent" />
        <span className="ml-auto rounded-field bg-neutral px-2 py-1 text-xs text-neutral-content">
          Aa
        </span>
      </div>
      <div className="flex items-center gap-2 px-3 py-2 text-sm font-medium">
        <span className="truncate">{themeLabel(theme.name)}</span>
        {selected && <Check className="ml-auto size-4 text-primary" aria-hidden="true" />}
      </div>
    </button>
  );
}

export function AppearanceCard({ userId }: { userId: string }) {
  const { preference, previewTheme, clearThemePreview, setMode, setLightTheme, setDarkTheme } =
    useTheme();
  const updateUser = useUserPatch(userId);
  const previewTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cancelScheduledPreview = useCallback(() => {
    if (previewTimer.current !== null) {
      clearTimeout(previewTimer.current);
      previewTimer.current = null;
    }
  }, []);

  const schedulePreview = useCallback(
    (theme: string) => {
      cancelScheduledPreview();
      previewTimer.current = setTimeout(() => previewTheme(theme), 300);
    },
    [cancelScheduledPreview, previewTheme],
  );

  const schedulePreviewEnd = useCallback(() => {
    cancelScheduledPreview();
    previewTimer.current = setTimeout(clearThemePreview, 300);
  }, [cancelScheduledPreview, clearThemePreview]);

  useEffect(
    () => () => {
      cancelScheduledPreview();
      clearThemePreview();
    },
    [cancelScheduledPreview, clearThemePreview],
  );

  const persist = useCallback(
    (body: UserUpdate, rollback: () => void) => {
      updateUser.mutate(body, {
        onError: (error) => {
          rollback();
          toast.error("Could not save appearance", { description: error.message });
        },
      });
    },
    [updateUser],
  );

  const selectMode = useCallback(
    (mode: ThemeMode) => {
      const previous = preference.mode;
      setMode(mode);
      persist({ appearanceMode: mode }, () => setMode(previous));
    },
    [persist, preference.mode, setMode],
  );

  const selectLightTheme = useCallback(
    (theme: string) => {
      const previous = preference.lightTheme;
      setLightTheme(theme);
      persist({ appearanceLightTheme: theme }, () => setLightTheme(previous));
    },
    [persist, preference.lightTheme, setLightTheme],
  );

  const selectDarkTheme = useCallback(
    (theme: string) => {
      const previous = preference.darkTheme;
      setDarkTheme(theme);
      persist({ appearanceDarkTheme: theme }, () => setDarkTheme(previous));
    },
    [persist, preference.darkTheme, setDarkTheme],
  );

  return (
    <SettingsSection
      icon={<Palette className="size-5" aria-hidden="true" />}
      title="Appearance"
      description="Choose how Horologia looks in light and dark mode."
    >
      <fieldset>
        <legend className="mb-2 text-sm font-medium">Mode</legend>
        <div className="join">
          {MODES.map((mode) => (
            <button
              key={mode.value}
              type="button"
              className={cx(
                "btn btn-sm join-item",
                preference.mode === mode.value ? "btn-primary" : "btn-ghost border-base-300",
              )}
              aria-pressed={preference.mode === mode.value}
              onClick={() => selectMode(mode.value)}
            >
              {mode.label}
            </button>
          ))}
        </div>
        <p className="mt-2 text-xs text-base-content/60">Hover to preview · Click to select.</p>
      </fieldset>

      {preference.mode !== "dark" && (
        <fieldset>
          <legend className="mb-3 flex items-center gap-2 text-sm font-medium">
            <Sun className="size-4" aria-hidden="true" />
            Light theme
          </legend>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
            {lightThemes.map((theme) => (
              <ThemePreview
                key={theme.name}
                theme={theme}
                selected={preference.lightTheme === theme.name}
                onSelect={() => selectLightTheme(theme.name)}
                onPreview={schedulePreview}
                onPreviewEnd={schedulePreviewEnd}
              />
            ))}
          </div>
        </fieldset>
      )}

      {preference.mode !== "light" && (
        <fieldset>
          <legend className="mb-3 flex items-center gap-2 text-sm font-medium">
            <Moon className="size-4" aria-hidden="true" />
            Dark theme
          </legend>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
            {darkThemes.map((theme) => (
              <ThemePreview
                key={theme.name}
                theme={theme}
                selected={preference.darkTheme === theme.name}
                onSelect={() => selectDarkTheme(theme.name)}
                onPreview={schedulePreview}
                onPreviewEnd={schedulePreviewEnd}
              />
            ))}
          </div>
        </fieldset>
      )}
    </SettingsSection>
  );
}
