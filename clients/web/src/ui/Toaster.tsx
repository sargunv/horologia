import { Toaster as SonnerToaster } from "sonner";
import { useTheme } from "../lib/theme.tsx";

/**
 * Toaster mount — place once in the app shell. Threads daisyUI semantic
 * colors (base/success/error/warning/info + -content variants) into
 * sonner's internal CSS vars via `style`, so light/dark theme flips for
 * free. `toastOptions.classNames` adds the app-wide daisyUI radius +
 * shadow. Consumers call `toast.*` from `sonner` directly, or
 * `notifyStaleData()` from `lib/toaster.ts`.
 */
/**
 * Sonner exposes its palette via custom CSS properties on the `<Toaster>`
 * element; map each to the matching daisyUI semantic color token so
 * toasts flip with the theme. Typed as `Record<string, string>` because
 * React's `CSSProperties` doesn't recognize custom `--*` props.
 */
const TOKEN_STYLE: Record<string, string> = {
  "--normal-bg": "var(--color-base-100)",
  "--normal-text": "var(--color-base-content)",
  "--normal-border": "var(--color-base-300)",
  "--success-bg": "var(--color-success)",
  "--success-text": "var(--color-success-content)",
  "--success-border": "var(--color-success)",
  "--error-bg": "var(--color-error)",
  "--error-text": "var(--color-error-content)",
  "--error-border": "var(--color-error)",
  "--warning-bg": "var(--color-warning)",
  "--warning-text": "var(--color-warning-content)",
  "--warning-border": "var(--color-warning)",
  "--info-bg": "var(--color-info)",
  "--info-text": "var(--color-info-content)",
  "--info-border": "var(--color-info)",
};

export function Toaster() {
  const { resolvedScheme } = useTheme();
  return (
    <SonnerToaster
      position="top-right"
      theme={resolvedScheme}
      closeButton
      style={TOKEN_STYLE}
      toastOptions={{
        classNames: {
          toast: "rounded-box shadow-lg",
          title: "font-medium",
          description: "text-base-content/70",
        },
      }}
    />
  );
}
