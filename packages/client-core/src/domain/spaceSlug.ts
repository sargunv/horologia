export function slugifySpaceName(name: string): string {
  return name
    .toLocaleLowerCase()
    .replace(/[^\p{L}0-9]+/gu, "-")
    .replace(/^-+|-+$/gu, "")
    .slice(0, 100);
}
