/**
 * Tiny classname combiner. Joins truthy string parts with a single space;
 * falsy parts are skipped. Use this instead of template literals so absent
 * classes don't leave trailing whitespace.
 */
export function cx(...parts: ReadonlyArray<string | false | null | undefined>): string {
  let out = "";
  for (const part of parts) {
    if (!part) continue;
    out = out.length === 0 ? part : `${out} ${part}`;
  }
  return out;
}
