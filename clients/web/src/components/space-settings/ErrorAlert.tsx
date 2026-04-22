import { CircleAlert } from "lucide-react";

export function ErrorAlert({ message }: { message: string }) {
  return (
    <div role="alert" className="alert alert-error alert-soft text-sm">
      <CircleAlert className="size-4 shrink-0" aria-hidden="true" />
      {message}
    </div>
  );
}
