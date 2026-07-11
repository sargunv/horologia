import type { ReactNode } from "react";

export function ListDetailLayout({
  list,
  detail,
  emptyState,
}: {
  list: ReactNode;
  detail: ReactNode | null;
  emptyState?: ReactNode;
}) {
  return (
    <div className="grid gap-6 lg:grid-cols-[350px_minmax(0,1fr)]">
      <div className={detail ? "hidden min-w-0 lg:block" : "min-w-0"}>{list}</div>

      <div className={!detail ? "hidden min-w-0 lg:block" : "min-w-0"}>
        {detail ?? (
          <div className="hidden flex-col items-center justify-center gap-4 p-12 lg:flex">
            {emptyState}
          </div>
        )}
      </div>
    </div>
  );
}
