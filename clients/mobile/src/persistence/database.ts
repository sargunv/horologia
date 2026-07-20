import {
  createServerProfile,
  normalizeServerUrl,
  type ServerProfile,
} from "@horologia/client-core";
import * as Crypto from "expo-crypto";
import * as SQLite from "expo-sqlite";

import type { components } from "@horologia/client-core/schema";

type Task = components["schemas"]["Task"];

export interface ActiveAccount {
  profile: ServerProfile;
  accountId: string | null;
}

let databasePromise: Promise<SQLite.SQLiteDatabase> | null = null;

function database(): Promise<SQLite.SQLiteDatabase> {
  databasePromise ??= initializeDatabase();
  return databasePromise;
}

async function initializeDatabase(): Promise<SQLite.SQLiteDatabase> {
  const db = await SQLite.openDatabaseAsync("horologia.db");
  await db.execAsync("PRAGMA journal_mode = WAL");
  const row = await db.getFirstAsync<{ user_version: number }>("PRAGMA user_version");
  const version = row?.user_version ?? 0;
  if (version > 2) throw new Error(`Unsupported Horologia database version ${version}`);

  if (version < 1) {
    await db.withTransactionAsync(() =>
      db.execAsync(`
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
        CREATE TABLE IF NOT EXISTS my_tasks_cache (
          server_id TEXT NOT NULL,
          account_id TEXT NOT NULL,
          updated_at TEXT NOT NULL,
          tasks_json TEXT NOT NULL,
          PRIMARY KEY(server_id, account_id)
        );
        PRAGMA user_version = 1;
      `),
    );
  }
  if (version < 2) {
    await db.withTransactionAsync(() =>
      db.execAsync(`
        ALTER TABLE my_tasks_cache
          ADD COLUMN has_more INTEGER NOT NULL DEFAULT 0 CHECK (has_more IN (0, 1));
        PRAGMA user_version = 2;
      `),
    );
  }
  return db;
}

export interface CachedMyTasks {
  tasks: Task[];
  updatedAt: string;
  hasMore: boolean;
}

export async function loadCachedMyTasks(
  serverId: string,
  accountId: string,
): Promise<CachedMyTasks | null> {
  const db = await database();
  const row = await db.getFirstAsync<{
    tasks_json: string;
    updated_at: string;
    has_more: number;
  }>(
    "SELECT tasks_json, updated_at, has_more FROM my_tasks_cache WHERE server_id = ? AND account_id = ?",
    serverId,
    accountId,
  );
  if (!row) return null;
  try {
    const tasks: unknown = JSON.parse(row.tasks_json);
    if (!Array.isArray(tasks)) return null;
    return {
      tasks: tasks.filter(isCachedTask),
      updatedAt: row.updated_at,
      hasMore: row.has_more === 1,
    };
  } catch {
    return null;
  }
}

export async function saveCachedMyTasks(
  serverId: string,
  accountId: string,
  tasks: Task[],
  updatedAt: string,
  hasMore: boolean,
): Promise<boolean> {
  const db = await database();
  const result = await db.runAsync(
    `INSERT INTO my_tasks_cache(server_id, account_id, updated_at, tasks_json, has_more)
     VALUES (?, ?, ?, ?, ?)
     ON CONFLICT(server_id, account_id) DO UPDATE SET
       updated_at = excluded.updated_at,
       tasks_json = excluded.tasks_json,
       has_more = excluded.has_more
     WHERE excluded.updated_at > my_tasks_cache.updated_at`,
    serverId,
    accountId,
    updatedAt,
    JSON.stringify(tasks),
    hasMore ? 1 : 0,
  );
  return result.changes === 1;
}

export async function clearCachedMyTasks(serverId: string, accountId: string): Promise<void> {
  const db = await database();
  await db.runAsync(
    "DELETE FROM my_tasks_cache WHERE server_id = ? AND account_id = ?",
    serverId,
    accountId,
  );
}

export async function clearCachedMyTasksVersion(
  serverId: string,
  accountId: string,
  updatedAt: string,
): Promise<void> {
  const db = await database();
  await db.runAsync(
    "DELETE FROM my_tasks_cache WHERE server_id = ? AND account_id = ? AND updated_at = ?",
    serverId,
    accountId,
    updatedAt,
  );
}

function isCachedTask(value: unknown): value is Task {
  return (
    typeof value === "object" &&
    value !== null &&
    "id" in value &&
    typeof value.id === "string" &&
    "spaceSlug" in value &&
    typeof value.spaceSlug === "string" &&
    "title" in value &&
    typeof value.title === "string"
  );
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
  const normalizedBaseUrl = normalizeServerUrl(baseUrl);
  const existing = await db.getFirstAsync<{ id: string }>(
    "SELECT id FROM server_profiles WHERE base_url = ? LIMIT 1",
    normalizedBaseUrl,
  );
  const profile = createServerProfile({
    id: existing?.id ?? Crypto.randomUUID(),
    baseUrl: normalizedBaseUrl,
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
  const result = await db.runAsync(
    "UPDATE server_profiles SET account_id = ? WHERE id = ? AND active = 1",
    accountId,
    profileId,
  );
  if (result.changes !== 1) throw new Error("The selected server is no longer active");
}

export async function detachActiveAccount(profileId: string, accountId: string): Promise<void> {
  const db = await database();
  await db.runAsync(
    "UPDATE server_profiles SET account_id = NULL WHERE id = ? AND account_id = ?",
    profileId,
    accountId,
  );
}
