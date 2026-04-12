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

import com.kroegerama.openapi.kmp.gen.`companion`.AuthItem
import kotlin.String
import kotlin.Suppress

public sealed interface Auth {
  public val key: String

  public suspend fun provideAuthItem(): AuthItem?

  public data class BearerAuth(
    public val getBearer: suspend () -> AuthItem.Bearer?,
  ) : Auth {
    override val key: String = ID

    override suspend fun provideAuthItem(): AuthItem? = getBearer()

    public companion object {
      public const val ID: String = "BearerAuth"
    }
  }
}
