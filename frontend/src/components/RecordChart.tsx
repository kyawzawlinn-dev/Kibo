import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import type { BodyRecord } from "../types";

interface RecordChartProps {
  data: BodyRecord[];
  type: string;
  unit: string;
}

// Function to format the date for the X-axis
const formatDate = (timestamp: string | undefined): string => {
    if (!timestamp) return 'N/A';
    const date = new Date(timestamp);
    // Format: MM/DD
    return `${date.getMonth() + 1}/${date.getDate()}`;
};

/**
 * Renders a responsive line chart for a specific type of body record.
 */
export default function RecordChart({ data, type, unit }: RecordChartProps) {
    
    // 1. Filter data to only include the requested type
    const filteredData = data
        .filter(record => record.recordType === type)
        // 2. Sort data by timestamp (oldest first) for charting
        .sort((a, b) => new Date(a.timestamp || 0).getTime() - new Date(b.timestamp || 0).getTime());

    if (filteredData.length === 0) {
        return (
            <div className="text-center p-8 bg-gray-50 rounded-lg border border-dashed border-gray-300">
                <p className="text-gray-500">No data points available for **{type}** yet.</p>
                <p className="text-sm text-gray-400 mt-1">Add records using the form above to see the trend.</p>
            </div>
        );
    }

    return (
        <div className="h-80 w-full">
            <ResponsiveContainer width="100%" height="100%">
                <LineChart
                    data={filteredData}
                    margin={{ top: 10, right: 30, left: 0, bottom: 0 }}
                >
                    <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
                    
                    {/* X-Axis: Date */}
                    <XAxis 
                        dataKey="timestamp" 
                        tickFormatter={formatDate}
                        stroke="#6b7280" 
                        fontSize={12}
                    />
                    
                    {/* Y-Axis: Value */}
                    <YAxis 
                        label={{ value: unit, angle: -90, position: 'insideLeft', style: { textAnchor: 'middle', fill: '#10b981' } }} 
                        stroke="#6b7280" 
                        fontSize={12}
                    />
                    
                    {/* Tooltip on hover */}
                    <Tooltip 
                        contentStyle={{ 
                            backgroundColor: '#fff', 
                            border: '1px solid #e0e0e0', 
                            borderRadius: '8px',
                            padding: '8px'
                        }}
                        formatter={(value, name) => [`${value} ${unit}`, 'Value']}
                        labelFormatter={(timestamp) => `Date: ${new Date(timestamp).toLocaleDateString()}`}
                    />
                    
                    {/* Line visualization */}
                    <Line 
                        type="monotone" 
                        dataKey="value" 
                        name={type} 
                        stroke="#10b981" // Tailwind green-500
                        strokeWidth={2}
                        dot={{ r: 4 }}
                        activeDot={{ r: 6, fill: '#10b981', stroke: '#fff', strokeWidth: 2 }}
                    />
                </LineChart>
            </ResponsiveContainer>
        </div>
    );
}