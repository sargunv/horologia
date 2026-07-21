package dev.horologia.mobile.persistence

import android.content.Context
import app.cash.sqldelight.db.SqlDriver
import app.cash.sqldelight.driver.android.AndroidSqliteDriver

actual class DatabaseDriverFactory(
    private val context: Context,
) {
    actual fun createDriver(): SqlDriver =
        AndroidSqliteDriver(HorologiaDatabase.Schema, context, DATABASE_NAME)

    private companion object {
        const val DATABASE_NAME = "horologia-cache.db"
    }
}
