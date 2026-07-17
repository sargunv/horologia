import { registerWebModule, NativeModule } from "expo";

// HorologiaAndroidWidgetModule is not available on the web platform.
class HorologiaAndroidWidgetModule extends NativeModule<Record<string, never>> {
  async publishSnapshot(_snapshotJson: string): Promise<void> {}
}

export default registerWebModule(HorologiaAndroidWidgetModule, "HorologiaAndroidWidgetModule");
