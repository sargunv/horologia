package dev.horologia.mobile.core.session

import kotlinx.cinterop.BetaInteropApi
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.cinterop.alloc
import kotlinx.cinterop.memScoped
import kotlinx.cinterop.ptr
import kotlinx.cinterop.value
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import platform.CoreFoundation.CFDictionaryRef
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
@OptIn(ExperimentalForeignApi::class, BetaInteropApi::class)
@Suppress("UNCHECKED_CAST", "CAST_NEVER_SUCCEEDS")
actual class SessionStore {
  actual suspend fun read(host: String): StoredSession? {
    val raw = copyData(host = host) ?: return null
    return try {
      json.decodeFromString<StoredSession>(nsDataToString(raw))
    } catch (_: Throwable) {
      null
    }
  }

  actual suspend fun write(host: String, session: StoredSession) {
    val payload = stringToNSData(json.encodeToString(session))
    if (copyData(host = host) != null) {
      val query = NSMutableDictionary()
      query.setObject(kSecClassGenericPassword!!, forKey = kSecClass!! as NSString)
      query.setObject(SERVICE, forKey = kSecAttrService!! as NSString)
      query.setObject(host, forKey = kSecAttrAccount!! as NSString)
      val updates = NSMutableDictionary()
      updates.setObject(payload, forKey = kSecValueData!! as NSString)
      SecItemUpdate(query.asCFDict(), updates.asCFDict())
    } else {
      val add = NSMutableDictionary()
      add.setObject(kSecClassGenericPassword!!, forKey = kSecClass!! as NSString)
      add.setObject(SERVICE, forKey = kSecAttrService!! as NSString)
      add.setObject(host, forKey = kSecAttrAccount!! as NSString)
      add.setObject(kSecAttrAccessibleAfterFirstUnlock!!, forKey = kSecAttrAccessible!! as NSString)
      add.setObject(payload, forKey = kSecValueData!! as NSString)
      SecItemAdd(add.asCFDict(), null)
    }
  }

  actual suspend fun clear(host: String) {
    val query = NSMutableDictionary()
    query.setObject(kSecClassGenericPassword!!, forKey = kSecClass!! as NSString)
    query.setObject(SERVICE, forKey = kSecAttrService!! as NSString)
    query.setObject(host, forKey = kSecAttrAccount!! as NSString)
    SecItemDelete(query.asCFDict())
  }

  private fun copyData(host: String): NSData? {
    memScoped {
      val query = NSMutableDictionary()
      query.setObject(kSecClassGenericPassword!!, forKey = kSecClass!! as NSString)
      query.setObject(SERVICE, forKey = kSecAttrService!! as NSString)
      query.setObject(host, forKey = kSecAttrAccount!! as NSString)
      query.setObject(kSecMatchLimitOne!!, forKey = kSecMatchLimit!! as NSString)
      query.setObject(NSNumber(bool = true), forKey = kSecReturnData!! as NSString)
      val dataOut = alloc<CFTypeRefVar>()
      val status = SecItemCopyMatching(query.asCFDict(), dataOut.ptr)
      if (status == errSecItemNotFound) return null
      if (status != errSecSuccess) return null
      val ptr = dataOut.value ?: return null
      @Suppress("UNCHECKED_CAST")
      return ptr.asNSData()
    }
  }

  private fun NSDictionary.asCFDict(): CFDictionaryRef? {
    // NSDictionary <-> CFDictionaryRef is toll-free bridged; Kotlin/Native
    // surfaces the dual type — an unchecked cast crosses the boundary.
    @Suppress("UNCHECKED_CAST")
    return this as CFDictionaryRef
  }

  private fun kotlinx.cinterop.CPointer<*>.asNSData(): NSData? {
    @Suppress("UNCHECKED_CAST")
    return objcObjectAsNSData(this)
  }

  private fun stringToNSData(raw: String): NSData {
    val nsString = NSString.create(string = raw)
    return nsString.dataUsingEncoding(NSUTF8StringEncoding)
      ?: error("Failed to UTF-8 encode session JSON")
  }

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
