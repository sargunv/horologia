/* 
 * NOTE: This file is auto generated. Do not edit the file manually!
 * 
 * Tend API
 * Version 0.0.0
 * 
 * Generated reproducibly; timestamp omitted.
 * OpenAPI KMP Gen (version 1.3.0) by kroegerama
 */
@file:Suppress("ArrayInDataClass", "RedundantVisibilityModifier", "unused", "ConstPropertyName")

package dev.tend.mobile.generated

import com.kroegerama.openapi.kmp.gen.`companion`.ApiHolder
import io.ktor.http.Url
import kotlin.String
import kotlin.Suppress
import kotlin.collections.List
import kotlin.collections.listOf

public object Api : ApiHolder() {
  public const val title: String = "Tend API"

  public const val description: String = ""

  public const val version: String = "0.0.0"

  public const val createdAt: String = "1970-01-01T00:00:00Z"

  public val servers: List<Url> = listOf(
    Url("http:///"),
  )

  override var baseUrl: Url = servers.first()

  public fun setAuthProvider(auth: Auth) {
    setAuthProvider(auth.key, auth::provideAuthItem)
  }

  public fun clearAuthProvider(auth: Auth) {
    clearAuthProvider(auth.key)
  }
}
