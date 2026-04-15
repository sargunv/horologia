export function shouldUseDocumentNavigation(target: string): boolean {
  return (
    target.startsWith("/oauth/") ||
    target.startsWith("/auth/") ||
    target.startsWith("/.well-known/") ||
    target.startsWith("/mcp/.well-known/")
  );
}

export function navigateToTarget(
  target: string,
  navigate: (options: { to: string }) => void | Promise<void>,
): void {
  if (shouldUseDocumentNavigation(target)) {
    window.location.assign(target);
    return;
  }
  void navigate({ to: target });
}

export function buildOIDCLoginURL(redirect?: string): string {
  const params = new URLSearchParams();
  if (redirect) {
    params.set("redirect", redirect);
  }
  const query = params.toString();
  return `/api/auth/oidc${query ? `?${query}` : ""}`;
}
