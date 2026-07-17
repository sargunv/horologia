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
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
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
        text = if (snapshot.signedIn) snapshot.tasks.size.toString() else "—",
        style =
          TextStyle(
            color = ColorProvider(Color(0xFF2F6D4B), Color(0xFF79D39D)),
            fontSize = 36.sp,
            fontWeight = FontWeight.Bold,
          ),
      )
      Text(
        text =
          if (!snapshot.signedIn) "Sign in to see your tasks"
          else nextTask?.title ?: "You're all caught up",
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
      if (size.height >= 180.dp) {
        Text(
          text =
            if (snapshot.signedIn) snapshotFreshness(snapshot.generatedAt)
            else "Open Horologia to sign in",
          style =
            TextStyle(
              color = ColorProvider(Color(0xFF66736B), Color(0xFF9BA89F)),
              fontSize = 11.sp,
            ),
        )
      }
    }
  }

  private fun readSnapshot(context: Context): WidgetSnapshot {
    val value =
      context
        .getSharedPreferences(PREFERENCES_NAME, 0)
        .getString(SNAPSHOT_KEY, null)
        ?: return WidgetSnapshot.signedOut

    return runCatching {
        val root = JSONObject(value)
        val tasks = root.getJSONArray("tasks")
        WidgetSnapshot(
          signedIn = true,
          generatedAt = root.getString("generatedAt"),
          tasks = List(tasks.length()) { index ->
            val task = tasks.getJSONObject(index)
            WidgetTask(
              id = task.getString("id"),
              spaceSlug = task.getString("spaceSlug"),
              title = task.getString("title"),
            )
          },
        )
      }
      .getOrDefault(WidgetSnapshot.signedOut)
  }
}

private fun formatGeneratedAt(value: String): String =
  runCatching {
      DateTimeFormatter.ofPattern("h:mm a")
        .withZone(ZoneId.systemDefault())
        .format(Instant.parse(value))
    }
    .getOrDefault("recently")

private fun snapshotFreshness(value: String): String =
  runCatching {
      val generatedAt = Instant.parse(value)
      if (generatedAt.isBefore(Instant.now().minusSeconds(24 * 60 * 60))) {
        "Saved tasks may be out of date"
      } else {
        "Saved ${formatGeneratedAt(value)}"
      }
    }
    .getOrDefault("Saved tasks may be out of date")

private data class WidgetSnapshot(
  val signedIn: Boolean,
  val generatedAt: String,
  val tasks: List<WidgetTask>,
) {
  companion object {
    val signedOut = WidgetSnapshot(signedIn = false, generatedAt = "", tasks = emptyList())
  }
}

private data class WidgetTask(val id: String, val spaceSlug: String, val title: String)

class MyTasksWidgetReceiver : GlanceAppWidgetReceiver() {
  override val glanceAppWidget: GlanceAppWidget = MyTasksWidget()
}
