export function navigateToTarget(
  target: string,
  navigate: (options: { to: string }) => void | Promise<void>,
): void {
  if (
    target.startsWith("/oauth/") ||
    target.startsWith("/auth/") ||
    target.startsWith("/.well-known/") ||
    target.startsWith("/mcp/.well-known/")
  ) {
    window.location.assign(target);
    return;
  }

  void navigate({ to: target });
}
