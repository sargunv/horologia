export interface WidgetTaskV1 {
  id: string;
  spaceSlug: string;
  title: string;
  due: string | null;
  status: string;
}

export interface WidgetSnapshotV1 {
  version: 1;
  serverId: string;
  accountId: string;
  generatedAt: string;
  tasks: WidgetTaskV1[];
}

export function createWidgetSnapshotV1(
  snapshot: Omit<WidgetSnapshotV1, "version">,
): WidgetSnapshotV1 {
  return { version: 1, ...snapshot };
}
