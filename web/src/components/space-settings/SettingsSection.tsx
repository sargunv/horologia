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
    <div className="card preset-outlined-surface-200-800 flex flex-col gap-4 p-6">
      <div className="flex items-center gap-3">
        <span className="text-surface-600-400">{icon}</span>
        <div>
          <h2 className="font-medium">{title}</h2>
          <p className="text-surface-600-400 text-sm">{description}</p>
        </div>
      </div>
      {children}
    </div>
  );
}
