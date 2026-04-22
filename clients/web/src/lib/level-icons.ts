/**
 * Icon utilities for effort, priority, and status levels.
 * Uses ALL Lucide icons — names stored in kebab-case in the DB/API.
 */

import { CircleHelp, icons, type LucideIcon } from "lucide-react";

// ─── Naming Conversion ──────────────────────────────────────────────────────

/** Convert PascalCase (lucide component key) to kebab-case (API/DB format). */
function pascalToKebab(str: string): string {
  return str
    .replace(/([a-z])(\d)/g, "$1-$2")
    .replace(/([a-z\d])([A-Z])/g, "$1-$2")
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1-$2")
    .toLowerCase();
}

// ─── Icon Registry ──────────────────────────────────────────────────────────

/** All Lucide icons, keyed by kebab-case name. Built once at module init. */
const ALL_ICONS = new Map<string, LucideIcon>();

/** All kebab-case icon names, sorted alphabetically. */
const ALL_ICON_NAMES: string[] = [];

for (const [pascal, component] of Object.entries(icons)) {
  const kebab = pascalToKebab(pascal);
  ALL_ICONS.set(kebab, component);
  ALL_ICON_NAMES.push(kebab);
}

ALL_ICON_NAMES.sort();

export { ALL_ICON_NAMES };

/** Fallback icon shown when an icon name is unrecognized or empty. */
export const FALLBACK_ICON: LucideIcon = CircleHelp;

/**
 * Resolve an icon name (kebab-case) to its Lucide component.
 * Falls back to looking up by PascalCase if kebab lookup fails.
 * Returns FALLBACK_ICON for empty/unrecognized names.
 */
export function getIcon(name: string | null | undefined): LucideIcon {
  if (!name) return FALLBACK_ICON;
  return ALL_ICONS.get(name) ?? FALLBACK_ICON;
}

// ─── Suggested Icon Sets ────────────────────────────────────────────────────

/** Suggested icons for effort level pickers. */
export const EFFORT_SUGGESTED_ICONS = ["feather", "leaf", "gauge", "mountain", "flame", "rocket"];

/** Suggested icons for priority level pickers. */
export const PRIORITY_SUGGESTED_ICONS = [
  "signal-low",
  "signal-medium",
  "signal-high",
  "flag",
  "alert-triangle",
  "siren",
];

/** Suggested icons for status pickers. */
export const STATUS_SUGGESTED_ICONS = [
  "circle",
  "circle-dot",
  "loader",
  "circle-check",
  "circle-x",
  "ban",
];

/**
 * Search icons by name substring match. Returns up to `limit` results.
 * When query is empty, returns the suggested icons for the given field type.
 */
export function searchIcons(query: string, suggested: string[], limit = 8): string[] {
  if (!query) return suggested;
  const q = query.toLowerCase();
  const results: string[] = [];
  for (const name of ALL_ICON_NAMES) {
    if (name.includes(q)) {
      results.push(name);
      if (results.length >= limit) break;
    }
  }
  return results;
}
