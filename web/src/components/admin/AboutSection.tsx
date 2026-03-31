import { queryOptions, useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { Check, ExternalLink, Heart, Info, X } from "lucide-react";
import { authConfigQueryOptions } from "../../lib/queries.ts";
import { SettingsSection } from "../space-settings/SettingsSection.tsx";

const healthQueryOptions = queryOptions({
  queryKey: ["health"],
  queryFn: async () => {
    const res = await fetch("/healthz");
    if (!res.ok) return { status: "error" };
    const data: { status: string } = await res.json();
    return data;
  },
  refetchInterval: 30_000,
});

function StatusBadge({ ok }: { ok: boolean }) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-base px-2 py-0.5 text-xs font-medium ${
        ok ? "preset-filled-success-500" : "preset-filled-error-500"
      }`}
    >
      {ok ? (
        <Check className="size-3" aria-hidden="true" />
      ) : (
        <X className="size-3" aria-hidden="true" />
      )}
      {ok ? "Healthy" : "Unhealthy"}
    </span>
  );
}

function EnabledBadge({ enabled, label }: { enabled: boolean; label: string }) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-base px-2 py-0.5 text-xs font-medium ${
        enabled ? "preset-filled-primary-500" : "preset-tonal-surface"
      }`}
    >
      {enabled ? "Enabled" : "Disabled"}
      <span className="sr-only">: {label}</span>
    </span>
  );
}

function InfoRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-4 border-b border-surface-200-800 py-3 last:border-b-0">
      <span className="text-surface-600-400 w-32 shrink-0 text-sm">{label}</span>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}

export function AboutSection() {
  const { data: authConfig } = useSuspenseQuery(authConfigQueryOptions);
  const { data: health } = useQuery(healthQueryOptions);

  return (
    <div className="flex flex-col gap-6">
      <SettingsSection
        icon={<Info className="size-5" aria-hidden="true" />}
        title="Status"
        description="Instance health and configuration."
      >
        <InfoRow label="Database">
          <StatusBadge ok={health?.status === "ok"} />
        </InfoRow>
        <InfoRow label="Password auth">
          <EnabledBadge enabled={authConfig.password.enabled} label="password authentication" />
        </InfoRow>
        <InfoRow label="OIDC">
          <EnabledBadge enabled={authConfig.oidc.enabled} label="OIDC authentication" />
        </InfoRow>
        {authConfig.oidc.enabled && (
          <>
            <InfoRow label="OIDC provider">
              <span className="text-sm">{authConfig.oidc.label}</span>
            </InfoRow>
            <InfoRow label="Auto-redirect">
              <EnabledBadge enabled={authConfig.oidc.autoRedirect} label="OIDC auto-redirect" />
            </InfoRow>
          </>
        )}
      </SettingsSection>

      <div className="text-surface-500 flex items-center justify-center gap-1.5 py-4 text-xs">
        <a
          href="https://github.com/sargunv/tend"
          target="_blank"
          rel="noopener noreferrer"
          className="hover:text-surface-700-300 inline-flex items-center gap-1 transition-colors"
        >
          Tend
          <ExternalLink className="size-3" aria-hidden="true" />
        </a>
        <span>·</span>
        Made with <Heart className="size-3" aria-hidden="true" /> for self-hosters
      </div>
    </div>
  );
}
