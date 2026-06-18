import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import type { ChartPointWithLabel } from "./types";

type DailyCompletionChartProps = {
  data: ChartPointWithLabel[];
  labels?: {
    completed: string;
    completedValue: (value: string | number) => string;
  };
};

const defaultLabels = {
  completed: "완료",
  completedValue: (value: string | number) => `${value}개`,
};

export default function DailyCompletionChart({
  data,
  labels = defaultLabels,
}: DailyCompletionChartProps) {
  const partialLabel = labels.completed === "완료" ? "부분 완료" : "Partial";
  const chartData = data.map((point) => ({
    ...point,
    partialCount:
      point.completedCount > 0 && point.completedCount < point.activeGoalCount
        ? point.activeGoalCount - point.completedCount
        : 0,
  }));

  return (
    <div className="h-72 min-h-72">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={chartData} margin={{ top: 10, left: -20, right: 12, bottom: 0 }}>
          <CartesianGrid
            stroke="var(--chart-grid)"
            strokeDasharray="3 3"
            vertical={false}
          />
          <XAxis
            dataKey="dayLabel"
            interval={3}
            tick={{ fontSize: 12, fill: "var(--chart-axis)" }}
          />
          <YAxis
            allowDecimals={false}
            domain={[0, 5]}
            tick={{ fontSize: 12, fill: "var(--chart-axis)" }}
          />
          <Tooltip
            contentStyle={{
              borderRadius: 8,
              backgroundColor: "var(--panel-bg)",
              borderColor: "var(--border-subtle)",
              boxShadow: "var(--shadow-popover)",
              color: "var(--text-primary)",
            }}
            formatter={(value, name) => [
              name === "completedCount" || name === "partialCount"
                ? labels.completedValue(formatTooltipValue(value))
                : value,
              name === "completedCount"
                ? labels.completed
                : name === "partialCount"
                ? partialLabel
                : name,
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
          <Legend
            iconType="rect"
            wrapperStyle={{
              color: "var(--text-secondary)",
              fontSize: 12,
              fontWeight: 700,
              paddingTop: 10,
            }}
          />
          <Bar
            dataKey="completedCount"
            fill="var(--accent-sage)"
            name={labels.completed}
            radius={[3, 3, 0, 0]}
          />
          <Bar
            dataKey="partialCount"
            fill="var(--accent-denim)"
            name={partialLabel}
            radius={[3, 3, 0, 0]}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

function formatTooltipValue(value: unknown) {
  if (Array.isArray(value)) {
    return value.join(" - ");
  }

  if (typeof value === "number" || typeof value === "string") {
    return value;
  }

  return String(value ?? "");
}
