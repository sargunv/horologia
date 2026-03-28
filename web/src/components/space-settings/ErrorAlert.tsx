import { CircleAlert } from "lucide-react";

export function ErrorAlert({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="preset-filled-error-500 flex items-center gap-2 rounded-base px-3 py-2 text-sm"
    >
      <CircleAlert className="size-4 shrink-0" aria-hidden="true" />
      {message}
    </div>
  );
}
