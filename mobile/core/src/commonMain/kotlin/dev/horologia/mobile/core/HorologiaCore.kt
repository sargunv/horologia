package dev.horologia.mobile.core

object HorologiaCore {
  fun createAppContainer(baseUrl: String, getToken: () -> String?): AppContainer =
    AppContainer(baseUrl = baseUrl, getToken = getToken)
}
