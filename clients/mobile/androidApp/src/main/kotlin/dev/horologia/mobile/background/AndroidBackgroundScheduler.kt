package dev.horologia.mobile.background

import android.app.NotificationChannel
import android.app.NotificationManager
import android.appwidget.AppWidgetManager
import android.content.BroadcastReceiver
import android.content.Context
import android.content.ComponentName
import android.content.Intent
import android.os.Build
import androidx.work.Constraints
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.workDataOf
import dev.horologia.mobile.widgets.MyTasksWidgetReceiver
import dev.horologia.mobile.widgets.TaskWidgetStore
import dev.horologia.mobile.widgets.WidgetScope
import java.util.concurrent.TimeUnit

/** Entry point for the authorized UI session. It never retains an Activity. */
object AndroidBackgroundScheduler {
    const val NOTIFICATION_CHANNEL_ID = "due_tasks"

    fun scheduleAfterAuthorization(
        context: Context,
        serverId: String,
        baseUrl: String,
        accountId: String,
    ) {
        val applicationContext = context.applicationContext
        val scope = WidgetScope(serverId, baseUrl, accountId)
        TaskWidgetStore.register(applicationContext, scope)
        requestWidgetRefresh(applicationContext)
        createNotificationChannel(applicationContext)
        enqueue(applicationContext, scope, refreshImmediately = true)
    }

    fun cancel(context: Context, serverId: String, accountId: String) {
        val applicationContext = context.applicationContext
        val storageId = WidgetScope.idFor(serverId, accountId)
        WorkManager.getInstance(applicationContext).apply {
            cancelUniqueWork(periodicWorkName(storageId))
            cancelUniqueWork(immediateWorkName(storageId))
        }
        TaskWidgetStore.unregister(applicationContext, serverId, accountId)
        requestWidgetRefresh(applicationContext)
    }

    internal fun reschedulePersisted(context: Context) {
        createNotificationChannel(context)
        TaskWidgetStore.registrations(context).forEach { enqueue(context, it, refreshImmediately = false) }
    }

    private fun requestWidgetRefresh(context: Context) {
        val manager = AppWidgetManager.getInstance(context)
        val receiver = ComponentName(context, MyTasksWidgetReceiver::class.java)
        val widgetIds = manager.getAppWidgetIds(receiver)
        if (widgetIds.isEmpty()) return
        context.sendBroadcast(
            Intent(AppWidgetManager.ACTION_APPWIDGET_UPDATE).apply {
                component = receiver
                putExtra(AppWidgetManager.EXTRA_APPWIDGET_IDS, widgetIds)
            },
        )
    }

    internal fun createNotificationChannel(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = context.getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(
            NotificationChannel(
                NOTIFICATION_CHANNEL_ID,
                "Due tasks",
                NotificationManager.IMPORTANCE_DEFAULT,
            ).apply { description = "Tasks assigned to you that are due" },
        )
    }

    private fun enqueue(context: Context, scope: WidgetScope, refreshImmediately: Boolean) {
        val input = workDataOf(
            TaskRefreshWorker.INPUT_SERVER_ID to scope.serverId,
            TaskRefreshWorker.INPUT_BASE_URL to scope.baseUrl,
            TaskRefreshWorker.INPUT_ACCOUNT_ID to scope.accountId,
        )
        val constraints = Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build()
        val manager = WorkManager.getInstance(context.applicationContext)
        val periodic = PeriodicWorkRequestBuilder<TaskRefreshWorker>(6, TimeUnit.HOURS)
            .setInputData(input)
            .setConstraints(constraints)
            .build()
        manager.enqueueUniquePeriodicWork(
            periodicWorkName(scope),
            ExistingPeriodicWorkPolicy.UPDATE,
            periodic,
        )
        if (refreshImmediately) {
            val immediate = OneTimeWorkRequestBuilder<TaskRefreshWorker>()
                .setInputData(input)
                .setConstraints(constraints)
                .build()
            manager.enqueueUniqueWork(immediateWorkName(scope), ExistingWorkPolicy.REPLACE, immediate)
        }
    }

    private fun periodicWorkName(scope: WidgetScope) = periodicWorkName(scope.storageId)
    private fun periodicWorkName(storageId: String) = "my-tasks-v1:$storageId:periodic"
    private fun immediateWorkName(scope: WidgetScope) = immediateWorkName(scope.storageId)
    private fun immediateWorkName(storageId: String) = "my-tasks-v1:$storageId:immediate"
}

class BackgroundRescheduleReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action == Intent.ACTION_BOOT_COMPLETED || intent.action == Intent.ACTION_MY_PACKAGE_REPLACED) {
            AndroidBackgroundScheduler.reschedulePersisted(context.applicationContext)
        }
    }
}
