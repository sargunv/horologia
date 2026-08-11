import type { ReactNode } from "react";

export function SettingsSection({
  icon,
  title,
  description,
  children,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  children?: ReactNode;
}) {
  return (
    <section className="surface-card specimen-sheet flex flex-col gap-4 rounded-box border border-base-300 bg-base-100 p-6">
      <header className="specimen-sheet-header border-b border-base-300 pb-4">
        <div className="flex items-center gap-3">
          <span className="text-base-content/70" aria-hidden="true">
            {icon}
          </span>
          <h2 className="specimen-title text-lg font-semibold leading-tight">{title}</h2>
        </div>
        <p className="specimen-caption mt-1 text-sm text-base-content/70 sm:pl-7">{description}</p>
      </header>
      <div className="specimen-sheet-body flex flex-col gap-4">{children}</div>
    </section>
  );
}
