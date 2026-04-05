import { useEffect, useId, useMemo, useRef, useState } from "react";
import type { components } from "../../api/schema.d.ts";

type TaskOverdueAction = components["schemas"]["TaskOverdueAction"];
type TaskOverdueActionRule = components["schemas"]["TaskOverdueActionRule"];
type TaskStatus = components["schemas"]["TaskStatus"];

// ─── Types ───────────────────────────────────────────────────────────────────

type SetStatusDraft = { action: "set_status"; status: string };

type DraftAction = { action: "advance_recurrence" } | { action: "clear_due_date" } | SetStatusDraft;

interface DraftState {
  enabled: boolean;
  afterMode: "immediate" | "days";
  afterDays: number;
  actionDraft: DraftAction;
}

// ─── Constants ───────────────────────────────────────────────────────────────

const OVERDUE_ACTION_LABELS: Record<TaskOverdueAction, string> = {
  advance_recurrence: "Advance to next recurrence",
  set_status: "Set status",
  clear_due_date: "Clear due date",
};

const OVERDUE_ACTION_VALUES: TaskOverdueAction[] = [
  "advance_recurrence",
  "set_status",
  "clear_due_date",
];

function isOverdueAction(value: string): value is TaskOverdueAction {
  return OVERDUE_ACTION_VALUES.some((v) => v === value);
}

// ─── State Helpers ───────────────────────────────────────────────────────────

function toDraftState(rule: TaskOverdueActionRule | null): DraftState {
  if (!rule) {
    return {
      enabled: false,
      afterMode: "immediate",
      afterDays: 1,
      actionDraft: { action: "advance_recurrence" },
    };
  }
  return {
    enabled: true,
    afterMode: rule.after == null ? "immediate" : "days",
    afterDays: rule.after ?? 1,
    actionDraft:
      rule.action === "set_status"
        ? { action: "set_status", status: rule.status ?? "" }
        : { action: rule.action },
  };
}

function toPayload(draft: DraftState): TaskOverdueActionRule | null {
  if (!draft.enabled) return null;
  const after = draft.afterMode === "immediate" ? null : draft.afterDays;
  if (draft.actionDraft.action === "set_status") {
    const rule: TaskOverdueActionRule = { after, action: "set_status" };
    if (draft.actionDraft.status) {
      rule.status = draft.actionDraft.status;
    }
    return rule;
  }
  return { after, action: draft.actionDraft.action };
}

// ─── Component ───────────────────────────────────────────────────────────────

