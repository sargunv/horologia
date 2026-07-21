@file:OptIn(kotlinx.cinterop.ExperimentalForeignApi::class)

package dev.horologia.mobile.auth

import kotlinx.cinterop.ByteVar
import kotlinx.cinterop.addressOf
import kotlinx.cinterop.alloc
import kotlinx.cinterop.get
import kotlinx.cinterop.interpretObjCPointer
import kotlinx.cinterop.memScoped
import kotlinx.cinterop.ptr
import kotlinx.cinterop.rawValue
import kotlinx.cinterop.readBytes
import kotlinx.cinterop.reinterpret
import kotlinx.cinterop.usePinned
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import platform.CoreFoundation.CFDataCreate
import platform.CoreFoundation.CFDataRef
import platform.CoreFoundation.CFDictionaryAddValue
import platform.CoreFoundation.CFDictionaryCreateMutable
import platform.CoreFoundation.CFMutableDictionaryRef
import platform.CoreFoundation.CFRelease
import platform.CoreFoundation.CFTypeRefVar
import platform.Foundation.CFBridgingRelease
import platform.Foundation.CFBridgingRetain
import platform.Foundation.NSData
import platform.Security.SecItemAdd
import platform.Security.SecItemCopyMatching
import platform.Security.SecItemDelete
import platform.Security.errSecItemNotFound
import platform.Security.errSecSuccess
import platform.Security.kSecAttrAccessible
import platform.Security.kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
import platform.Security.kSecAttrAccount
import platform.Security.kSecAttrService
import platform.Security.kSecClass
import platform.Security.kSecClassGenericPassword
import platform.Security.kSecMatchLimit
import platform.Security.kSecMatchLimitOne
import platform.Security.kSecReturnData
import platform.Security.kSecValueData

class IosCredentialStore : CredentialStore {
    private val mutex = Mutex()
    private val json = Json { ignoreUnknownKeys = true }

    override suspend fun save(serverId: String, credentials: CredentialBundle) = mutex.withLock {
        put(CREDENTIAL_SERVICE_PREFIX + serverId, credentials.accountId, json.encodeToString(credentials))
    }

    override suspend fun load(serverId: String, accountId: String): CredentialBundle? = mutex.withLock {
        get(CREDENTIAL_SERVICE_PREFIX + serverId, accountId)?.let { json.decodeFromString<CredentialBundle>(it) }
    }

    override suspend fun delete(serverId: String, accountId: String) = mutex.withLock {
        remove(CREDENTIAL_SERVICE_PREFIX + serverId, accountId)
    }

    override suspend fun setActiveAccount(serverId: String, accountId: String?) = mutex.withLock {
        if (accountId == null) remove(ACTIVE_SERVICE, serverId) else put(ACTIVE_SERVICE, serverId, accountId)
    }

    override suspend fun getActiveAccount(serverId: String): String? = mutex.withLock {
        get(ACTIVE_SERVICE, serverId)
    }

    private fun put(service: String, account: String, value: String) {
        remove(service, account)
        val data = value.encodeToByteArray().toCFData()
        try {
            withKeychainQuery(service, account, capacity = 5) { query ->
                CFDictionaryAddValue(query, kSecAttrAccessible, kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly)
                CFDictionaryAddValue(query, kSecValueData, data)
                val status = SecItemAdd(query, null)
                check(status == errSecSuccess) { "Keychain write failed (OSStatus $status)" }
            }
        } finally {
            CFRelease(data)
        }
    }

    private fun get(service: String, account: String): String? = memScoped {
        val result = alloc<CFTypeRefVar>()
        withKeychainQuery(service, account, capacity = 5) { query ->
            CFDictionaryAddValue(query, kSecReturnData, platform.CoreFoundation.kCFBooleanTrue)
            CFDictionaryAddValue(query, kSecMatchLimit, kSecMatchLimitOne)
            val status = SecItemCopyMatching(query, result.ptr)
            if (status == errSecItemNotFound) return@memScoped null
            check(status == errSecSuccess) { "Keychain read failed (OSStatus $status)" }
            val returnedData = result.ptr[0]
                ?: error("Keychain returned invalid credential data")
            try {
                val data = returnedData.let { pointer -> interpretObjCPointer<NSData>(pointer.rawValue) }
                data.toByteArray().decodeToString()
            } finally {
                CFRelease(returnedData)
            }
        }
    }

    private fun remove(service: String, account: String) {
        withKeychainQuery(service, account, capacity = 3) { query ->
            val status = SecItemDelete(query)
            check(status == errSecSuccess || status == errSecItemNotFound) {
                "Keychain delete failed (OSStatus $status)"
            }
        }
    }

    private fun ByteArray.toCFData(): CFDataRef {
        return if (isEmpty()) {
            CFDataCreate(null, null, 0)
        } else {
            usePinned { pinned ->
                CFDataCreate(null, pinned.addressOf(0).reinterpret(), size.toLong())
            }
        } ?: error("Could not create credential data")
    }

    private fun NSData.toByteArray(): ByteArray {
        if (length == 0UL) return ByteArray(0)
        check(length <= Int.MAX_VALUE.toULong()) { "Credential data is too large" }
        val source = bytes ?: error("Credential data has no byte buffer")
        return source.reinterpret<ByteVar>().readBytes(length.toInt())
    }

    private inline fun <T> withKeychainQuery(
        service: String,
        account: String,
        capacity: Int,
        block: (CFMutableDictionaryRef) -> T,
    ): T {
        val query = CFDictionaryCreateMutable(null, capacity.toLong(), null, null)
            ?: error("Could not create Keychain query")
        try {
            val retainedService = CFBridgingRetain(service)
                ?: error("Could not bridge Keychain service")
            try {
                val retainedAccount = CFBridgingRetain(account)
                    ?: error("Could not bridge Keychain account")
                try {
                    CFDictionaryAddValue(query, kSecClass, kSecClassGenericPassword)
                    CFDictionaryAddValue(query, kSecAttrService, retainedService)
                    CFDictionaryAddValue(query, kSecAttrAccount, retainedAccount)
                    return block(query)
                } finally {
                    CFBridgingRelease(retainedAccount)
                }
            } finally {
                CFBridgingRelease(retainedService)
            }
        } finally {
            CFRelease(query)
        }
    }

    private companion object {
        const val CREDENTIAL_SERVICE_PREFIX = "dev.horologia.mobile.credentials."
        const val ACTIVE_SERVICE = "dev.horologia.mobile.active-account"
    }
}
