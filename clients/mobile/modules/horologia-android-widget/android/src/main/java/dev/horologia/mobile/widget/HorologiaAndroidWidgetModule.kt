package dev.horologia.mobile.widget

import androidx.glance.appwidget.GlanceAppWidgetManager
import expo.modules.kotlin.modules.Module
import expo.modules.kotlin.modules.ModuleDefinition
import kotlinx.coroutines.launch

class HorologiaAndroidWidgetModule : Module() {
  override fun definition() = ModuleDefinition {
    Name("HorologiaAndroidWidget")

    AsyncFunction("publishSnapshot") { snapshotJson: String ->
      val context = requireNotNull(appContext.reactContext)
      context
        .getSharedPreferences(MyTasksWidget.PREFERENCES_NAME, 0)
        .edit()
        .putString(MyTasksWidget.SNAPSHOT_KEY, snapshotJson)
        .apply()

      appContext.backgroundCoroutineScope.launch {
        GlanceAppWidgetManager(context)
          .getGlanceIds(MyTasksWidget::class.java)
          .forEach { MyTasksWidget().update(context, it) }
      }
      Unit
    }
  }
}
