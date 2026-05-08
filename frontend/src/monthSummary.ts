import {
  buildDailyRecordGoalSlots,
  isGoalVisibleInDisplay,
} from "./goalSlots";
import {
  buildChartData,
  currentDate,
  getReferencePoint,
  isCurrentMonth,
  monthStartDate,
} from "./monthLogic";
import type { ChartPointWithLabel, Goal, MonthView } from "./types";

export type MonthSummary = {
  activeMetricGoalCount: number;
  activeMetricLabel: string;
  averageRate: number;
  chartData: ChartPointWithLabel[];
  dailyRecordGoalSlots: Goal[][];
  goalListReferenceDate: string;
  totalCompleted: number;
  visibleGoals: Goal[];
};

export function buildMonthSummary(
  monthView: MonthView,
  referenceDate = new Date(),
): MonthSummary {
  const { checks, days, goals, month } = monthView;
  const chartData = buildChartData(days, goals, checks);
  const totalCompleted = chartData.reduce(
    (sum, point) => sum + point.completedCount,
    0,
  );
  const averageRate =
    chartData.length === 0
      ? 0
      : Math.round(
          (chartData.reduce((sum, point) => sum + point.completionRate, 0) /
            chartData.length) *
            100,
        );
  const currentMonthSelected = isCurrentMonth(month, referenceDate);
  const referencePoint = getReferencePoint(month, chartData, referenceDate);
  const activeMetricLabel = currentMonthSelected
    ? "오늘 활성 목표"
    : `${Number(referencePoint?.date.slice(8) ?? "1")}일 활성 목표`;
  const goalListReferenceDate = currentMonthSelected
    ? currentDate(referenceDate)
    : monthStartDate(month);

  return {
    activeMetricGoalCount: referencePoint?.activeGoalCount ?? 0,
    activeMetricLabel,
    averageRate,
    chartData,
    dailyRecordGoalSlots: buildDailyRecordGoalSlots(goals),
    goalListReferenceDate,
    totalCompleted,
    visibleGoals: goals.filter((goal) =>
      isGoalVisibleInDisplay(goal, goalListReferenceDate),
    ),
  };
}
