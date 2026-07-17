package dev.horologia.mobile.widget

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.compose.runtime.Composable
import androidx.glance.GlanceTheme
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.LocalSize
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetReceiver
import androidx.glance.appwidget.action.actionStartActivity
import androidx.glance.appwidget.cornerRadius
import androidx.glance.appwidget.provideContent
import androidx.glance.background
import androidx.glance.color.ColorProvider
import androidx.glance.layout.Column
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.padding
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.json.JSONObject

class MyTasksWidget : GlanceAppWidget() {
  companion object {
    const val PREFERENCES_NAME = "horologia_widget"
    const val SNAPSHOT_KEY = "my_tasks_snapshot_v1"
  }

  override suspend fun provideGlance(context: Context, id: GlanceId) {
    val snapshot = readSnapshot(context)
    provideContent { GlanceTheme { MyTasksContent(snapshot) } }
  }

  @Composable
  private fun MyTasksContent(snapshot: WidgetSnapshot) {
    val size = LocalSize.current
    val nextTask = snapshot.tasks.firstOrNull()
    val deepLink =
      Intent(
          Intent.ACTION_VIEW,
          Uri.parse(
            nextTask?.let { "horologia://task/${it.spaceSlug}/${it.id}" } ?: "horologia://"
          ),
        )
        .setPackage("dev.horologia.mobile")

    Column(
      modifier =
        GlanceModifier.fillMaxSize()
          .background(ColorProvider(Color(0xFFF4F8F5), Color(0xFF18211B)))
          .cornerRadius(24.dp)
          .padding(16.dp)
          .clickable(actionStartActivity(deepLink))
    ) {
      Text(
        text = "My Tasks",
        style =
          TextStyle(
            color = ColorProvider(Color(0xFF152019), Color(0xFFE8F0EA)),
            fontSize = 18.sp,
            fontWeight = FontWeight.Bold,
          ),
      )
      Text(
        text = snapshot.tasks.size.toString(),
        style =
          TextStyle(
            color = ColorProvider(Color(0xFF2F6D4B), Color(0xFF79D39D)),
            fontSize = 36.sp,
            fontWeight = FontWeight.Bold,
          ),
      )
      Text(
        text = nextTask?.title ?: "You're all caught up",
        style = TextStyle(color = ColorProvider(Color(0xFF26322A), Color(0xFFD6DED8))),
      )
      if (size.width >= 250.dp) {
        snapshot.tasks.drop(1).take(if (size.height >= 220.dp) 5 else 2).forEach { task ->
          Text(
            text = task.title,
            style = TextStyle(color = ColorProvider(Color(0xFF455149), Color(0xFFB7C2BA))),
          )
        }
      }
    }
  }

  private fun readSnapshot(context: Context): WidgetSnapshot {
    val value =
      context
        .getSharedPreferences(PREFERENCES_NAME, 0)
        .getString(SNAPSHOT_KEY, null)
        ?: return WidgetSnapshot.demo

    return runCatching {
        val root = JSONObject(value)
        val tasks = root.getJSONArray("tasks")
        WidgetSnapshot(
          List(tasks.length()) { index ->
            val task = tasks.getJSONObject(index)
            WidgetTask(
              id = task.getString("id"),
              spaceSlug = task.getString("spaceSlug"),
              title = task.getString("title"),
            )
          }
        )
      }
      .getOrDefault(WidgetSnapshot.demo)
  }
}

private data class WidgetSnapshot(val tasks: List<WidgetTask>) {
  companion object {
    val demo =
      WidgetSnapshot(
        listOf(
          WidgetTask("1", "home", "Water the herbs"),
          WidgetTask("2", "home", "Change the air filter"),
          WidgetTask("3", "kitchen", "Plan next week's meals"),
        )
      )
  }
}

private data class WidgetTask(val id: String, val spaceSlug: String, val title: String)

class MyTasksWidgetReceiver : GlanceAppWidgetReceiver() {
  override val glanceAppWidget: GlanceAppWidget = MyTasksWidget()
}
