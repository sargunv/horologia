export function shouldUseDocumentNavigation(target: string): boolean {
  return (
    target.startsWith("/oauth/") ||
    target.startsWith("/auth/") ||
    target.startsWith("/.well-known/") ||
    target.startsWith("/mcp/.well-known/")
  );
}
