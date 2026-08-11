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
    <div className="surface-card flex flex-col gap-4 rounded-box border border-base-300 bg-base-100 p-6">
      <div className="flex items-center gap-3">
        <span className="text-base-content/70">{icon}</span>
        <div>
          <h2 className="font-medium">{title}</h2>
          <p className="text-sm text-base-content/70">{description}</p>
        </div>
      </div>
      {children}
    </div>
  );
}
