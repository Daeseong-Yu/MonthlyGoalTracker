import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import type { ChartPointWithLabel } from "./types";

type DailyCompletionChartProps = {
  data: ChartPointWithLabel[];
};

export default function DailyCompletionChart({
  data,
}: DailyCompletionChartProps) {
  return (
    <div className="h-72 min-h-72">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ left: -20, right: 12 }}>
          <CartesianGrid stroke="#e5e7eb" strokeDasharray="3 3" />
          <XAxis
            dataKey="dayLabel"
            interval={3}
            tick={{ fontSize: 12, fill: "#52525b" }}
          />
          <YAxis
            allowDecimals={false}
            domain={[0, 5]}
            tick={{ fontSize: 12, fill: "#52525b" }}
          />
          <Tooltip
            contentStyle={{
              borderRadius: 8,
              borderColor: "#d4d4d8",
              boxShadow: "0 12px 28px rgba(39, 39, 42, 0.12)",
            }}
            formatter={(value, name) => [
              name === "completedCount" ? `${value}개` : value,
              name === "completedCount" ? "완료" : name,
            ]}
            labelFormatter={(_, payload) => {
              const point = payload?.[0]?.payload as
                | ChartPointWithLabel
                | undefined;
              return point
                ? `${point.date} · ${Math.round(point.completionRate * 100)}%`
                : "";
            }}
          />
          <Line
            type="monotone"
            dataKey="completedCount"
            stroke="#0f766e"
            strokeWidth={3}
            dot={{ r: 3, strokeWidth: 2, fill: "#ffffff" }}
            activeDot={{ r: 5 }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