export function OverdueActionEditor({
  overdueActionRule,
  statuses,
  onSave,
  disabled,
}: {
  overdueActionRule: TaskOverdueActionRule | null;
  statuses: TaskStatus[];
  onSave: (rule: TaskOverdueActionRule | null) => void;
  disabled?: boolean;
}) {
  const radioName = useId();
  const [draft, setDraft] = useState<DraftState>(() => toDraftState(overdueActionRule));
  const [editing, setEditing] = useState(false);
  const cancellingRef = useRef(false);

  const [savedRule, setSavedRule] = useState<TaskOverdueActionRule | null>(() => overdueActionRule);
  const [savedPayload, setSavedPayload] = useState(() =>
    JSON.stringify(toPayload(toDraftState(overdueActionRule))),
  );

  // Sync from props when not editing
  useEffect(() => {
    if (!editing) {
      const newDraft = toDraftState(overdueActionRule);
      setDraft(newDraft);
      setSavedRule(overdueActionRule);
      setSavedPayload(JSON.stringify(toPayload(newDraft)));
    }
  }, [overdueActionRule, editing]);

  const currentPayload = useMemo(() => JSON.stringify(toPayload(draft)), [draft]);
  const isDirty = currentPayload !== savedPayload;

  function save() {
    setEditing(false);
    if (!isDirty) return;
    const payload = toPayload(draft);
    setSavedRule(payload);
    setSavedPayload(JSON.stringify(payload));
    onSave(payload);
  }

  function cancel() {
    setDraft(toDraftState(savedRule));
    setEditing(false);
  }

  function update(patch: Partial<DraftState>) {
    setDraft((prev) => ({ ...prev, ...patch }));
  }

  const setStatusDraft: SetStatusDraft | null =
    draft.actionDraft.action === "set_status" ? draft.actionDraft : null;

  return (
    <div
      className="flex flex-col gap-3"
      onFocus={() => setEditing(true)}
      onBlur={(e) => {
        if (cancellingRef.current) {
          cancellingRef.current = false;
          return;
        }
        if (!(e.relatedTarget instanceof Node) || !e.currentTarget.contains(e.relatedTarget)) {
          save();
        }
      }}
    >
      {/* Enable toggle */}
      <label className="flex items-center gap-2">
        <input
          type="checkbox"
          checked={draft.enabled}
          onChange={(e) => update({ enabled: e.target.checked })}
          disabled={disabled}
          className="accent-primary-500"
        />
        <span className="text-sm">Enable overdue action</span>
      </label>

      {draft.enabled && (
        <>
          {/* When */}
          <div className="flex flex-col gap-2">
            <span className="text-surface-600-400 text-sm">When</span>
            <div className="flex flex-wrap items-center gap-3">
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  name={radioName}
                  checked={draft.afterMode === "immediate"}
                  onChange={() => update({ afterMode: "immediate" })}
                  disabled={disabled}
                  className="accent-primary-500"
                />
                <span className="text-sm">Immediately when overdue</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  name={radioName}
                  checked={draft.afterMode === "days"}
                  onChange={() => update({ afterMode: "days" })}
                  disabled={disabled}
                  className="accent-primary-500"
                />
                <span className="text-sm">After</span>
                {draft.afterMode === "days" && (
                  <>
                    <input
                      type="number"
                      min={1}
                      value={draft.afterDays}
                      onChange={(e) =>
                        update({ afterDays: Math.max(1, parseInt(e.target.value, 10) || 1) })
                      }
                      disabled={disabled}
                      className="input preset-outlined-surface-200-800 w-20 text-center"
                      aria-label="Days after due date"
                    />
                    <span className="text-sm">days</span>
                  </>
                )}
              </label>
            </div>
          </div>

          {/* Action */}
          <div className="flex flex-col gap-1">
            <span className="text-surface-600-400 text-sm">Action</span>
            <select
              value={draft.actionDraft.action}
              onChange={(e) => {
                if (!isOverdueAction(e.target.value)) return;
                const action = e.target.value;
                if (action === "set_status") {
                  const firstStatus = statuses[0]?.name ?? "";
                  update({ actionDraft: { action: "set_status", status: firstStatus } });
                } else {
                  update({ actionDraft: { action } });
                }
              }}
              disabled={disabled}
              className="select preset-outlined-surface-200-800 w-full"
              aria-label="Overdue action"
            >
              {OVERDUE_ACTION_VALUES.map((value) => (
                <option key={value} value={value}>
                  {OVERDUE_ACTION_LABELS[value]}
                </option>
              ))}
            </select>
          </div>

          {/* Status (only for set_status) */}
          {setStatusDraft && (
            <div className="flex flex-col gap-1">
              <span className="text-surface-600-400 text-sm">Status</span>
              <select
                value={setStatusDraft.status}
                onChange={(e) => {
                  update({ actionDraft: { action: "set_status", status: e.target.value } });
                }}
                disabled={disabled}
                className="select preset-outlined-surface-200-800 w-full"
                aria-label="Target status"
              >
                {!statuses.some((s) => s.name === setStatusDraft.status) &&
                  setStatusDraft.status && (
                    <option value={setStatusDraft.status} disabled>
                      {setStatusDraft.status} (removed)
                    </option>
                  )}
                {statuses.map((s) => (
                  <option key={s.name} value={s.name}>
                    {s.name}
                  </option>
                ))}
              </select>
            </div>
          )}
        </>
      )}

      {/* Action bar */}
      {isDirty && (
        <div className="flex justify-end gap-2 border-t border-surface-200-800 pt-2">
          <button
            type="button"
            onMouseDown={() => {
              cancellingRef.current = true;
            }}
            onClick={() => {
              cancellingRef.current = false;
              cancel();
            }}
            disabled={disabled}
            className="btn btn-sm preset-outlined-surface-200-800"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={save}
            disabled={disabled}
            className="btn btn-sm preset-filled-primary-500"
          >
            Save
          </button>
        </div>
      )}
    </div>
  );
}
