import { useMemo, useState } from "react";
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import type { BodyRecord } from "../types";

export interface Metric {
  value: string; // record type name, e.g. "Weight"
  unit: string;
  color: string;
}

interface CombinedChartProps {
  data: BodyRecord[];
  metrics: Metric[];
}

const RANGES = [
  { label: "7 days", days: 7 },
  { label: "30 days", days: 30 },
  { label: "All", days: 0 },
];

// "2026-07-09T…" -> "7/9"
const formatDay = (day: string): string => `${+day.slice(5, 7)}/${+day.slice(8, 10)}`;

/**
 * All four metrics in one chart. The metrics use different units and
 * scales (kg vs litres), so each line is normalized to its own 0-100
 * range to stay readable; the tooltip always shows the real values.
 */
export default function RecordChart({ data, metrics }: CombinedChartProps) {
    const [rangeDays, setRangeDays] = useState(0);
    const [hidden, setHidden] = useState<Set<string>>(new Set());

    const toggleMetric = (name: string) => {
        setHidden(prev => {
            const next = new Set(prev);
            if (next.has(name)) next.delete(name);
            else next.add(name);
            return next;
        });
    };

    // Latest record per metric, for the legend chips
    const latestByMetric = useMemo(() => {
        const out: Record<string, BodyRecord> = {};
        for (const r of data) {
            const t = new Date(r.timestamp || 0).getTime();
            const cur = out[r.recordType];
            if (!cur || t > new Date(cur.timestamp || 0).getTime()) out[r.recordType] = r;
        }
        return out;
    }, [data]);

    // One row per calendar day; each metric normalized to its own range,
    // with the raw value kept alongside for the tooltip
    const chartData = useMemo(() => {
        const cutoff = rangeDays > 0 ? Date.now() - rangeDays * 86400000 : 0;

        const byDay = new Map<string, Record<string, any>>();
        const sorted = [...data].sort(
            (a, b) => new Date(a.timestamp || 0).getTime() - new Date(b.timestamp || 0).getTime()
        );
        for (const r of sorted) {
            if (!r.timestamp) continue;
            if (cutoff && new Date(r.timestamp).getTime() < cutoff) continue;
            const day = r.timestamp.slice(0, 10);
            const row = byDay.get(day) ?? { day };
            row[r.recordType] = r.value; // later record wins within a day
            byDay.set(day, row);
        }

        const rows = [...byDay.values()].sort((a, b) => a.day.localeCompare(b.day));

        for (const m of metrics) {
            const vals = rows.map(r => r[m.value]).filter((v): v is number => v != null);
            if (vals.length === 0) continue;
            const min = Math.min(...vals);
            const max = Math.max(...vals);
            for (const row of rows) {
                if (row[m.value] == null) continue;
                row[`${m.value}Raw`] = row[m.value];
                row[m.value] = max === min ? 50 : ((row[m.value] - min) / (max - min)) * 100;
            }
        }
        return rows;
    }, [data, metrics, rangeDays]);

    const unitOf = (name: string) => metrics.find(m => m.value === name)?.unit ?? "";

    return (
        <div>
            {/* Legend chips (click to show/hide) + range filter */}
            <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
                <div className="flex flex-wrap gap-2">
                    {metrics.map(m => {
                        const off = hidden.has(m.value);
                        const latest = latestByMetric[m.value];
                        return (
                            <button
                                key={m.value}
                                onClick={() => toggleMetric(m.value)}
                                title={off ? `Show ${m.value}` : `Hide ${m.value}`}
                                className={`flex items-center gap-2 px-3 py-1.5 rounded-lg border text-sm transition-opacity
                                    border-night-700 ${off ? "opacity-40" : ""}`}
                            >
                                <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ backgroundColor: m.color }} />
                                <span className="text-night-200">{m.value}</span>
                                <span className="text-night-50 font-medium">
                                    {latest ? `${latest.value} ${latest.unit}` : "—"}
                                </span>
                            </button>
                        );
                    })}
                </div>

                <div className="flex rounded-lg border border-night-700 overflow-hidden">
                    {RANGES.map(r => (
                        <button
                            key={r.label}
                            onClick={() => setRangeDays(r.days)}
                            className={`px-3 py-1.5 text-xs transition-colors ${
                                rangeDays === r.days
                                    ? "bg-mint/10 text-mint-soft font-medium"
                                    : "text-night-400 hover:text-night-200"
                            }`}
                        >
                            {r.label}
                        </button>
                    ))}
                </div>
            </div>

            {chartData.length === 0 ? (
                <div className="h-72 flex items-center justify-center text-night-500 border border-dashed border-night-700 rounded-lg">
                    No data in this period
                </div>
            ) : (
                <div className="h-72 w-full">
                    <ResponsiveContainer width="100%" height="100%">
                        <LineChart data={chartData} margin={{ top: 8, right: 12, left: 12, bottom: 0 }}>
                            <CartesianGrid strokeDasharray="3 3" stroke="#2A3A34" vertical={false} />

                            <XAxis
                                dataKey="day"
                                tickFormatter={formatDay}
                                stroke="#7C918A"
                                fontSize={11}
                                tickLine={false}
                                minTickGap={20}
                            />

                            {/* Values are per-metric normalized, so numeric ticks
                                would be meaningless — the tooltip carries real values */}
                            <YAxis hide domain={[0, 100]} />

                            <Tooltip
                                contentStyle={{
                                    backgroundColor: '#1A2420',
                                    border: '1px solid #2A3A34',
                                    borderRadius: '8px',
                                    padding: '8px',
                                    color: '#D2E8DE'
                                }}
                                labelStyle={{ color: '#A8C4B8' }}
                                formatter={(_value, name, props) => [
                                    `${props.payload[`${name}Raw`]} ${unitOf(String(name))}`,
                                    name,
                                ]}
                                labelFormatter={(day) => new Date(`${day}T12:00:00`).toLocaleDateString()}
                            />

                            {metrics.map(m => (
                                <Line
                                    key={m.value}
                                    type="monotone"
                                    dataKey={m.value}
                                    hide={hidden.has(m.value)}
                                    connectNulls
                                    stroke={m.color}
                                    strokeWidth={2}
                                    dot={{ r: 3, fill: m.color, stroke: '#111816' }}
                                    activeDot={{ r: 5, fill: m.color, stroke: '#D2E8DE', strokeWidth: 2 }}
                                />
                            ))}
                        </LineChart>
                    </ResponsiveContainer>
                </div>
            )}

            <p className="text-xs text-night-500 mt-2">
                Each line is scaled to its own range so all metrics stay visible — hover a point for actual values.
            </p>
        </div>
    );
}
