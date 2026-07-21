package dev.horologia.mobile.widgets

import android.content.Context
import android.net.Uri
import dev.horologia.mobile.widgets.TaskWidgetSnapshotJson
import dev.horologia.mobile.widgets.TaskWidgetSnapshotV1
import java.security.MessageDigest

internal data class WidgetScope(
    val serverId: String,
    val baseUrl: String,
    val accountId: String,
) {
    init {
        require(serverId.isNotBlank())
        require(baseUrl.isNotBlank())
        require(accountId.isNotBlank())
    }

    val storageId: String
        get() = idFor(serverId, accountId)

    fun encode(): String = listOf(serverId, baseUrl, accountId).joinToString("|") { Uri.encode(it) }

    companion object {
        fun idFor(serverId: String, accountId: String): String {
            require(serverId.isNotBlank() && accountId.isNotBlank())
            return MessageDigest.getInstance("SHA-256")
                .digest("$serverId\u0000$accountId".toByteArray())
                .joinToString("") { "%02x".format(it) }
        }

        fun decode(value: String): WidgetScope? {
            val fields = value.split('|')
            if (fields.size != 3) return null
            return runCatching { WidgetScope(Uri.decode(fields[0]), Uri.decode(fields[1]), Uri.decode(fields[2])) }.getOrNull()
        }
    }
}

internal data class WidgetContent(
    val scope: WidgetScope?,
    val snapshot: TaskWidgetSnapshotV1?,
    val error: String?,
)

internal object TaskWidgetStore {
    private const val PREFERENCES = "horologia_widget_v1"
    private const val ACTIVE_SCOPE = "active_scope"
    private const val REGISTRATIONS = "registrations"

    private fun preferences(context: Context) =
        context.applicationContext.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)

    fun register(context: Context, scope: WidgetScope) {
        val preferences = preferences(context)
        val encoded = scope.encode()
        val registrations = preferences.getStringSet(REGISTRATIONS, emptySet()).orEmpty().toMutableSet()
        registrations.removeAll { WidgetScope.decode(it)?.let { old -> old.serverId == scope.serverId && old.accountId == scope.accountId } != false }
        registrations += encoded
        check(preferences.edit().putStringSet(REGISTRATIONS, registrations).putString(ACTIVE_SCOPE, encoded).commit()) {
            "Could not persist widget registration"
        }
    }
    fun unregister(context: Context, serverId: String, accountId: String) {
        val preferences = preferences(context)
        val current = preferences.getStringSet(REGISTRATIONS, emptySet()).orEmpty()
        val removed = current.firstOrNull {
            WidgetScope.decode(it)?.let { scope -> scope.serverId == serverId && scope.accountId == accountId } == true
        } ?: return
        val remaining = current - removed
        val editor = preferences.edit().putStringSet(REGISTRATIONS, remaining)
        if (preferences.getString(ACTIVE_SCOPE, null) == removed) {
            remaining.firstOrNull()?.let { editor.putString(ACTIVE_SCOPE, it) } ?: editor.remove(ACTIVE_SCOPE)
        }
        val storageId = WidgetScope.idFor(serverId, accountId)
        check(editor
            .remove("snapshot:$storageId")
            .remove("error:$storageId")
            .remove("notified:$storageId")
            .commit()) { "Could not remove widget registration" }
    }


    fun registrations(context: Context): List<WidgetScope> =
        preferences(context).getStringSet(REGISTRATIONS, emptySet()).orEmpty().mapNotNull(WidgetScope::decode)

    fun activeScope(context: Context): WidgetScope? =
        preferences(context).getString(ACTIVE_SCOPE, null)?.let(WidgetScope::decode)

    fun writeSnapshot(context: Context, scope: WidgetScope, snapshot: TaskWidgetSnapshotV1) {
        require(snapshot.serverId == scope.serverId && snapshot.accountId == scope.accountId)
        check(preferences(context).edit()
            .putString(snapshotKey(scope), TaskWidgetSnapshotJson.encode(snapshot))
            .remove(errorKey(scope))
            .commit()) { "Could not persist widget snapshot" }
    }

    fun writeError(context: Context, scope: WidgetScope, message: String) {
        check(preferences(context).edit().putString(errorKey(scope), message.take(160)).commit()) {
            "Could not persist widget error"
        }
    }

    fun read(context: Context): WidgetContent {
        val scope = activeScope(context) ?: return WidgetContent(null, null, null)
        val preferences = preferences(context)
        val snapshot = preferences.getString(snapshotKey(scope), null)?.let { encoded ->
            runCatching { TaskWidgetSnapshotJson.decode(encoded) }
                .getOrNull()
                ?.takeIf { it.serverId == scope.serverId && it.accountId == scope.accountId }
        }
        return WidgetContent(scope, snapshot, preferences.getString(errorKey(scope), null))
    }

    fun notifiedTaskIds(context: Context, scope: WidgetScope): Set<String> =
        preferences(context).getStringSet(notifiedKey(scope), emptySet()).orEmpty()

    fun setNotifiedTaskIds(context: Context, scope: WidgetScope, taskIds: Set<String>) {
        preferences(context).edit().putStringSet(notifiedKey(scope), taskIds).apply()
    }

    private fun snapshotKey(scope: WidgetScope) = "snapshot:${scope.storageId}"
    private fun errorKey(scope: WidgetScope) = "error:${scope.storageId}"
    private fun notifiedKey(scope: WidgetScope) = "notified:${scope.storageId}"
}
