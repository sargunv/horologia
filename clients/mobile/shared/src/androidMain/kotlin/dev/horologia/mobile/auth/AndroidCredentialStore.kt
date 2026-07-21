package dev.horologia.mobile.auth

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import java.security.KeyStore
import java.security.MessageDigest
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

class AndroidCredentialStore(context: Context) : CredentialStore {
    private val preferences = context.applicationContext.getSharedPreferences(PREFERENCES_NAME, Context.MODE_PRIVATE)
    private val mutex = Mutex()
    private val json = Json { ignoreUnknownKeys = true }

    override suspend fun save(serverId: String, credentials: CredentialBundle) = mutex.withLock {
        val key = credentialKey(serverId, credentials.accountId)
        preferences.edit().putString(key, encrypt(json.encodeToString(credentials), key)).commitOrThrow()
    }

    override suspend fun load(serverId: String, accountId: String): CredentialBundle? = mutex.withLock {
        val key = credentialKey(serverId, accountId)
        preferences.getString(key, null)?.let { json.decodeFromString<CredentialBundle>(decrypt(it, key)) }
    }

    override suspend fun delete(serverId: String, accountId: String) = mutex.withLock {
        preferences.edit().remove(credentialKey(serverId, accountId)).commitOrThrow()
    }

    override suspend fun setActiveAccount(serverId: String, accountId: String?) = mutex.withLock {
        val key = activeKey(serverId)
        val editor = preferences.edit()
        if (accountId == null) editor.remove(key) else editor.putString(key, encrypt(accountId, key))
        editor.commitOrThrow()
    }

    override suspend fun getActiveAccount(serverId: String): String? = mutex.withLock {
        val key = activeKey(serverId)
        preferences.getString(key, null)?.let { decrypt(it, key) }
    }

    private fun encrypt(value: String, associatedData: String): String {
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, secretKey())
        cipher.updateAAD(associatedData.toByteArray(Charsets.UTF_8))
        val ciphertext = cipher.doFinal(value.toByteArray(Charsets.UTF_8))
        return Base64.encodeToString(cipher.iv + ciphertext, Base64.NO_WRAP)
    }

    private fun decrypt(value: String, associatedData: String): String {
        val combined = Base64.decode(value, Base64.NO_WRAP)
        require(combined.size > IV_SIZE) { "Invalid encrypted credential" }
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.DECRYPT_MODE, secretKey(), GCMParameterSpec(TAG_BITS, combined, 0, IV_SIZE))
        cipher.updateAAD(associatedData.toByteArray(Charsets.UTF_8))
        return cipher.doFinal(combined, IV_SIZE, combined.size - IV_SIZE).toString(Charsets.UTF_8)
    }

    private fun secretKey(): SecretKey {
        val keyStore = KeyStore.getInstance(KEYSTORE).apply { load(null) }
        (keyStore.getKey(KEY_ALIAS, null) as? SecretKey)?.let { return it }
        return KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, KEYSTORE).run {
            init(
                KeyGenParameterSpec.Builder(
                    KEY_ALIAS,
                    KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
                ).setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                    .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                    .setKeySize(256)
                    .build(),
            )
            generateKey()
        }
    }

    private fun credentialKey(serverId: String, accountId: String) = "credential:${digest("$serverId\u0000$accountId")}" 
    private fun activeKey(serverId: String) = "active:${digest(serverId)}"

    private fun digest(value: String): String = Base64.encodeToString(
        MessageDigest.getInstance("SHA-256").digest(value.toByteArray(Charsets.UTF_8)),
        Base64.NO_WRAP or Base64.URL_SAFE or Base64.NO_PADDING,
    )

    private fun android.content.SharedPreferences.Editor.commitOrThrow() {
        check(commit()) { "Could not persist encrypted credentials" }
    }

    private companion object {
        const val PREFERENCES_NAME = "horologia_secure_credentials"
        const val KEYSTORE = "AndroidKeyStore"
        const val KEY_ALIAS = "dev.horologia.mobile.credentials.v1"
        const val TRANSFORMATION = "AES/GCM/NoPadding"
        const val IV_SIZE = 12
        const val TAG_BITS = 128
    }
}
