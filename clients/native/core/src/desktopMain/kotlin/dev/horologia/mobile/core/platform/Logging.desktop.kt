package dev.horologia.mobile.core.platform

actual fun platformLog(tag: String, message: String) {
  println("[$tag] $message")
}
