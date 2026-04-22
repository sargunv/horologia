package dev.horologia.mobile.core.session

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * In-memory fast path for the current session. The authoritative store is [SessionStore] on disk;
 * this holder caches the active tokens so the `getToken` closure inside the generated Api singleton
 * — which is called per outbound request on the Ktor main thread — can read them without hitting
 * disk.
 *
 * `currentAccessToken` is intentionally synchronous: the Ktor `DefaultRequest` block the
 * companion's auth provider hangs off of is not a suspend lambda. `install` is a suspend function
 * because it writes through to [SessionStore].
 *
 * [session] exposes the current value as a StateFlow so UI observers (profile, sign-out trigger)
 * can react if a sibling flow (silent refresh, explicit sign-out) swaps it.
 */
class SessionHolder(private val store: SessionStore) {
  private val _session = MutableStateFlow<ActiveSession?>(null)
  val session: StateFlow<ActiveSession?> = _session.asStateFlow()

  fun currentAccessToken(): String? = _session.value?.session?.accessToken

  suspend fun install(host: String, session: StoredSession) {
    store.write(host = host, session = session)
    _session.value = ActiveSession(host = host, session = session)
  }

  /**
   * Hydrate the in-memory holder from the on-disk store for a given host. Returns the tokens so
   * cold-launch routing can decide what to do next.
   */
  suspend fun load(host: String): StoredSession? {
    val stored = store.read(host = host) ?: return null
    _session.value = ActiveSession(host = host, session = stored)
    return stored
  }

  suspend fun clear() {
    val current = _session.value ?: return
    store.clear(host = current.host)
    _session.value = null
  }
}

data class ActiveSession(val host: String, val session: StoredSession)
