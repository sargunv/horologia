package dev.horologia.mobile.widget

import androidx.glance.appwidget.GlanceAppWidgetManager
import expo.modules.kotlin.functions.Coroutine
import expo.modules.kotlin.modules.Module
import expo.modules.kotlin.modules.ModuleDefinition

class HorologiaAndroidWidgetModule : Module() {
  override fun definition() = ModuleDefinition {
    Name("HorologiaAndroidWidget")

    AsyncFunction("publishSnapshot") Coroutine { snapshotJson: String ->
      val context = requireNotNull(appContext.reactContext)
      context
        .getSharedPreferences(MyTasksWidget.PREFERENCES_NAME, 0)
        .edit()
        .putString(MyTasksWidget.SNAPSHOT_KEY, snapshotJson)
        .apply()

      GlanceAppWidgetManager(context)
        .getGlanceIds(MyTasksWidget::class.java)
        .forEach { MyTasksWidget().update(context, it) }
      Unit
    }

    val clearSnapshot: suspend () -> Unit = {
      val context = requireNotNull(appContext.reactContext)
      context
        .getSharedPreferences(MyTasksWidget.PREFERENCES_NAME, 0)
        .edit()
        .remove(MyTasksWidget.SNAPSHOT_KEY)
        .apply()

      GlanceAppWidgetManager(context)
        .getGlanceIds(MyTasksWidget::class.java)
        .forEach { MyTasksWidget().update(context, it) }
      Unit
    }
    AsyncFunction("clearSnapshot").SuspendBody(clearSnapshot)
  }
}
