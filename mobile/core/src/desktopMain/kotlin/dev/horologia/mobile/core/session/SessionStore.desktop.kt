package dev.horologia.mobile.core.session

import java.io.File
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardOpenOption
import java.nio.file.attribute.PosixFilePermission
import java.nio.file.attribute.PosixFilePermissions
import java.security.SecureRandom
import javax.crypto.Cipher
import javax.crypto.spec.GCMParameterSpec
import javax.crypto.spec.SecretKeySpec
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/**
 * Desktop session store: AES-GCM-encrypted JSON file at the platform user-data dir with a sibling
 * random-key file (mode 0600 on POSIX). No real OS keychain integration — R12's fallback clause
 * allows this. Real macOS Keychain / Windows Credential Manager / libsecret integration is tracked
 * in the workpad "Deferred Items" section.
 *
 * File layout inside `userDataDir()`:
 * - `key.bin` — 32 random bytes, the AES-256 key
 * - `sessions.json` — encrypted JSON Map<host, StoredSession>
 */
actual class SessionStore {
  private val dir: Path = userDataDir().also { Files.createDirectories(it) }
  private val keyFile: Path = dir.resolve("key.bin")
  private val dataFile: Path = dir.resolve("sessions.json")

  actual suspend fun read(host: String): StoredSession? = loadMap()[host]

  actual suspend fun write(host: String, session: StoredSession) {
    val map = loadMap().toMutableMap()
    map[host] = session
    writeMap(map = map)
  }

  actual suspend fun clear(host: String) {
    val map = loadMap().toMutableMap()
    if (map.remove(host) != null) {
      writeMap(map = map)
    }
  }

  private fun loadMap(): Map<String, StoredSession> {
    if (!Files.exists(dataFile)) return emptyMap()
    val blob = Files.readAllBytes(dataFile)
    if (blob.size <= IV_LENGTH) return emptyMap()
    val iv = blob.copyOfRange(fromIndex = 0, toIndex = IV_LENGTH)
    val ciphertext = blob.copyOfRange(fromIndex = IV_LENGTH, toIndex = blob.size)
    val cipher = Cipher.getInstance("AES/GCM/NoPadding")
    cipher.init(
      Cipher.DECRYPT_MODE,
      SecretKeySpec(loadOrCreateKey(), "AES"),
      GCMParameterSpec(TAG_BITS, iv),
    )
    val plaintext = cipher.doFinal(ciphertext)
    return try {
      json.decodeFromString<Map<String, StoredSession>>(plaintext.toString(Charsets.UTF_8))
    } catch (_: Throwable) {
      emptyMap()
    }
  }

  private fun writeMap(map: Map<String, StoredSession>) {
    val iv = ByteArray(IV_LENGTH).also { random.nextBytes(it) }
    val cipher = Cipher.getInstance("AES/GCM/NoPadding")
    cipher.init(
      Cipher.ENCRYPT_MODE,
      SecretKeySpec(loadOrCreateKey(), "AES"),
      GCMParameterSpec(TAG_BITS, iv),
    )
    val plaintext = json.encodeToString(map).toByteArray(Charsets.UTF_8)
    val ciphertext = cipher.doFinal(plaintext)
    val payload = iv + ciphertext
    Files.write(
      dataFile,
      payload,
      StandardOpenOption.CREATE,
      StandardOpenOption.TRUNCATE_EXISTING,
      StandardOpenOption.WRITE,
    )
    tightenPermissions(path = dataFile)
  }

  private fun loadOrCreateKey(): ByteArray {
    if (Files.exists(keyFile)) {
      val existing = Files.readAllBytes(keyFile)
      if (existing.size == KEY_LENGTH) return existing
    }
    val fresh = ByteArray(KEY_LENGTH).also { random.nextBytes(it) }
    Files.write(
      keyFile,
      fresh,
      StandardOpenOption.CREATE,
      StandardOpenOption.TRUNCATE_EXISTING,
      StandardOpenOption.WRITE,
    )
    tightenPermissions(path = keyFile)
    return fresh
  }

  private fun tightenPermissions(path: Path) {
    // Best-effort: POSIX file systems get 0600, Windows is left to its ACL inheritance.
    try {
      val supported = path.fileSystem.supportedFileAttributeViews()
      if ("posix" in supported) {
        Files.setPosixFilePermissions(
          path,
          PosixFilePermissions.fromString("rw-------").toMutableSet().apply {
            removeAll(setOf(PosixFilePermission.GROUP_READ, PosixFilePermission.OTHERS_READ))
          },
        )
      } else {
        val file: File = path.toFile()
        file.setReadable(false, false)
        file.setReadable(true, true)
        file.setWritable(false, false)
        file.setWritable(true, true)
      }
    } catch (_: Throwable) {
      // Best-effort; the encryption still protects data at rest.
    }
  }

  private companion object {
    const val KEY_LENGTH = 32
    const val IV_LENGTH = 12
    const val TAG_BITS = 128
    val random: SecureRandom = SecureRandom()
    val json: Json = Json { ignoreUnknownKeys = true }
  }
}

/**
 * Platform user-data dir:
 * - macOS: `~/Library/Application Support/Horologia`
 * - Linux (respects XDG): `$XDG_DATA_HOME/horologia` or `~/.local/share/horologia`
 * - Windows: `%APPDATA%\horologia`
 */
private fun userDataDir(): Path {
  val os = System.getProperty("os.name").lowercase()
  val home = System.getProperty("user.home")
  return when {
    os.contains("mac") -> Path.of(home, "Library", "Application Support", "Horologia")
    os.contains("win") -> {
      val appData = System.getenv("APPDATA") ?: "$home\\AppData\\Roaming"
      Path.of(appData, "horologia")
    }
    else -> {
      val xdg = System.getenv("XDG_DATA_HOME")
      if (!xdg.isNullOrEmpty()) Path.of(xdg, "horologia")
      else Path.of(home, ".local", "share", "horologia")
    }
  }
}
