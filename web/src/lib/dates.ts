import * as chrono from "chrono-node/en";

export function toISODate(date: Date): string {
  const y = date.getFullYear().toString().padStart(4, "0");
  const m = (date.getMonth() + 1).toString().padStart(2, "0");
  const d = date.getDate().toString().padStart(2, "0");
  return `${y}-${m}-${d}`;
}

export function formatDateDisplay(date: Date): string {
  return date.toLocaleDateString(undefined, {
    weekday: "short",
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function parseDateInput(input: string): { label: string; value: string } | null {
  if (!input.trim()) return null;
  const parsed = chrono.parseDate(input);
  if (!parsed) return null;
  return { label: formatDateDisplay(parsed), value: toISODate(parsed) };
}

export function addDays(date: Date, days: number): Date {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}
