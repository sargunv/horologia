import type { CredentialKey, OAuthCredentials } from "@horologia/client-core";
import * as SecureStore from "expo-secure-store";

function storageKey(key: CredentialKey): string {
  return `oauth.${key.serverId}.${key.accountId}`;
}

export async function getCredentials(key: CredentialKey): Promise<OAuthCredentials | null> {
  const serialized = await SecureStore.getItemAsync(storageKey(key));
  if (!serialized) return null;
  const parsed: unknown = JSON.parse(serialized);
  if (!isCredentials(parsed)) {
    await SecureStore.deleteItemAsync(storageKey(key));
    return null;
  }
  return parsed;
}

export async function setCredentials(
  key: CredentialKey,
  credentials: OAuthCredentials,
): Promise<void> {
  await SecureStore.setItemAsync(storageKey(key), JSON.stringify(credentials), {
    keychainAccessible: SecureStore.WHEN_UNLOCKED_THIS_DEVICE_ONLY,
  });
}

export async function deleteCredentials(key: CredentialKey): Promise<void> {
  await SecureStore.deleteItemAsync(storageKey(key));
}

export async function setCredentialsWhileCurrent(
  key: CredentialKey,
  credentials: OAuthCredentials,
  isCurrent: () => boolean,
): Promise<boolean> {
  if (!isCurrent()) return false;
  await setCredentials(key, credentials);
  if (isCurrent()) return true;
  await deleteCredentials(key);
  return false;
}

function isCredentials(value: unknown): value is OAuthCredentials {
  return (
    typeof value === "object" &&
    value !== null &&
    "accessToken" in value &&
    typeof value.accessToken === "string" &&
    "refreshToken" in value &&
    typeof value.refreshToken === "string" &&
    "expiresAt" in value &&
    typeof value.expiresAt === "string" &&
    "scope" in value &&
    typeof value.scope === "string"
  );
}
