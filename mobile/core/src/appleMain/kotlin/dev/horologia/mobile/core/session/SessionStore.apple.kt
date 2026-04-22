package dev.horologia.mobile.core.session

import kotlinx.cinterop.BetaInteropApi
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.cinterop.addressOf
import kotlinx.cinterop.alloc
import kotlinx.cinterop.autoreleasepool
import kotlinx.cinterop.memScoped
import kotlinx.cinterop.ptr
import kotlinx.cinterop.readBytes
import kotlinx.cinterop.reinterpret
import kotlinx.cinterop.usePinned
import kotlinx.cinterop.value
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import platform.CoreFoundation.CFDataCreate
import platform.CoreFoundation.CFDataGetBytePtr
import platform.CoreFoundation.CFDataGetLength
import platform.CoreFoundation.CFDataRef
import platform.CoreFoundation.CFDictionaryAddValue
import platform.CoreFoundation.CFDictionaryCreateMutable
import platform.CoreFoundation.CFMutableDictionaryRef
import platform.CoreFoundation.CFRelease
import platform.CoreFoundation.CFStringCreateWithCString
import platform.CoreFoundation.CFStringRef
import platform.CoreFoundation.CFTypeRefVar
import platform.CoreFoundation.kCFBooleanTrue
import platform.CoreFoundation.kCFStringEncodingUTF8
import platform.CoreFoundation.kCFTypeDictionaryKeyCallBacks
import platform.CoreFoundation.kCFTypeDictionaryValueCallBacks
import platform.Security.SecItemAdd
import platform.Security.SecItemCopyMatching
import platform.Security.SecItemDelete
import platform.Security.SecItemUpdate
import platform.Security.errSecDuplicateItem
import platform.Security.errSecItemNotFound
import platform.Security.errSecSuccess
import platform.Security.kSecAttrAccessible
import platform.Security.kSecAttrAccessibleAfterFirstUnlock
import platform.Security.kSecAttrAccount
import platform.Security.kSecAttrService
import platform.Security.kSecClass
import platform.Security.kSecClassGenericPassword
import platform.Security.kSecMatchLimit
import platform.Security.kSecMatchLimitOne
import platform.Security.kSecReturnData
import platform.Security.kSecValueData

private val json: Json = Json { ignoreUnknownKeys = true }

/**
 * iOS Keychain session store. One item per host under service `dev.horologia.mobile.session`.
 * Accessibility is `kSecAttrAccessibleAfterFirstUnlock` so tokens survive reboot after first unlock
 * without being available on a locked device.
 *
 * Works entirely in Core Foundation types — user-supplied strings/data go through
 * `CFStringCreateWithCString` / `CFDataCreate`, and the `kSec*` constants are already
 * `CFStringRef`. This avoids the toll-free-bridge trap where `kSecClass as NSString` compiles but
 * throws `CPointer cannot be cast to NSString` at runtime, because Kotlin/Native does **not**
 * implicitly bridge CF ↔ NS — an `as` cast between them is a lie the runtime catches.
 */
internal class SessionStoreException(message: String) : RuntimeException(message)

@OptIn(ExperimentalForeignApi::class, BetaInteropApi::class)
class IosSessionStore : SessionStore {
  override suspend fun read(host: String): StoredSession? = autoreleasepool {
    val bytes = copyDataBytes(host = host) ?: return@autoreleasepool null
    try {
      json.decodeFromString<StoredSession>(bytes.decodeToString())
    } catch (_: Throwable) {
      null
    }
  }

  override suspend fun write(host: String, session: StoredSession) {
    val payload = json.encodeToString(session).encodeToByteArray()
    autoreleasepool {
      withCFString(SERVICE) { serviceCF ->
        withCFString(host) { accountCF ->
          withCFData(payload) { payloadCF ->
            val added = withCFMutableDict { add ->
              CFDictionaryAddValue(add, kSecClass, kSecClassGenericPassword)
              CFDictionaryAddValue(add, kSecAttrService, serviceCF)
              CFDictionaryAddValue(add, kSecAttrAccount, accountCF)
              CFDictionaryAddValue(add, kSecAttrAccessible, kSecAttrAccessibleAfterFirstUnlock)
              CFDictionaryAddValue(add, kSecValueData, payloadCF)
              val status = SecItemAdd(add, null)
              when (status) {
                errSecSuccess -> true
                errSecDuplicateItem -> false
                else ->
                  throw SessionStoreException("Keychain save failed (SecItemAdd OSStatus=$status).")
              }
            }
            if (added) return@withCFData
            // Duplicate — update in place. Nested scopes guarantee CFRelease order matches
            // creation order even if the inner body throws.
            withCFMutableDict { query ->
              CFDictionaryAddValue(query, kSecClass, kSecClassGenericPassword)
              CFDictionaryAddValue(query, kSecAttrService, serviceCF)
              CFDictionaryAddValue(query, kSecAttrAccount, accountCF)
              withCFMutableDict { updates ->
                CFDictionaryAddValue(updates, kSecValueData, payloadCF)
                val status = SecItemUpdate(query, updates)
                if (status != errSecSuccess) {
                  throw SessionStoreException(
                    "Keychain save failed (SecItemUpdate OSStatus=$status)."
                  )
                }
              }
            }
          }
        }
      }
    }
  }

