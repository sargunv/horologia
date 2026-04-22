package dev.horologia.mobile.core.platform

import android.util.Log

actual fun platformLog(tag: String, message: String) {
  Log.i(tag, message)
}
