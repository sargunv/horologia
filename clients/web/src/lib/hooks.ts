import { useMemo } from "react";
import { useSuspenseQuery } from "@tanstack/react-query";
import { spaceMembersQueryOptions } from "./queries.ts";
import type { components } from "../api/schema.d.ts";

type SpaceMember = components["schemas"]["SpaceMember"];

export function useSpaceMemberMap(spaceSlug: string): Map<string, SpaceMember> {
  const { data: members } = useSuspenseQuery(spaceMembersQueryOptions(spaceSlug));
  return useMemo(() => new Map(members.map((m) => [m.userId, m])), [members]);
}
