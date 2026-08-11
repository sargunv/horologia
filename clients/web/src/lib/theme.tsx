import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

export type ThemeMode = "system" | "light" | "dark";
export type ThemeScheme = "light" | "dark";

export interface ThemeDefinition {
  readonly name: string;
  readonly scheme: ThemeScheme;
}

export interface AppearancePreference {
  readonly mode: ThemeMode;
  readonly lightTheme: string;
  readonly darkTheme: string;
}

const STORAGE_KEY = "horologia.appearance";
const DEFAULT_PREFERENCE: AppearancePreference = {
  mode: "system",
  lightTheme: "frutiger-aero",
  darkTheme: "dark",
};

export const themes: readonly ThemeDefinition[] = __DAISYUI_THEMES__;
export const lightThemes = themes.filter((theme) => theme.scheme === "light");
export const darkThemes = themes.filter((theme) => theme.scheme === "dark");

const lightThemeNames = new Set(lightThemes.map((theme) => theme.name));
const darkThemeNames = new Set(darkThemes.map((theme) => theme.name));
const themesByName = new Map(themes.map((theme) => [theme.name, theme]));

function isMode(value: unknown): value is ThemeMode {
  return value === "system" || value === "light" || value === "dark";
}

function readPreference(): AppearancePreference {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null");
    if (typeof value !== "object" || value === null) return DEFAULT_PREFERENCE;
    const mode = "mode" in value ? value.mode : undefined;
    const lightTheme = "lightTheme" in value ? value.lightTheme : undefined;
    const darkTheme = "darkTheme" in value ? value.darkTheme : undefined;
    return {
      mode: isMode(mode) ? mode : DEFAULT_PREFERENCE.mode,
      lightTheme:
        typeof lightTheme === "string" && lightThemeNames.has(lightTheme)
          ? lightTheme
          : DEFAULT_PREFERENCE.lightTheme,
      darkTheme:
        typeof darkTheme === "string" && darkThemeNames.has(darkTheme)
          ? darkTheme
          : DEFAULT_PREFERENCE.darkTheme,
    };
  } catch {
    return DEFAULT_PREFERENCE;
  }
}

function systemScheme(): ThemeScheme {
  return matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

interface ThemeContextValue {
  readonly preference: AppearancePreference;
  readonly resolvedScheme: ThemeScheme;
  readonly resolvedTheme: string;
  readonly previewTheme: (theme: string) => void;
  readonly clearThemePreview: () => void;
  readonly syncPreference: (preference: AppearancePreference) => void;
  readonly setMode: (mode: ThemeMode) => void;
  readonly setLightTheme: (theme: string) => void;
  readonly setDarkTheme: (theme: string) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [preference, setPreference] = useState(readPreference);
  const [preferredScheme, setPreferredScheme] = useState(systemScheme);
  const [previewedTheme, setPreviewedTheme] = useState<ThemeDefinition | null>(null);

  useEffect(() => {
    const media = matchMedia("(prefers-color-scheme: dark)");
    const update = () => setPreferredScheme(media.matches ? "dark" : "light");
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);

  const resolvedScheme = preference.mode === "system" ? preferredScheme : preference.mode;
  const resolvedTheme = resolvedScheme === "dark" ? preference.darkTheme : preference.lightTheme;
  const activeScheme = previewedTheme?.scheme ?? resolvedScheme;
  const activeTheme = previewedTheme?.name ?? resolvedTheme;

  useEffect(() => {
    document.documentElement.dataset["theme"] = activeTheme;
  }, [activeTheme]);

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(preference));
  }, [preference]);

  const previewTheme = useCallback((theme: string) => {
    setPreviewedTheme(themesByName.get(theme) ?? null);
  }, []);
  const clearThemePreview = useCallback(() => {
    setPreviewedTheme(null);
  }, []);
  const syncPreference = useCallback((next: AppearancePreference) => {
    setPreference({
      mode: next.mode,
      lightTheme: lightThemeNames.has(next.lightTheme)
        ? next.lightTheme
        : DEFAULT_PREFERENCE.lightTheme,
      darkTheme: darkThemeNames.has(next.darkTheme) ? next.darkTheme : DEFAULT_PREFERENCE.darkTheme,
    });
  }, []);

  const setMode = useCallback((mode: ThemeMode) => {
    setPreviewedTheme(null);
    setPreference((current) => ({ ...current, mode }));
  }, []);
  const setLightTheme = useCallback((lightTheme: string) => {
    if (lightThemeNames.has(lightTheme)) {
      setPreference((current) => ({ ...current, lightTheme }));
    }
  }, []);
  const setDarkTheme = useCallback((darkTheme: string) => {
    if (darkThemeNames.has(darkTheme)) {
      setPreference((current) => ({ ...current, darkTheme }));
    }
  }, []);

  const value = useMemo(
    () => ({
      preference,
      resolvedScheme: activeScheme,
      resolvedTheme: activeTheme,
      previewTheme,
      clearThemePreview,
      syncPreference,
      setMode,
      setLightTheme,
      setDarkTheme,
    }),
    [
      preference,
      activeScheme,
      activeTheme,
      previewTheme,
      clearThemePreview,
      syncPreference,
      setMode,
      setLightTheme,
      setDarkTheme,
    ],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  if (!context) throw new Error("useTheme must be used inside ThemeProvider");
  return context;
}
