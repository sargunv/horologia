import { NativeModule, requireNativeModule } from "expo";

declare class HorologiaAndroidWidgetModule extends NativeModule<Record<string, never>> {
  publishSnapshot(snapshotJson: string): Promise<void>;
  clearSnapshot(): Promise<void>;
}

export default requireNativeModule<HorologiaAndroidWidgetModule>("HorologiaAndroidWidget");
