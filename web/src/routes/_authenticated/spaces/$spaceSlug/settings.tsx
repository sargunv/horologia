import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute, createLink } from "@tanstack/react-router";
import { ArrowLeft, Gauge, Shield, SignalHigh, SquareKanban, Trash2, Users } from "lucide-react";
import type { ReactNode } from "react";
import { spaceQueryOptions } from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/settings")({
  component: SpaceSettingsPage,
});

const BackLink = createLink("a");

function SettingsSection({
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

function SpaceSettingsPage() {
  const { spaceSlug } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));

  return (
    <div className="mx-auto max-w-3xl p-6">
      <BackLink
        to="/spaces/$spaceSlug"
        params={{ spaceSlug }}
        className="text-surface-600-400 hover:text-surface-950-50 mb-4 inline-flex items-center gap-1 text-sm transition-colors"
      >
        <ArrowLeft className="size-4" />
        Back to {space.name}
      </BackLink>

      <h1 className="h3">Space Settings</h1>
      <p className="text-surface-600-400 mt-1">Manage settings for {space.name}.</p>

      <div className="mt-6 flex flex-col gap-4">
        <SettingsSection
          icon={<SquareKanban className="size-5" />}
          title="General"
          description="Name, slug, and description for this space."
        >
          <p className="text-surface-500 text-sm">Coming soon.</p>
        </SettingsSection>

        <SettingsSection
          icon={<Users className="size-5" />}
          title="Members"
          description="Manage who has access to this space."
        >
          <p className="text-surface-500 text-sm">Coming soon.</p>
        </SettingsSection>

        <SettingsSection
          icon={<Shield className="size-5" />}
          title="Task Statuses"
          description="Configure the workflow statuses for tasks in this space."
        >
          <p className="text-surface-500 text-sm">Coming soon.</p>
        </SettingsSection>

        <SettingsSection
          icon={<Gauge className="size-5" />}
          title="Effort Levels"
          description="Define effort levels for estimating task complexity."
        >
          <p className="text-surface-500 text-sm">Coming soon.</p>
        </SettingsSection>

        <SettingsSection
          icon={<SignalHigh className="size-5" />}
          title="Priority Levels"
          description="Configure priority levels for organizing tasks."
        >
          <p className="text-surface-500 text-sm">Coming soon.</p>
        </SettingsSection>

        <SettingsSection
          icon={<Trash2 className="size-5" />}
          title="Danger Zone"
          description="Irreversible actions for this space."
        >
          <p className="text-surface-500 text-sm">Coming soon.</p>
        </SettingsSection>
      </div>
    </div>
  );
}
