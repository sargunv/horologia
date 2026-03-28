import { createFileRoute, createLink } from "@tanstack/react-router";
import { useSuspenseQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { spaceQueryOptions } from "../../../../lib/queries.ts";

export const Route = createFileRoute("/_authenticated/spaces/$spaceSlug/settings")({
  component: SpaceSettingsPage,
});

const BackLink = createLink("a");

const sections = [
  {
    title: "General",
    description: "Name, slug, and description for this space.",
  },
  {
    title: "Members",
    description: "Manage who has access to this space.",
  },
  {
    title: "Task Statuses",
    description: "Configure the workflow statuses for tasks.",
  },
  {
    title: "Effort Levels",
    description: "Define effort levels for estimating tasks.",
  },
  {
    title: "Priority Levels",
    description: "Configure priority levels for tasks.",
  },
];

function SpaceSettingsPage() {
  const { spaceSlug } = Route.useParams();
  const { data: space } = useSuspenseQuery(spaceQueryOptions(spaceSlug));

  return (
    <div className="mx-auto max-w-3xl p-6">
      <div className="mb-6">
        <BackLink
          to="/spaces/$spaceSlug"
          params={{ spaceSlug }}
          className="btn preset-outlined-surface-200-800 mb-4 inline-flex items-center gap-2"
        >
          <ArrowLeft className="size-4" />
          Back to {space.name}
        </BackLink>
        <h1 className="h3">{space.name} Settings</h1>
      </div>

      <div className="flex flex-col gap-4">
        {sections.map((section) => (
          <div key={section.title} className="card preset-outlined-surface-200-800 p-6">
            <h2 className="h5">{section.title}</h2>
            <p className="text-surface-600-400 mt-1 text-sm">{section.description}</p>
            <p className="text-surface-500 mt-4 text-sm italic">Coming soon.</p>
          </div>
        ))}

        <div className="card preset-outlined-error-500 p-6">
          <h2 className="h5 text-error-500">Danger Zone</h2>
          <p className="text-surface-600-400 mt-1 text-sm">
            Permanently delete this space and all of its data.
          </p>
          <p className="text-surface-500 mt-4 text-sm italic">Coming soon.</p>
        </div>
      </div>
    </div>
  );
}
