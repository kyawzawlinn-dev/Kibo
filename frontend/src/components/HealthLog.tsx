import { useEffect, useState } from "react";
import { Plus, Trash2, Edit3, NotebookPen } from "lucide-react";
import {
  getHealthLog,
  addHealthLogEntry,
  updateHealthLogEntry,
  deleteHealthLogEntry,
} from "../services/api";
import type { HealthLogEntry } from "../types";

const SEVERITIES = ["", "mild", "moderate", "severe"];

const severityStyle: Record<string, string> = {
  mild: "bg-mint/15 text-mint-soft",
  moderate: "bg-amber-400/15 text-amber-300",
  severe: "bg-red-500/15 text-red-300",
};

const todayStr = () => {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
};

const fmtDate = (d: string) => {
  const dt = new Date(`${d}T12:00:00`);
  return isNaN(dt.getTime()) ? d : dt.toLocaleDateString();
};

const emptyDraft = () => ({ id: 0, date: todayStr(), title: "", severity: "", notes: "" });

export default function HealthLog() {
  const [entries, setEntries] = useState<HealthLogEntry[]>([]);
  const [draft, setDraft] = useState<HealthLogEntry>(emptyDraft());
  const [editingId, setEditingId] = useState<number | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getHealthLog().then(setEntries).catch(() => setError("Failed to load the health log."));
  }, []);

  const openAdd = () => {
    setDraft(emptyDraft());
    setEditingId(null);
    setShowForm(true);
    setError(null);
  };

  const openEdit = (e: HealthLogEntry) => {
    setDraft({ ...e });
    setEditingId(e.id);
    setShowForm(true);
    setError(null);
  };

  const handleSave = async () => {
    if (!draft.title.trim() || !draft.date) {
      setError("Add a date and a short description.");
      return;
    }
    try {
      if (editingId) {
        const updated = await updateHealthLogEntry(draft);
        setEntries((prev) =>
          prev.map((e) => (e.id === updated.id ? updated : e)).sort(byDateDesc)
        );
      } else {
        const created = await addHealthLogEntry({
          date: draft.date,
          title: draft.title.trim(),
          severity: draft.severity,
          notes: draft.notes.trim(),
        });
        setEntries((prev) => [created, ...prev].sort(byDateDesc));
      }
      setShowForm(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save.");
    }
  };

  const handleDelete = async (e: HealthLogEntry) => {
    if (!window.confirm(`Delete "${e.title}" from ${fmtDate(e.date)}?`)) return;
    try {
      await deleteHealthLogEntry(e.id);
      setEntries((prev) => prev.filter((x) => x.id !== e.id));
    } catch {
      setError("Failed to delete.");
    }
  };

  return (
    <div className="bg-night-850 border border-night-800 rounded-xl p-6 print:hidden">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-lg font-medium text-mint-soft flex items-center gap-2">
          <NotebookPen className="w-5 h-5" /> Health log
        </h3>
        <button
          onClick={openAdd}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-mint text-mint-ink text-sm font-medium hover:opacity-90"
        >
          <Plus className="w-4 h-4" /> Add episode
        </button>
      </div>
      <p className="text-sm text-night-400 mb-4">
        Illnesses, symptoms, and visits over time — the history to show a health worker.
      </p>

      {error && <p className="text-sm text-red-400 mb-3">{error}</p>}

      {showForm && (
        <div className="bg-night-900 border border-night-700 rounded-lg p-4 mb-4 space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <input
              type="date"
              value={draft.date}
              max={todayStr()}
              onChange={(e) => setDraft({ ...draft, date: e.target.value })}
              className="p-2.5 bg-night-850 border border-night-700 text-night-50 rounded-lg focus:outline-none focus:border-mint"
            />
            <input
              value={draft.title}
              onChange={(e) => setDraft({ ...draft, title: e.target.value })}
              placeholder="What happened (e.g. Fever)"
              className="p-2.5 bg-night-850 border border-night-700 text-night-50 placeholder-night-400 rounded-lg focus:outline-none focus:border-mint sm:col-span-2"
            />
          </div>
          <select
            value={draft.severity}
            onChange={(e) => setDraft({ ...draft, severity: e.target.value })}
            className="p-2.5 bg-night-850 border border-night-700 text-night-50 rounded-lg focus:outline-none focus:border-mint w-full sm:w-48"
          >
            {SEVERITIES.map((s) => (
              <option key={s} value={s}>
                {s === "" ? "Severity (optional)" : s}
              </option>
            ))}
          </select>
          <textarea
            value={draft.notes}
            onChange={(e) => setDraft({ ...draft, notes: e.target.value })}
            placeholder="Notes — what you did, medicines taken, etc."
            rows={2}
            className="w-full p-2.5 bg-night-850 border border-night-700 text-night-50 placeholder-night-400 rounded-lg focus:outline-none focus:border-mint"
          />
          <div className="flex gap-2">
            <button
              onClick={handleSave}
              className="px-4 py-2 rounded-lg bg-mint text-mint-ink font-medium hover:opacity-90"
            >
              {editingId ? "Save changes" : "Add episode"}
            </button>
            <button
              onClick={() => setShowForm(false)}
              className="px-4 py-2 rounded-lg border border-night-700 text-night-200 hover:bg-night-800"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {entries.length === 0 && !showForm && (
        <p className="text-night-500 italic">No episodes logged yet.</p>
      )}

      <div className="space-y-2">
        {entries.map((e) => (
          <div
            key={e.id}
            className="group flex items-start gap-3 border-b border-night-800 last:border-b-0 py-3"
          >
            <div className="text-xs text-night-400 w-24 shrink-0 pt-0.5">{fmtDate(e.date)}</div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-night-50 font-medium">{e.title}</span>
                {e.severity && (
                  <span className={`text-[11px] px-2 py-0.5 rounded-full ${severityStyle[e.severity] ?? ""}`}>
                    {e.severity}
                  </span>
                )}
              </div>
              {e.notes && <p className="text-sm text-night-300 mt-0.5">{e.notes}</p>}
            </div>
            <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <button onClick={() => openEdit(e)} title="Edit" className="p-1.5 rounded hover:bg-night-800">
                <Edit3 size={14} className="text-night-400 hover:text-mint-soft" />
              </button>
              <button onClick={() => handleDelete(e)} title="Delete" className="p-1.5 rounded hover:bg-night-800">
                <Trash2 size={14} className="text-night-400 hover:text-red-400" />
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function byDateDesc(a: HealthLogEntry, b: HealthLogEntry) {
  return b.date.localeCompare(a.date) || b.id - a.id;
}
