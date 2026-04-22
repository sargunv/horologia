package dev.horologia.mobile.core.session

import kotlinx.cinterop.BetaInteropApi
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.cinterop.alloc
import kotlinx.cinterop.autoreleasepool
import kotlinx.cinterop.memScoped
import kotlinx.cinterop.ptr
import kotlinx.cinterop.value
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import platform.CoreFoundation.CFDictionaryRef
import platform.CoreFoundation.CFRelease
import platform.CoreFoundation.CFTypeRefVar
import platform.Foundation.NSData
import platform.Foundation.NSDictionary
import platform.Foundation.NSMutableDictionary
import platform.Foundation.NSNumber
import platform.Foundation.NSString
import platform.Foundation.NSUTF8StringEncoding
import platform.Foundation.create
import platform.Foundation.dataUsingEncoding
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

/**
 * iOS Keychain session store, one item per host under service `dev.horologia.mobile.session`.
 * Accessibility is `kSecAttrAccessibleAfterFirstUnlock` so the token survives reboot + first unlock
 * without leaking to a locked device.
 */
internal class SessionStoreException(message: String) : RuntimeException(message)

@OptIn(ExperimentalForeignApi::class, BetaInteropApi::class)
class IosSessionStore : SessionStore {
  override suspend fun read(host: String): StoredSession? = autoreleasepool {
    val raw = copyData(host = host) ?: return@autoreleasepool null
    try {
      json.decodeFromString<StoredSession>(nsDataToString(raw))
    } catch (_: Throwable) {
      null
    }
  }

  override suspend fun write(host: String, session: StoredSession) {
    autoreleasepool {
      val payload = stringToNSData(json.encodeToString(session))
      // Try an add first; on duplicate, fall back to update.
      val add = NSMutableDictionary()
      @Suppress("CAST_NEVER_SUCCEEDS")
      add.setObject(kSecClassGenericPassword!!, forKey = kSecClass!! as NSString)
      @Suppress("CAST_NEVER_SUCCEEDS")
      add.setObject(SERVICE, forKey = kSecAttrService!! as NSString)
      @Suppress("CAST_NEVER_SUCCEEDS") add.setObject(host, forKey = kSecAttrAccount!! as NSString)
      @Suppress("CAST_NEVER_SUCCEEDS")
      add.setObject(kSecAttrAccessibleAfterFirstUnlock!!, forKey = kSecAttrAccessible!! as NSString)
      @Suppress("CAST_NEVER_SUCCEEDS") add.setObject(payload, forKey = kSecValueData!! as NSString)
      val addStatus = SecItemAdd(add.asCFDict(), null)
      if (addStatus == errSecSuccess) return@autoreleasepool
      if (addStatus == errSecDuplicateItem) {
        val query = NSMutableDictionary()
        @Suppress("CAST_NEVER_SUCCEEDS")
        query.setObject(kSecClassGenericPassword!!, forKey = kSecClass!! as NSString)
        @Suppress("CAST_NEVER_SUCCEEDS")
        query.setObject(SERVICE, forKey = kSecAttrService!! as NSString)
        @Suppress("CAST_NEVER_SUCCEEDS")
        query.setObject(host, forKey = kSecAttrAccount!! as NSString)
        val updates = NSMutableDictionary()
        @Suppress("CAST_NEVER_SUCCEEDS")
        updates.setObject(payload, forKey = kSecValueData!! as NSString)
        val updateStatus = SecItemUpdate(query.asCFDict(), updates.asCFDict())
        if (updateStatus == errSecSuccess) return@autoreleasepool
        throw SessionStoreException("Couldn't save your session to the keychain.")
      }
      throw SessionStoreException("Couldn't save your session to the keychain.")
    }
  }

  override suspend fun clear(host: String) {
    autoreleasepool {
      val query = NSMutableDictionary()
      @Suppress("CAST_NEVER_SUCCEEDS")
      query.setObject(kSecClassGenericPassword!!, forKey = kSecClass!! as NSString)
      @Suppress("CAST_NEVER_SUCCEEDS")
      query.setObject(SERVICE, forKey = kSecAttrService!! as NSString)
      @Suppress("CAST_NEVER_SUCCEEDS") query.setObject(host, forKey = kSecAttrAccount!! as NSString)
      val status = SecItemDelete(query.asCFDict())
      if (status == errSecSuccess || status == errSecItemNotFound) return@autoreleasepool
      throw SessionStoreException("Couldn't clear the keychain item.")
    }
  }

  private fun copyData(host: String): NSData? = autoreleasepool {
    memScoped {
      val query = NSMutableDictionary()
      @Suppress("CAST_NEVER_SUCCEEDS")
      query.setObject(kSecClassGenericPassword!!, forKey = kSecClass!! as NSString)
      @Suppress("CAST_NEVER_SUCCEEDS")
      query.setObject(SERVICE, forKey = kSecAttrService!! as NSString)
      @Suppress("CAST_NEVER_SUCCEEDS") query.setObject(host, forKey = kSecAttrAccount!! as NSString)
      @Suppress("CAST_NEVER_SUCCEEDS")
      query.setObject(kSecMatchLimitOne!!, forKey = kSecMatchLimit!! as NSString)
      @Suppress("CAST_NEVER_SUCCEEDS")
      query.setObject(NSNumber(bool = true), forKey = kSecReturnData!! as NSString)
      val dataOut = alloc<CFTypeRefVar>()
      val status = SecItemCopyMatching(query.asCFDict(), dataOut.ptr)
      if (status == errSecItemNotFound) return@memScoped null
      if (status != errSecSuccess) return@memScoped null
      val raw = dataOut.value ?: return@memScoped null
      // NSData <-> CFDataRef are toll-free bridged. We retain the Obj-C / Kotlin
      // reference via objcObjectAsNSData, then CFRelease the +1 retain we got
      // from SecItemCopyMatching so it doesn't leak.
      val nsData: NSData? = objcObjectAsNSData(raw)
      CFRelease(raw)
      nsData
    }
  }

  @Suppress("UNCHECKED_CAST")
  private fun NSDictionary.asCFDict(): CFDictionaryRef? = this as CFDictionaryRef

  private fun stringToNSData(raw: String): NSData {
    val nsString = NSString.create(string = raw)
    return nsString.dataUsingEncoding(NSUTF8StringEncoding)
      ?: error("Failed to UTF-8 encode session JSON")
  }

  @Suppress("CAST_NEVER_SUCCEEDS")
  private fun nsDataToString(data: NSData): String =
    (NSString.create(data = data, encoding = NSUTF8StringEncoding) as? String)
      ?: error("Failed to UTF-8 decode keychain payload")

  private companion object {
    const val SERVICE = "dev.horologia.mobile.session"
    val json: Json = Json { ignoreUnknownKeys = true }
  }
}

@OptIn(ExperimentalForeignApi::class, BetaInteropApi::class)
private fun objcObjectAsNSData(ptr: kotlinx.cinterop.CPointer<*>): NSData? =
  kotlinx.cinterop.interpretObjCPointerOrNull<NSData>(ptr.rawValue)
