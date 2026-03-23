interface ErrorDisplayProps {
  error: unknown;
}

export function ErrorDisplay({ error }: ErrorDisplayProps) {
  const message =
    error instanceof Error
      ? error.message
      : typeof error === "object" && error !== null && "message" in error
        ? String((error as { message: unknown }).message)
        : "An unexpected error occurred";

  return (
    <div className="alert preset-filled-error-500" role="alert">
      <p>{message}</p>
    </div>
  );
}
