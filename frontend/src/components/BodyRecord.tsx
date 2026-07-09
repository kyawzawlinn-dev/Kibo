import { useState, useEffect, useMemo } from "react";
import type { BodyRecord as BodyRecordType } from "../types";
import { getBodyRecords, addBodyRecord } from "../services/api";
import { PlusCircle } from "lucide-react";
import RecordChart from "./RecordChart"; // Import the new chart component

// List of all supported record types for the dropdown and charts
const SUPPORTED_RECORD_TYPES = [
  { value: "Weight", unit: "kg" },
  { value: "Sleep", unit: "hours" },
  { value: "Activity", unit: "minutes" },
  { value: "Water", unit: "L" }
];

// Initial Form State
const initialFormState = {
  recordType: "",
  value: "",
  unit: "",
};

export default function BodyRecord() {
  const [records, setRecords] = useState<BodyRecordType[]>([]);
  const [form, setForm] = useState(initialFormState);
  const [loading, setLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // State to control which record type is currently being charted
  const [chartType, setChartType] = useState<string>("Weight");

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

    if (!form.recordType || !form.unit || isNaN(numericValue) || numericValue <= 0) {
        setError("Please fill out all fields correctly.");
        setIsSubmitting(false);
        return;
    }

    try {
        const newRecord = await addBodyRecord({
            recordType: form.recordType,
            value: numericValue,
            unit: form.unit
        });
        
        // Add the new record to the list, set chart to the new type, and reset form
        setRecords(prev => [newRecord, ...prev]);
        setChartType(newRecord.recordType); // Switch chart to the newly added type
        setForm(initialFormState);
    } catch (err) {
        setError("Failed to save record. Please try again.");
        console.error(err);
    } finally {
        setIsSubmitting(false);
    }
  };

  // Memoize the chart unit based on the current chart type
  const currentChartUnit = useMemo(() => {
    return SUPPORTED_RECORD_TYPES.find(t => t.value === chartType)?.unit || '';
  }, [chartType]);

  // --- Render Functions ---

  const RecordForm = (
    <div className="bg-white p-6 rounded-xl shadow-lg mb-6">
      <h3 className="text-lg font-semibold text-green-700 mb-4 flex items-center">
        <PlusCircle className="mr-2 w-5 h-5" /> Add New Record
      </h3>
      <form onSubmit={handleFormSubmit} className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {/* Record Type */}
        <select
          name="recordType"
          value={form.recordType}
          onChange={handleFormChange}
          className="p-3 border border-gray-300 rounded-lg focus:ring-green-500 focus:border-green-500"
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
          className="p-3 border border-gray-300 rounded-lg focus:ring-green-500 focus:border-green-500"
          disabled={isSubmitting}
        />

        {/* Unit */}
        <input
          type="text"
          name="unit"
          value={form.unit}
          onChange={handleFormChange}
          placeholder="Unit (e.g., kg or hours)"
          className="p-3 border border-gray-300 rounded-lg focus:ring-green-500 focus:border-green-500"
          disabled={isSubmitting}
        />
        
        {/* Submit Button */}
        <button
          type="submit"
          className={`p-3 rounded-lg text-white font-semibold transition duration-150 ${
            isSubmitting
              ? "bg-gray-400 cursor-not-allowed"
              : "bg-green-600 hover:bg-green-700"
          }`}
          disabled={isSubmitting}
        >
          {isSubmitting ? "Saving..." : "Save Record"}
        </button>
      </form>
      {error && (
        <p className="mt-3 text-sm text-red-600 p-2 bg-red-50 rounded-lg border border-red-200">
          {error}
        </p>
      )}
    </div>
  );
  
  const RecordChartSection = (
    <div className="bg-white p-6 rounded-xl shadow-lg mb-6">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-xl font-semibold text-green-700">
            {chartType} Trend
        </h3>
        <select
          value={chartType}
          onChange={(e) => setChartType(e.target.value)}
          className="p-2 border border-gray-300 rounded-lg text-sm"
        >
          {SUPPORTED_RECORD_TYPES.map(type => (
            <option key={`chart-${type.value}`} value={type.value}>
              {type.value}
            </option>
          ))}
        </select>
      </div>
      
      {loading ? (
        <div className="h-64 flex items-center justify-center text-gray-500">
            Loading charts...
        </div>
      ) : (
        <RecordChart 
            data={records} 
            type={chartType} 
            unit={currentChartUnit}
        />
      )}
      
    </div>
  );

  const RecordList = (
    <div className="bg-white p-6 rounded-xl shadow-lg">
      <h3 className="text-xl font-semibold mb-4 text-green-700">Recent Records</h3>
      
      {loading && <p className="text-gray-500">Loading records...</p>}
      
      {!loading && records.length === 0 && (
        <p className="text-gray-500 italic">No records tracked yet. Add one above!</p>
      )}

      {!loading && records.length > 0 && (
        <ul className="space-y-3">
          {/* Display newest first by reversing the array copy */}
          {[...records].reverse().map((record) => (
            <li
              key={record.id}
              className="flex justify-between items-center p-3 border-b border-gray-100 last:border-b-0 hover:bg-green-50 rounded-lg transition"
            >
              <div className="flex flex-col">
                <span className="font-medium text-gray-800">{record.recordType}</span>
                <span className="text-xs text-gray-400">
                    {record.timestamp ? new Date(record.timestamp).toLocaleString() : 'N/A'}
                </span>
              </div>
              <div className="text-lg font-bold text-green-600">
                {record.value} {record.unit}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <h2 className="text-3xl font-bold text-green-700 mb-6 border-b pb-2">🏋️ Health & Body Record</h2>
      
      {RecordForm}
      
      {RecordChartSection}
      
      {RecordList}
      
    </div>
  );
}