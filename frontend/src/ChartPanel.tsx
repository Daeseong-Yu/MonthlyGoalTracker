import { lazy, Suspense } from "react";

import ChartErrorBoundary from "./ChartErrorBoundary";
import type { ChartPointWithLabel } from "./types";

const DailyCompletionChart = lazy(() => import("./DailyCompletionChart"));

type ChartPanelProps = {
  chartData: ChartPointWithLabel[];
  goalCount: number;
  labels?: {
    heading: string;
    loading: string;
    failed: string;
    completed: string;
    goalCount: (value: number) => string;
    completedValue: (value: string | number) => string;
  };
  month: string;
};

const defaultLabels = {
  heading: "일별 완료 개수",
  loading: "차트 불러오는 중",
  failed: "차트를 불러오지 못했습니다.",
  completed: "완료",
  goalCount: (value: number) => `${value}개 목표`,
  completedValue: (value: string | number) => `${value}개`,
};

export default function ChartPanel({
  chartData,
  goalCount,
  labels = defaultLabels,
  month,
}: ChartPanelProps) {
  return (
    <section className="rounded-lg border border-zinc-200 bg-white p-4 shadow-soft">
      <div className="mb-4 flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold text-zinc-950">
          {labels.heading}
        </h2>
        <span className="rounded-full bg-teal-50 px-3 py-1 text-xs font-medium text-teal-800">
          {labels.goalCount(goalCount)}
        </span>
      </div>
      <ChartErrorBoundary failedLabel={labels.failed} resetKey={month}>
        <Suspense
          fallback={
            <div
              className="h-72 min-h-72"
              role="status"
              aria-label={labels.loading}
            />
          }
        >
          <DailyCompletionChart data={chartData} labels={labels} />
        </Suspense>
      </ChartErrorBoundary>
    </section>
  );
}
