import { useState, useEffect, useRef } from "react";
import type { BodyRecord as BodyRecordType } from "../types";
import { getBodyRecords, saveDayRecords, importRecordsCSV, exportRecordsUrl } from "../services/api";
import { CalendarDays, Download, Upload } from "lucide-react";
import RecordChart from "./RecordChart";
import DoctorSummary from "./DoctorSummary";
import HealthLog from "./HealthLog";

// List of all supported record types for the form and trend charts.
// Chart colors are a validated categorical palette for the dark surface.
const SUPPORTED_RECORD_TYPES = [
  { value: "Weight", unit: "kg", color: "#16A34A" },
  { value: "Sleep", unit: "hours", color: "#8B5CF6" },
  { value: "Activity", unit: "minutes", color: "#D97706" },
  { value: "Water", unit: "L", color: "#0284C7" }
];

// Today as YYYY-MM-DD in local time (toISOString would shift the day in
// timezones ahead of UTC)
const todayStr = () => {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
};

export default function Health() {
  const [records, setRecords] = useState<BodyRecordType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [importMsg, setImportMsg] = useState<string | null>(null);
  const importInput = useRef<HTMLInputElement>(null);

  // Daily record sheet: the chosen day and one editable field per metric
  const [sheetDate, setSheetDate] = useState(todayStr());
  const [fields, setFields] = useState<Record<string, string>>({});
  const [isSaving, setIsSaving] = useState(false);
  const [savedMsg, setSavedMsg] = useState<string | null>(null);

  // --- Fetch Data on Load ---
  useEffect(() => {
    fetchRecords();
  }, []);

  const fetchRecords = async () => {
    setLoading(true);
    setError(null);
    try {
      const fetchedRecords = await getBodyRecords();
      setRecords(fetchedRecords);
    } catch (err) {
      setError("Failed to load records. Check API connection.");
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  // Pre-fill the sheet with the chosen day's existing values whenever
  // the day or the underlying records change.
  useEffect(() => {
    const next: Record<string, string> = {};
    for (const m of SUPPORTED_RECORD_TYPES) {
      const forDay = records.filter(
        (r) => r.recordType === m.value && r.timestamp?.slice(0, 10) === sheetDate
      );
      // most recent wins if the day has legacy duplicates
      const latest = forDay.sort((a, b) =>
        (b.timestamp || "").localeCompare(a.timestamp || "")
      )[0];
      next[m.value] = latest ? String(latest.value) : "";
    }
    setFields(next);
    setSavedMsg(null);
  }, [sheetDate, records]);

  const handleSaveDay = async () => {
    setError(null);
    setSavedMsg(null);
    setIsSaving(true);

    // Build the metrics payload: a number for a filled field, null to
    // clear a field that had a value, and omit fields that stay empty.
    const metrics: Record<string, number | null> = {};
    for (const m of SUPPORTED_RECORD_TYPES) {
      const raw = (fields[m.value] ?? "").trim();
      const existing = records.some(
        (r) => r.recordType === m.value && r.timestamp?.slice(0, 10) === sheetDate
      );
      if (raw === "") {
        if (existing) metrics[m.value] = null; // cleared → delete
        continue;
      }
      const num = parseFloat(raw);
      if (isNaN(num) || num <= 0) {
        setError(`Enter a valid number for ${m.value}, or leave it blank.`);
        setIsSaving(false);
        return;
      }
      metrics[m.value] = num;
    }

    try {
      const updated = await saveDayRecords(sheetDate, metrics);
      setRecords(updated);
      setSavedMsg(sheetDate === todayStr() ? "Saved today's record." : `Saved record for ${sheetDate}.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save records.");
    } finally {
      setIsSaving(false);
    }
  };

  const dayHasData = SUPPORTED_RECORD_TYPES.some((m) => (fields[m.value] ?? "").trim() !== "");

  // --- Render Functions ---

  const RecordForm = (
    <div className="bg-night-850 border border-night-800 p-6 rounded-xl mb-6">
      <div className="flex flex-wrap items-center justify-between gap-3 mb-5">
        <h3 className="text-lg font-medium text-mint-soft flex items-center">
          <CalendarDays className="mr-2 w-5 h-5" /> Daily record
        </h3>
        <div className="flex items-center gap-2">
          <label className="text-sm text-night-400">Date</label>
          <input
            type="date"
            value={sheetDate}
            max={todayStr()}
            onChange={(e) => setSheetDate(e.target.value)}
            className="p-2 bg-night-900 border border-night-700 text-night-50 rounded-lg focus:outline-none focus:border-mint"
          />
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {SUPPORTED_RECORD_TYPES.map((m) => (
          <label key={m.value} className="flex flex-col gap-1.5">
            <span className="flex items-center gap-2 text-sm text-night-200">
              <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ backgroundColor: m.color }} />
              {m.value}
            </span>
            <div className="flex items-center bg-night-900 border border-night-700 rounded-lg focus-within:border-mint">
              <input
                type="number"
                inputMode="decimal"
                value={fields[m.value] ?? ""}
                onChange={(e) => setFields((prev) => ({ ...prev, [m.value]: e.target.value }))}
                placeholder="—"
                className="flex-1 min-w-0 p-3 bg-transparent text-night-50 placeholder-night-500 rounded-lg focus:outline-none"
              />
              <span className="px-3 text-sm text-night-400 shrink-0">{m.unit}</span>
            </div>
          </label>
        ))}
      </div>

      <div className="flex items-center gap-4 mt-5">
        <button
          onClick={handleSaveDay}
          disabled={isSaving}
          className={`px-5 py-2.5 rounded-lg bg-mint text-mint-ink font-medium transition ${
            isSaving ? "opacity-50 cursor-not-allowed" : "hover:opacity-90"
          }`}
        >
          {isSaving ? "Saving..." : "Save day"}
        </button>
        {savedMsg && <span className="text-sm text-mint-soft">{savedMsg}</span>}
        {!savedMsg && !dayHasData && (
          <span className="text-sm text-night-500">
            Fill in what you have — leave the rest blank.
          </span>
        )}
      </div>

      {error && (
        <p className="mt-3 text-sm text-red-400 p-2 bg-red-950/40 rounded-lg border border-red-900">
          {error}
        </p>
      )}
    </div>
  );

  const RecordChartSection = (
    <div className="bg-night-850 border border-night-800 p-6 rounded-xl mb-6">
      <h3 className="text-xl font-medium text-mint-soft mb-4">Trends</h3>

      {loading ? (
        <div className="h-64 flex items-center justify-center text-night-400">
            Loading charts...
        </div>
      ) : (
        <RecordChart data={records} metrics={SUPPORTED_RECORD_TYPES} />
      )}
    </div>
  );

  const handleImportFile = async (file: File) => {
    setImportMsg(null);
    try {
      const csv = await file.text();
      const res = await importRecordsCSV(csv);
      setImportMsg(
        `Imported ${res.imported} records` +
          (res.skipped_duplicates ? `, ${res.skipped_duplicates} duplicates skipped` : "") +
          (res.skipped_invalid ? `, ${res.skipped_invalid} invalid rows skipped` : "") +
          "."
      );
      if (res.imported > 0) fetchRecords();
    } catch (err) {
      setImportMsg(err instanceof Error ? err.message : "Import failed.");
    }
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between border-b border-night-800 pb-3 print:hidden">
        <h2 className="text-2xl font-medium text-night-50">Health</h2>
        <div className="flex gap-2">
          <a
            href={exportRecordsUrl()}
            download
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-night-700 text-sm text-night-200 hover:bg-night-800"
          >
            <Download className="w-3.5 h-3.5" /> Export CSV
          </a>
          <button
            onClick={() => importInput.current?.click()}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-night-700 text-sm text-night-200 hover:bg-night-800"
          >
            <Upload className="w-3.5 h-3.5" /> Import CSV
          </button>
          <input
            ref={importInput}
            type="file"
            accept=".csv,text/csv"
            className="hidden"
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) handleImportFile(f);
              e.target.value = "";
            }}
          />
        </div>
      </div>
      {importMsg && <p className="text-sm text-mint-soft print:hidden">{importMsg}</p>}

      {/* Health log at the top */}
      <HealthLog />

      {/* Daily record + trends */}
      <div className="space-y-6 print:hidden">
        {RecordForm}
        {RecordChartSection}
      </div>

      {/* Doctor summary (prints) */}
      <DoctorSummary />
    </div>
  );
}