  override suspend fun clear(host: String) {
    autoreleasepool {
      withCFString(SERVICE) { serviceCF ->
        withCFString(host) { accountCF ->
          withCFMutableDict { query ->
            CFDictionaryAddValue(query, kSecClass, kSecClassGenericPassword)
            CFDictionaryAddValue(query, kSecAttrService, serviceCF)
            CFDictionaryAddValue(query, kSecAttrAccount, accountCF)
            val status = SecItemDelete(query)
            if (status != errSecSuccess && status != errSecItemNotFound) {
              throw SessionStoreException(
                "Keychain delete failed (SecItemDelete OSStatus=$status)."
              )
            }
          }
        }
      }
    }
  }

  private fun copyDataBytes(host: String): ByteArray? = autoreleasepool {
    withCFString(SERVICE) { serviceCF ->
      withCFString(host) { accountCF ->
        withCFMutableDict { query ->
          CFDictionaryAddValue(query, kSecClass, kSecClassGenericPassword)
          CFDictionaryAddValue(query, kSecAttrService, serviceCF)
          CFDictionaryAddValue(query, kSecAttrAccount, accountCF)
          CFDictionaryAddValue(query, kSecMatchLimit, kSecMatchLimitOne)
          CFDictionaryAddValue(query, kSecReturnData, kCFBooleanTrue)
          val dataRef = secItemCopyDataRef(query) ?: return@withCFMutableDict null
          try {
            val length = CFDataGetLength(dataRef).toInt()
            if (length == 0) ByteArray(0)
            else CFDataGetBytePtr(dataRef)?.readBytes(length) ?: ByteArray(0)
          } finally {
            CFRelease(dataRef)
          }
        }
      }
    }
  }

  /**
   * Runs `SecItemCopyMatching` and returns the +1-retained `CFDataRef` on success. Distinguishes
   * "no stored session" (`errSecItemNotFound` → null) from any other failure — the latter throws so
   * a silently broken keychain doesn't present as "user is signed out."
   */
  private fun secItemCopyDataRef(query: CFMutableDictionaryRef): CFDataRef? = memScoped {
    val out = alloc<CFTypeRefVar>()
    val status = SecItemCopyMatching(query, out.ptr)
    when (status) {
      errSecSuccess -> out.value?.reinterpret()
      errSecItemNotFound -> null
      else ->
        throw SessionStoreException("Keychain read failed (SecItemCopyMatching OSStatus=$status).")
    }
  }

  /**
   * Build a `CFMutableDictionaryRef` with standard CF type key/value callbacks, run [block], and
   * release the dict. Nested `withCFMutableDict` calls preserve LIFO release ordering even when the
   * inner body throws.
   */
  private inline fun <T> withCFMutableDict(block: (CFMutableDictionaryRef) -> T): T {
    val dict =
      CFDictionaryCreateMutable(
        null,
        0,
        kCFTypeDictionaryKeyCallBacks.ptr,
        kCFTypeDictionaryValueCallBacks.ptr,
      ) ?: error("CFDictionaryCreateMutable returned null")
    try {
      return block(dict)
    } finally {
      CFRelease(dict)
    }
  }

  /**
   * Create a `CFStringRef` from a Kotlin UTF-8 string, invoke [block], then release it. Scoped so
   * the caller can't accidentally leak the +1 retain.
   */
  private inline fun <T> withCFString(value: String, block: (CFStringRef) -> T): T {
    val cf =
      CFStringCreateWithCString(null, value, kCFStringEncodingUTF8)
        ?: error("CFStringCreateWithCString returned null for: $value")
    try {
      return block(cf)
    } finally {
      CFRelease(cf)
    }
  }

  /** Create a `CFDataRef` from a Kotlin [ByteArray], invoke [block], then release it. */
  private inline fun <T> withCFData(bytes: ByteArray, block: (CFDataRef) -> T): T {
    // `Pinned<ByteArray>.addressOf(0)` throws IOOB on empty arrays, so pin-and-read only when
    // there's at least one byte; pass `null,0` to CFDataCreate otherwise.
    val cf: CFDataRef =
      if (bytes.isEmpty()) {
        CFDataCreate(null, null, 0) ?: error("CFDataCreate returned null (empty)")
      } else {
        bytes.usePinned { pinned ->
          CFDataCreate(null, pinned.addressOf(0).reinterpret(), bytes.size.toLong())
            ?: error("CFDataCreate returned null (${bytes.size} bytes)")
        }
      }
    try {
      return block(cf)
    } finally {
      CFRelease(cf)
    }
  }

  private companion object {
    const val SERVICE = "dev.horologia.mobile.session"
  }
}
