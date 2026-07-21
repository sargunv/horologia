package dev.horologia.mobile.widgets

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetReceiver
import androidx.glance.appwidget.action.actionStartActivity
import androidx.glance.appwidget.provideContent
import androidx.glance.background
import androidx.glance.action.clickable
import androidx.glance.layout.Column
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.layout.width
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import androidx.glance.unit.ColorProvider
import dev.horologia.mobile.R
import dev.horologia.mobile.navigation.HorologiaDeepLinks
import dev.horologia.mobile.navigation.SemanticDestination

private val WidgetBackground = ColorProvider(Color(0xFFFAF8FF))
private val WidgetPrimaryText = ColorProvider(Color(0xFF23232A))
private val WidgetWarningText = ColorProvider(Color(0xFF964B00))
private val WidgetSecondaryText = ColorProvider(Color(0xFF504F58))
private val WidgetActionText = ColorProvider(Color(0xFF444791))

class MyTasksGlanceWidget : GlanceAppWidget() {
    override suspend fun provideGlance(context: Context, id: GlanceId) {
        val content = TaskWidgetStore.read(context)
        provideContent { TaskWidgetContent(context, content) }
    }
}

class MyTasksWidgetReceiver : GlanceAppWidgetReceiver() {
    override val glanceAppWidget: GlanceAppWidget = MyTasksGlanceWidget()
}

@Composable
private fun TaskWidgetContent(context: Context, content: WidgetContent) {
    Column(
        modifier = GlanceModifier
            .fillMaxSize()
            .background(WidgetBackground)
            .padding(14.dp),
    ) {
        Text(
            text = "My Tasks",
            style = TextStyle(
                color = WidgetPrimaryText,
                fontSize = 18.sp,
                fontWeight = FontWeight.Bold,
            ),
        )
        Spacer(GlanceModifier.height(8.dp))

        val snapshot = content.snapshot
        when {
            content.scope == null -> StatusText("Open Horologia and sign in to show your tasks.")
            snapshot == null && content.error != null -> StatusText(content.error)
            snapshot == null -> StatusText("Tasks will appear after the next refresh.")
            snapshot.rows.isEmpty() -> StatusText("No tasks assigned to you.")
            else -> {
                content.error?.let {
                    Text(
                        text = "Last refresh failed — showing saved tasks",
                        style = TextStyle(color = WidgetWarningText, fontSize = 11.sp),
                    )
                    Spacer(GlanceModifier.height(5.dp))
                }
                snapshot.rows.forEach { task ->
                    val link = HorologiaDeepLinks.formatApp(
                        SemanticDestination.Task(task.spaceSlug, task.id),
                        snapshot.serverId,
                    )
                    Row(
                        modifier = GlanceModifier
                            .fillMaxWidth()
                            .clickable(
                                actionStartActivity(
                                    Intent(Intent.ACTION_VIEW, Uri.parse(link)).apply {
                                        setPackage(context.packageName)
                                        addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP)
                                    },
                                ),
                            )
                            .padding(vertical = 5.dp),
                    ) {
                        Text(
                            text = task.title,
                            modifier = GlanceModifier.defaultWeight(),
                            style = TextStyle(color = WidgetPrimaryText, fontSize = 14.sp),
                            maxLines = 2,
                        )
                        task.due?.takeIf(String::isNotBlank)?.let { due ->
                            Spacer(GlanceModifier.width(8.dp))
                            Text(
                                text = due,
                                style = TextStyle(color = WidgetSecondaryText, fontSize = 11.sp),
                                maxLines = 1,
                            )
                        }
                    }
                }
                if (snapshot.hasMore) {
                    Text(
                        text = "+ more tasks",
                        modifier = GlanceModifier.clickable(
                            actionStartActivity(
                                Intent(
                                    Intent.ACTION_VIEW,
                                    Uri.parse(HorologiaDeepLinks.formatApp(SemanticDestination.Tasks, snapshot.serverId)),
                                ).apply { setPackage(context.packageName) },
                            ),
                        ),
                        style = TextStyle(color = WidgetActionText, fontSize = 12.sp),
                    )
                }
            }
        }
    }
}

@Composable
private fun StatusText(message: String) {
    Text(
        text = message,
        style = TextStyle(color = WidgetSecondaryText, fontSize = 13.sp),
    )
}
