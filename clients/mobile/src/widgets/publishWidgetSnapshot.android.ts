import type { WidgetSnapshotV1 } from "@horologia/client-core";

import HorologiaAndroidWidget from "../../modules/horologia-android-widget/src/HorologiaAndroidWidgetModule";

export async function publishWidgetSnapshot(snapshot: WidgetSnapshotV1): Promise<void> {
  await HorologiaAndroidWidget.publishSnapshot(JSON.stringify(snapshot));
}

export async function clearWidgetSnapshot(): Promise<void> {
  await HorologiaAndroidWidget.clearSnapshot();
}
