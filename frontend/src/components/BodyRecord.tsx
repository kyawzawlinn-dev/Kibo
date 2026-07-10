import { useState, useEffect, useRef } from "react";
import type { BodyRecord as BodyRecordType } from "../types";
import { getBodyRecords, addBodyRecord, importRecordsCSV, exportRecordsUrl } from "../services/api";
import { Download, PlusCircle, Upload } from "lucide-react";
import RecordChart from "./RecordChart"; // Import the new chart component

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

// Initial Form State
const initialFormState = {
  recordType: "",
  value: "",
  unit: "",
  date: todayStr(),
};

export default function BodyRecord() {
  const [records, setRecords] = useState<BodyRecordType[]>([]);
  const [form, setForm] = useState(initialFormState);
  const [loading, setLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [importMsg, setImportMsg] = useState<string | null>(null);
  const importInput = useRef<HTMLInputElement>(null);

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

  // --- Form Handlers ---
  const handleFormChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    
    // If the recordType changes, update the unit automatically if possible
    if (name === 'recordType') {
        const selectedType = SUPPORTED_RECORD_TYPES.find(t => t.value === value);
        setForm(prev => ({ 
            ...prev, 
            recordType: value,
            unit: selectedType ? selectedType.unit : prev.unit
        }));
    } else {
        setForm(prev => ({ ...prev, [name]: value }));
    }
  };

  const handleFormSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    const numericValue = parseFloat(form.value);

    if (!form.recordType || !form.unit || !form.date || isNaN(numericValue) || numericValue <= 0) {
        setError("Please fill out all fields correctly.");
        setIsSubmitting(false);
        return;
    }

    try {
        // Noon local time keeps the record on the chosen calendar day
        // regardless of timezone
        const newRecord = await addBodyRecord({
            recordType: form.recordType,
            value: numericValue,
            unit: form.unit,
            timestamp: new Date(`${form.date}T12:00:00`).toISOString()
        });
        
        // Add the new record to the list and reset form
        setRecords(prev => [newRecord, ...prev]);
        setForm(initialFormState);
    } catch (err) {
        setError("Failed to save record. Please try again.");
        console.error(err);
    } finally {
        setIsSubmitting(false);
    }
  };

  // --- Render Functions ---

  const RecordForm = (
    <div className="bg-night-850 border border-night-800 p-6 rounded-xl mb-6">
      <h3 className="text-lg font-medium text-mint-soft mb-4 flex items-center">
        <PlusCircle className="mr-2 w-5 h-5" /> Add new record
      </h3>
      <form onSubmit={handleFormSubmit} className="grid grid-cols-1 md:grid-cols-5 gap-4">
        {/* Record Type */}
        <select
          name="recordType"
          value={form.recordType}
          onChange={handleFormChange}
          className="p-3 bg-night-900 border border-night-700 text-night-50 placeholder-night-400 rounded-lg focus:outline-none focus:border-mint"
          disabled={isSubmitting}
        >
          <option value="">Select Type</option>
          {SUPPORTED_RECORD_TYPES.map(type => (
            <option key={type.value} value={type.value}>{type.value}</option>
          ))}
        </select>

        {/* Value */}
        <input
          type="number"
          name="value"
          value={form.value}
          onChange={handleFormChange}
          placeholder="Value (e.g., 75 or 8.5)"
          className="p-3 bg-night-900 border border-night-700 text-night-50 placeholder-night-400 rounded-lg focus:outline-none focus:border-mint"
          disabled={isSubmitting}
        />

        {/* Unit */}
        <input
          type="text"
          name="unit"
          value={form.unit}
          onChange={handleFormChange}
          placeholder="Unit (e.g., kg or hours)"
          className="p-3 bg-night-900 border border-night-700 text-night-50 placeholder-night-400 rounded-lg focus:outline-none focus:border-mint"
          disabled={isSubmitting}
        />

        {/* Date (defaults to today; pick a past date to backfill) */}
        <input
          type="date"
          name="date"
          value={form.date}
          onChange={handleFormChange}
          max={todayStr()}
          className="p-3 bg-night-900 border border-night-700 text-night-50 rounded-lg focus:outline-none focus:border-mint"
          disabled={isSubmitting}
        />

        {/* Submit Button */}
        <button
          type="submit"
          className={`p-3 rounded-lg bg-mint text-mint-ink font-medium transition duration-150 ${
            isSubmitting ? "opacity-50 cursor-not-allowed" : "hover:opacity-90"
          }`}
          disabled={isSubmitting}
        >
          {isSubmitting ? "Saving..." : "Save record"}
        </button>
      </form>
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

  // Newest first, paginated
  const PAGE_SIZE = 10;
  const sortedRecords = [...records].sort(
    (a, b) => new Date(b.timestamp || 0).getTime() - new Date(a.timestamp || 0).getTime()
  );
  const totalPages = Math.max(1, Math.ceil(sortedRecords.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages);
  const pageRecords = sortedRecords.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE);

  const RecordList = (
    <div className="bg-night-850 border border-night-800 p-6 rounded-xl">
      <h3 className="text-xl font-medium mb-4 text-mint-soft">Recent records</h3>

      {loading && <p className="text-night-400">Loading records...</p>}

      {!loading && records.length === 0 && (
        <p className="text-night-400 italic">No records tracked yet. Add one above.</p>
      )}

      {!loading && records.length > 0 && (
        <>
          <ul className="space-y-3">
            {pageRecords.map((record) => (
              <li
                key={record.id}
                className="flex justify-between items-center p-3 border-b border-night-800 last:border-b-0 hover:bg-night-800/60 rounded-lg transition"
              >
                <div className="flex flex-col">
                  <span className="font-medium text-night-50">{record.recordType}</span>
                  <span className="text-xs text-night-400">
                      {record.timestamp ? new Date(record.timestamp).toLocaleString() : 'N/A'}
                  </span>
                </div>
                <div className="text-lg font-medium text-mint">
                  {record.value} {record.unit}
                </div>
              </li>
            ))}
          </ul>

          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4 pt-3 border-t border-night-800">
              <button
                onClick={() => setPage(safePage - 1)}
                disabled={safePage <= 1}
                className={`px-3 py-1.5 rounded-lg border border-night-700 text-sm ${
                  safePage <= 1
                    ? "text-night-500 opacity-50 cursor-not-allowed"
                    : "text-night-200 hover:bg-night-800"
                }`}
              >
                Previous
              </button>

              <span className="text-sm text-night-400">
                Page {safePage} of {totalPages}
              </span>

              <button
                onClick={() => setPage(safePage + 1)}
                disabled={safePage >= totalPages}
                className={`px-3 py-1.5 rounded-lg border border-night-700 text-sm ${
                  safePage >= totalPages
                    ? "text-night-500 opacity-50 cursor-not-allowed"
                    : "text-night-200 hover:bg-night-800"
                }`}
              >
                Next
              </button>
            </div>
          )}
        </>
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
    <div className="p-6 max-w-4xl mx-auto">
      <div className="flex items-center justify-between border-b border-night-800 pb-3 mb-2">
        <h2 className="text-2xl font-medium text-night-50">Health and body record</h2>
        <div className="flex gap-2">
          <a
            href={exportRecordsUrl}
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
      {importMsg && <p className="text-sm text-mint-soft mb-4">{importMsg}</p>}
      <div className="mb-4" />
      
      {RecordForm}
      
      {RecordChartSection}
      
      {RecordList}
      
    </div>
  );
}