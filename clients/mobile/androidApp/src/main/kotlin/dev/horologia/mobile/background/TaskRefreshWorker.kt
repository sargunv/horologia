package dev.horologia.mobile.background

import android.Manifest
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import androidx.glance.appwidget.updateAll
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import dev.horologia.mobile.auth.AndroidCredentialStore
import dev.horologia.mobile.domain.MobileTask
import dev.horologia.mobile.domain.RepositoryException
import dev.horologia.mobile.domain.SessionScope
import dev.horologia.mobile.navigation.HorologiaDeepLinks
import dev.horologia.mobile.navigation.SemanticDestination
import dev.horologia.mobile.persistence.CachedTask
import dev.horologia.mobile.persistence.DatabaseDriverFactory
import dev.horologia.mobile.persistence.SnapshotCache
import dev.horologia.mobile.repositories.GeneratedMobileRepository
import dev.horologia.mobile.widgets.MyTasksGlanceWidget
import dev.horologia.mobile.widgets.TaskWidgetStore
import dev.horologia.mobile.widgets.WidgetScope
import dev.horologia.mobile.widgets.projectTaskWidgetSnapshotV1
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone
import kotlin.coroutines.cancellation.CancellationException

class TaskRefreshWorker(
    appContext: Context,
    parameters: WorkerParameters,
) : CoroutineWorker(appContext, parameters) {
    override suspend fun doWork(): Result {
        val scope = inputScope() ?: return Result.failure()
        val credentials = try {
            val store = AndroidCredentialStore(applicationContext)
            if (store.getActiveAccount(scope.serverId) != scope.accountId) {
                markError(scope, "Sign in again to refresh tasks.")
                return Result.failure()
            }
            store.load(scope.serverId, scope.accountId)
        } catch (cancelled: CancellationException) {
            throw cancelled
        } catch (_: Throwable) {
            markError(scope, "Sign in again to refresh tasks.")
            return Result.failure()
        }
        if (credentials == null) {
            markError(scope, "Sign in again to refresh tasks.")
            return Result.failure()
        }

        val session = SessionScope(scope.serverId, scope.baseUrl, scope.accountId, credentials.accessToken)
        return try {
            val page = GeneratedMobileRepository().myTasks(session, limit = REFRESH_LIMIT)
            val generatedAtMillis = System.currentTimeMillis()
            replaceCache(scope, page.items, generatedAtMillis, page.nextCursor)
            val snapshot = projectTaskWidgetSnapshotV1(
                scope = session,
                tasks = page.items,
                generatedAt = isoTimestamp(generatedAtMillis),
                hasMore = page.nextCursor != null,
            )
            TaskWidgetStore.writeSnapshot(applicationContext, scope, snapshot)
            MyTasksGlanceWidget().updateAll(applicationContext)
            postDueTaskNotifications(scope, page.items)
            Result.success()
        } catch (error: RepositoryException) {
            markError(scope, "Couldn't refresh tasks. Showing saved data.")
            when (error.statusCode) {
                401, 403 -> Result.failure()
                null, 408, 425, 429 -> Result.retry()
                in 500..599 -> Result.retry()
                else -> Result.failure()
            }
        } catch (cancelled: CancellationException) {
            throw cancelled
        } catch (_: Throwable) {
            markError(scope, "Couldn't refresh tasks. Showing saved data.")
            Result.retry()
        }
    }

    private fun inputScope(): WidgetScope? {
        val serverId = inputData.getString(INPUT_SERVER_ID) ?: return null
        val baseUrl = inputData.getString(INPUT_BASE_URL) ?: return null
        val accountId = inputData.getString(INPUT_ACCOUNT_ID) ?: return null
        return runCatching { WidgetScope(serverId, baseUrl, accountId) }.getOrNull()
    }

    private fun replaceCache(scope: WidgetScope, tasks: List<MobileTask>, generatedAtMillis: Long, cursor: String?) {
        val driver = DatabaseDriverFactory(applicationContext).createDriver()
        try {
            SnapshotCache(driver).replaceTasks(
                serverId = scope.serverId,
                accountId = scope.accountId,
                items = tasks.map { task ->
                    CachedTask(
                        id = task.id,
                        spaceSlug = task.spaceSlug,
                        title = task.title,
                        description = task.description,
                        status = task.status,
                        effort = task.effort,
                        priority = task.priority,
                        dueText = task.dueText,
                        tags = task.tags,
                    )
                },
                generatedAt = generatedAtMillis / 1_000,
                cursor = cursor,
                hasMore = cursor != null,
            )
        } finally {
            driver.close()
        }
    }

    private suspend fun markError(scope: WidgetScope, message: String) {
        TaskWidgetStore.writeError(applicationContext, scope, message)
        try {
            MyTasksGlanceWidget().updateAll(applicationContext)
        } catch (cancelled: CancellationException) {
            throw cancelled
        } catch (_: Throwable) {
            // The persisted state remains available for the next system widget update.
        }
    }

    private fun postDueTaskNotifications(scope: WidgetScope, tasks: List<MobileTask>) {
        if (!notificationPermissionGranted()) return
        AndroidBackgroundScheduler.createNotificationChannel(applicationContext)
        val today = localDate(System.currentTimeMillis())
        val due = tasks.filter { task ->
            task.status.lowercase() !in TERMINAL_STATUSES &&
                task.dueText?.take(DATE_LENGTH)?.matches(DATE_PATTERN) == true &&
                task.dueText!!.take(DATE_LENGTH) <= today
        }
        val dueIds = due.mapTo(mutableSetOf(), MobileTask::id)
        val alreadyNotified = TaskWidgetStore.notifiedTaskIds(applicationContext, scope)
        val manager = applicationContext.getSystemService(NotificationManager::class.java)
        due.filterNot { it.id in alreadyNotified }.forEach { task ->
            val deepLink = HorologiaDeepLinks.formatApp(
                SemanticDestination.Task(task.spaceSlug, task.id),
                scope.serverId,
            )
            val intent = Intent(Intent.ACTION_VIEW, Uri.parse(deepLink)).apply {
                setPackage(applicationContext.packageName)
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP)
            }
            val requestCode = "${scope.storageId}\u0000${task.id}".hashCode()
            val pendingIntent = PendingIntent.getActivity(
                applicationContext,
                requestCode,
                intent,
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
            )
            val notification = NotificationCompat.Builder(applicationContext, AndroidBackgroundScheduler.NOTIFICATION_CHANNEL_ID)
                .setSmallIcon(applicationContext.applicationInfo.icon)
                .setContentTitle(task.title)
                .setContentText(task.dueText?.let { "Due $it" } ?: "Task is due")
                .setContentIntent(pendingIntent)
                .setAutoCancel(true)
                .setCategory(NotificationCompat.CATEGORY_REMINDER)
                .build()
            manager.notify(requestCode, notification)
        }
        TaskWidgetStore.setNotifiedTaskIds(applicationContext, scope, dueIds)
    }

    private fun notificationPermissionGranted(): Boolean =
        Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
            ContextCompat.checkSelfPermission(applicationContext, Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED

    companion object {
        const val INPUT_SERVER_ID = "server_id"
        const val INPUT_BASE_URL = "base_url"
        const val INPUT_ACCOUNT_ID = "account_id"
        private const val REFRESH_LIMIT = 50
        private const val DATE_LENGTH = 10
        private val DATE_PATTERN = Regex("\\d{4}-\\d{2}-\\d{2}")
        private val TERMINAL_STATUSES = setOf("done", "completed", "cancelled", "canceled")

        private fun isoTimestamp(epochMillis: Long): String =
            SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'", Locale.US).apply {
                timeZone = TimeZone.getTimeZone("UTC")
            }.format(Date(epochMillis))

        private fun localDate(epochMillis: Long): String =
            SimpleDateFormat("yyyy-MM-dd", Locale.US).format(Date(epochMillis))
    }
}
