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
    <section className="panel-card panel-card-padded">
      <div className="panel-header panel-header--flush">
        <div>
          <p className="panel-kicker">Trend</p>
          <h2 className="panel-heading">
            {labels.heading}
          </h2>
        </div>
        <span className="accent-pill">
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
