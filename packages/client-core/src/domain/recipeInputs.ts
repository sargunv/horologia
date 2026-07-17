export interface ParsedDuration {
  minutes: number;
  label: string;
}

export interface ParsedYield {
  amount: number;
  unit: string;
  label: string;
}

export interface ParsedIngredientQuantity {
  quantity?: number;
  quantityMax?: number;
  unit: string;
}

export function formatDuration(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const remaining = minutes % 60;
  if (hours === 0) return `${remaining} min`;
  if (remaining === 0) return `${hours}h`;
  return `${hours}h ${remaining}m`;
}

export function parseDurationInput(input: string): ParsedDuration | null {
  const normalized = input.trim().toLowerCase();
  if (!normalized) return null;

  if (/^\d+$/u.test(normalized)) {
    const minutes = Number(normalized);
    return Number.isSafeInteger(minutes) ? { minutes, label: formatDuration(minutes) } : null;
  }

  const token = /(\d+(?:\.\d+)?)\s*(hours?|hrs?|h|minutes?|mins?|m)/guy;
  let minutes = 0;
  let cursor = 0;
  let matched = false;
  while (cursor < normalized.length) {
    token.lastIndex = cursor;
    const match = token.exec(normalized);
    if (!match) return null;
    matched = true;
    const amount = Number(match[1]);
    const unit = match[2]!;
    minutes += unit.startsWith("h") ? amount * 60 : amount;
    cursor = token.lastIndex;
    while (normalized[cursor] === " ") cursor += 1;
  }

  if (!matched || !Number.isSafeInteger(minutes) || minutes < 0) return null;
  return { minutes, label: formatDuration(minutes) };
}

export function formatYield(amount: number, unit: string): string {
  const displayUnit = amount === 1 && unit.toLowerCase() === "servings" ? "serving" : unit;
  return `${amount.toLocaleString()} ${displayUnit}`;
}

export function parseYieldInput(input: string): ParsedYield | null {
  const match = input.trim().match(/^(\d+(?:\.\d+)?)\s*(.*)$/u);
  if (!match) return null;
  const amount = Number(match[1]);
  const unit = match[2]!.trim() || "servings";
  if (!Number.isFinite(amount) || amount <= 0 || unit.length > 100) return null;
  return { amount, unit, label: formatYield(amount, unit) };
}

function parseAmountPrefix(input: string): { value: number; rest: string } | null {
  const mixed = input.match(/^(\d+)\s+(\d+)\/(\d+)(.*)$/u);
  if (mixed) {
    const denominator = Number(mixed[3]);
    if (denominator === 0) return null;
    return {
      value: Number(mixed[1]) + Number(mixed[2]) / denominator,
      rest: mixed[4]!,
    };
  }

  const fraction = input.match(/^(\d+)\/(\d+)(.*)$/u);
  if (fraction) {
    const denominator = Number(fraction[2]);
    if (denominator === 0) return null;
    return { value: Number(fraction[1]) / denominator, rest: fraction[3]! };
  }

  const decimal = input.match(/^(\d+(?:\.\d+)?)(.*)$/u);
  if (!decimal) return null;
  return { value: Number(decimal[1]), rest: decimal[2]! };
}

export function parseIngredientQuantity(input: string): ParsedIngredientQuantity | null {
  const trimmed = input.trim();
  if (!trimmed) return { unit: "" };

  const first = parseAmountPrefix(trimmed);
  if (!first) return /^\d/u.test(trimmed) ? null : { unit: trimmed };

  let rest = first.rest.trim();
  if (rest.startsWith("/")) return null;
  let quantityMax: number | undefined;
  if (/^[-–]/u.test(rest)) {
    const second = parseAmountPrefix(rest.replace(/^[-–]\s*/u, ""));
    if (!second) return null;
    quantityMax = second.value;
    rest = second.rest.trim();
  }

  if (
    !Number.isFinite(first.value) ||
    first.value < 0 ||
    (quantityMax !== undefined && (!Number.isFinite(quantityMax) || quantityMax < first.value))
  ) {
    return null;
  }

  return {
    quantity: first.value,
    ...(quantityMax === undefined ? {} : { quantityMax }),
    unit: rest,
  };
}

export function formatIngredientQuantity(
  quantity: number | null | undefined,
  quantityMax: number | null | undefined,
  unit: string,
): string {
  const amount = quantity == null ? "" : String(quantity);
  const maximum = quantityMax == null ? "" : String(quantityMax);
  const range = amount && maximum ? `${amount}–${maximum}` : amount;
  return [range, unit.trim()].filter(Boolean).join(" ");
}
