import { lazy, Suspense } from "react";

import ChartErrorBoundary from "./ChartErrorBoundary";
import type { ChartPointWithLabel } from "./types";

const DailyCompletionChart = lazy(() => import("./DailyCompletionChart"));

type ChartPanelProps = {
  chartData: ChartPointWithLabel[];
  goalCount: number;
  month: string;
};

export default function ChartPanel({
  chartData,
  goalCount,
  month,
}: ChartPanelProps) {
  return (
    <section className="rounded-lg border border-zinc-200 bg-white p-4 shadow-soft">
      <div className="mb-4 flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold text-zinc-950">
          일별 완료 개수
        </h2>
        <span className="rounded-full bg-teal-50 px-3 py-1 text-xs font-medium text-teal-800">
          {goalCount}개 목표
        </span>
      </div>
      <ChartErrorBoundary resetKey={month}>
        <Suspense
          fallback={
            <div
              className="h-72 min-h-72"
              role="status"
              aria-label="차트 불러오는 중"
            />
          }
        >
          <DailyCompletionChart data={chartData} />
        </Suspense>
      </ChartErrorBoundary>
    </section>
  );
}
