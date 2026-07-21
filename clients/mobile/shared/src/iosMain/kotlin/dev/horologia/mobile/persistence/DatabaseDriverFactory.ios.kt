package dev.horologia.mobile.persistence

import app.cash.sqldelight.db.SqlDriver
import app.cash.sqldelight.driver.native.NativeSqliteDriver

actual class DatabaseDriverFactory {
    actual fun createDriver(): SqlDriver =
        NativeSqliteDriver(HorologiaDatabase.Schema, DATABASE_NAME)

    private companion object {
        const val DATABASE_NAME = "horologia-cache.db"
    }
}
