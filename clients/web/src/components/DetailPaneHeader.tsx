import type { ReactNode } from "react";

export const DETAIL_PANE_TITLE_CLASS = "text-xl font-semibold";

export function DetailPaneHeader({
  backLink,
  breadcrumb,
  actions,
  title,
  description,
}: {
  backLink: ReactNode;
  breadcrumb: ReactNode;
  actions?: ReactNode;
  title: ReactNode;
  description?: ReactNode;
}) {
  return (
    <header className="space-y-4">
      {backLink}

      <div className="flex h-8 min-w-0 items-center gap-2">
        <nav aria-label="Breadcrumb" className="flex min-w-0 items-center">
          {breadcrumb}
        </nav>
        {actions && <div className="ml-auto shrink-0">{actions}</div>}
      </div>

      <div>
        {title}
        {description && <div className="text-base-content/70 mt-1">{description}</div>}
      </div>
    </header>
  );
}
