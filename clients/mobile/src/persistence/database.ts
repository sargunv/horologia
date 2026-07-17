import { createServerProfile, type ServerProfile } from "@horologia/client-core";
import * as Crypto from "expo-crypto";
import * as SQLite from "expo-sqlite";

export interface ActiveAccount {
  profile: ServerProfile;
  accountId: string | null;
}

let databasePromise: ReturnType<typeof SQLite.openDatabaseAsync> | null = null;

async function database() {
  databasePromise ??= SQLite.openDatabaseAsync("horologia.db");
  const db = await databasePromise;
  await db.execAsync(`
    PRAGMA journal_mode = WAL;
    CREATE TABLE IF NOT EXISTS server_profiles (
      id TEXT PRIMARY KEY NOT NULL,
      base_url TEXT NOT NULL,
      display_name TEXT NOT NULL,
      last_used_at TEXT NOT NULL,
      account_id TEXT,
      active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1))
    );
    CREATE UNIQUE INDEX IF NOT EXISTS one_active_server
      ON server_profiles(active) WHERE active = 1;
  `);
  return db;
}

export async function loadActiveAccount(): Promise<ActiveAccount | null> {
  const db = await database();
  const row = await db.getFirstAsync<{
    id: string;
    base_url: string;
    display_name: string;
    last_used_at: string;
    account_id: string | null;
  }>(
    "SELECT id, base_url, display_name, last_used_at, account_id FROM server_profiles WHERE active = 1",
  );
  if (!row) return null;
  return {
    profile: {
      id: row.id,
      baseUrl: row.base_url,
      displayName: row.display_name,
      lastUsedAt: row.last_used_at,
    },
    accountId: row.account_id,
  };
}

export async function saveActiveServer(baseUrl: string): Promise<ServerProfile> {
  const db = await database();
  const existing = await db.getFirstAsync<{ id: string }>(
    "SELECT id FROM server_profiles WHERE base_url = ? LIMIT 1",
    baseUrl,
  );
  const profile = createServerProfile({
    id: existing?.id ?? Crypto.randomUUID(),
    baseUrl,
    now: new Date().toISOString(),
  });
  await db.withTransactionAsync(async () => {
    await db.runAsync("UPDATE server_profiles SET active = 0 WHERE active = 1");
    await db.runAsync(
      `INSERT INTO server_profiles(id, base_url, display_name, last_used_at, active)
       VALUES (?, ?, ?, ?, 1)
       ON CONFLICT(id) DO UPDATE SET
         base_url = excluded.base_url,
         display_name = excluded.display_name,
         last_used_at = excluded.last_used_at,
         active = 1`,
      profile.id,
      profile.baseUrl,
      profile.displayName,
      profile.lastUsedAt,
    );
  });
  return profile;
}

export async function attachActiveAccount(profileId: string, accountId: string): Promise<void> {
  const db = await database();
  await db.runAsync("UPDATE server_profiles SET account_id = ? WHERE id = ?", accountId, profileId);
}

export async function clearActiveAccount(): Promise<void> {
  const db = await database();
  await db.runAsync("UPDATE server_profiles SET account_id = NULL, active = 0 WHERE active = 1");
}
