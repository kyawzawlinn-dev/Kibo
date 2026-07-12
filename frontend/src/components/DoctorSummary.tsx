import { useEffect, useMemo, useState } from "react";
import { Printer } from "lucide-react";
import { getBodyRecords, getHealthLog } from "../services/api";
import type { BodyRecord, HealthLogEntry } from "../types";

const METRICS = [
  { type: "Weight", unit: "kg" },
  { type: "Sleep", unit: "hours" },
  { type: "Activity", unit: "minutes" },
  { type: "Water", unit: "L" },
];

const RANGES = [
  { label: "30 days", days: 30 },
  { label: "90 days", days: 90 },
  { label: "All", days: 0 },
];

const fmtDate = (ts?: string) =>
  ts ? new Date(ts).toLocaleDateString() : "—";

// health-log dates are plain YYYY-MM-DD; read at noon so the calendar
// day is timezone-stable
const fmtDay = (d: string) => new Date(`${d}T12:00:00`).toLocaleDateString();

/**
 * A printable summary of the user's health records to bring to a
 * doctor. Rendered as a light "paper preview" regardless of the app
 * theme; printing hides the app chrome (see index.css) so only this
 * lands on paper.
 */
export default function DoctorSummary() {
  const [records, setRecords] = useState<BodyRecord[]>([]);
  const [log, setLog] = useState<HealthLogEntry[]>([]);
  const [days, setDays] = useState(90);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getBodyRecords()
      .then(setRecords)
      .catch(() => setError("Failed to load records."));
    getHealthLog().then(setLog).catch(() => {});
  }, []);

  const logEntries = useMemo(() => {
    const cutoff = days > 0 ? Date.now() - days * 86400000 : 0;
    return log
      .filter((e) => new Date(`${e.date}T12:00:00`).getTime() >= cutoff)
      .sort((a, b) => b.date.localeCompare(a.date));
  }, [log, days]);

  const filtered = useMemo(() => {
    const cutoff = days > 0 ? Date.now() - days * 86400000 : 0;
    return records.filter(
      (r) => r.timestamp && new Date(r.timestamp).getTime() >= cutoff
    );
  }, [records, days]);

  const metricData = useMemo(
    () =>
      METRICS.map((m) => {
        const entries = filtered
          .filter((r) => r.recordType === m.type)
          .sort(
            (a, b) =>
              new Date(b.timestamp || 0).getTime() -
              new Date(a.timestamp || 0).getTime()
          );
        const values = entries.map((r) => r.value);
        return {
          ...m,
          entries,
          latest: entries[0],
          avg: values.length
            ? values.reduce((s, v) => s + v, 0) / values.length
            : null,
          min: values.length ? Math.min(...values) : null,
          max: values.length ? Math.max(...values) : null,
        };
      }),
    [filtered]
  );

  const periodLabel =
    days > 0 ? `last ${days} days` : "all recorded data";

  return (
    <div className="print:p-0 print:max-w-none">
      <h3 className="text-lg font-medium text-mint-soft mb-3 print:hidden">Doctor summary</h3>
      {/* Controls — never printed */}
      <div className="flex items-center justify-between mb-4 print:hidden">
        <div className="flex rounded-lg border border-night-700 overflow-hidden">
          {RANGES.map((r) => (
            <button
              key={r.label}
              onClick={() => setDays(r.days)}
              className={`px-3 py-1.5 text-xs transition-colors ${
                days === r.days
                  ? "bg-mint/10 text-mint-soft font-medium"
                  : "text-night-400 hover:text-night-200"
              }`}
            >
              {r.label}
            </button>
          ))}
        </div>

        <button
          onClick={() => window.print()}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-mint text-mint-ink text-sm font-medium hover:opacity-90"
        >
          <Printer className="w-4 h-4" /> Print
        </button>
      </div>

      {error && <p className="text-red-400 mb-4 print:hidden">{error}</p>}

      {/* The paper */}
      <div className="bg-white text-gray-900 rounded-xl print:rounded-none p-8 print:p-0">
        <div className="flex items-baseline justify-between border-b-2 border-gray-800 pb-3 mb-5">
          <h2 className="text-2xl font-semibold">Health summary</h2>
          <p className="text-sm text-gray-500">
            Period: {periodLabel} · Printed {new Date().toLocaleDateString()}
          </p>
        </div>

        <div className="flex gap-8 mb-6 text-sm">
          <p className="flex-1">
            Name: <span className="inline-block w-56 border-b border-gray-400" />
          </p>
          <p className="flex-1">
            Date of birth: <span className="inline-block w-40 border-b border-gray-400" />
          </p>
        </div>

        {/* Health log — the episode history a clinician reads first */}
        <div className="mb-6 break-inside-avoid">
          <h3 className="text-base font-semibold border-b border-gray-300 pb-1 mb-2">
            Health log
          </h3>
          {logEntries.length === 0 ? (
            <p className="text-sm text-gray-500 italic">No episodes in this period.</p>
          ) : (
            <table className="w-full text-sm border-collapse">
              <tbody>
                {logEntries.map((e) => (
                  <tr key={e.id} className="border-b border-gray-200 align-top">
                    <td className="py-1 text-gray-600 w-28">{fmtDay(e.date)}</td>
                    <td className="py-1">
                      <span className="font-medium">{e.title}</span>
                      {e.severity ? ` (${e.severity})` : ""}
                      {e.notes ? <span className="text-gray-600"> — {e.notes}</span> : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {metricData.map((m) => (
          <div key={m.type} className="mb-6 break-inside-avoid">
            <h3 className="text-base font-semibold border-b border-gray-300 pb-1 mb-2">
              {m.type} ({m.unit})
            </h3>

            {m.entries.length === 0 ? (
              <p className="text-sm text-gray-500 italic">
                No entries in this period.
              </p>
            ) : (
              <>
                <p className="text-sm text-gray-700 mb-2">
                  {m.entries.length} entries · Latest:{" "}
                  <strong>
                    {m.latest!.value} {m.latest!.unit}
                  </strong>{" "}
                  ({fmtDate(m.latest!.timestamp)}) · Average:{" "}
                  {m.avg!.toFixed(1)} · Range: {m.min}–{m.max}
                </p>
                <table className="w-full text-sm border-collapse">
                  <tbody>
                    {m.entries.slice(0, 8).map((r) => (
                      <tr key={r.id} className="border-b border-gray-200">
                        <td className="py-1 text-gray-600 w-40">
                          {fmtDate(r.timestamp)}
                        </td>
                        <td className="py-1">
                          {r.value} {r.unit}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </>
            )}
          </div>
        ))}

        <div className="mt-8 break-inside-avoid">
          <h3 className="text-base font-semibold border-b border-gray-300 pb-1 mb-3">
            Notes
          </h3>
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className="border-b border-gray-300 h-7" />
          ))}
        </div>

        <p className="text-xs text-gray-400 mt-6">
          Self-recorded data, generated by Kibo (offline health companion).
          Not a medical record.
        </p>
      </div>
    </div>
  );
}
