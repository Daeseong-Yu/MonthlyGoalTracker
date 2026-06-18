import { CheckCircle2, Gauge, Target } from "lucide-react";
import type { JSX } from "react";

type MetricTone = "sage" | "denim" | "mauve";

type MetricSummaryProps = {
  activeGoalCount: number;
  activeMetricLabel: string;
  averageRate: number;
  totalCompleted: number;
  labels?: {
    totalCompleted: string;
    averageRate: string;
    goalValue: (value: number) => string;
    completedValue: (value: number) => string;
  };
};

const defaultLabels = {
  totalCompleted: "이번 달 완료",
  averageRate: "평균 달성률",
  completedValue: (value: number) => `${value}개`,
  goalValue: (value: number) => `${value}개`,
};

export default function MetricSummary({
  activeGoalCount,
  activeMetricLabel,
  averageRate,
  labels = defaultLabels,
  totalCompleted,
}: MetricSummaryProps) {
  return (
    <section className="grid gap-3 sm:grid-cols-3">
      <Metric
        label={labels.totalCompleted}
        value={labels.completedValue(totalCompleted)}
        tone="sage"
        icon={<CheckCircle2 size={18} />}
      />
      <Metric
        icon={<Gauge size={18} />}
        label={labels.averageRate}
        value={`${averageRate}%`}
        tone="denim"
      />
      <Metric
        icon={<Target size={18} />}
        label={activeMetricLabel}
        value={labels.goalValue(activeGoalCount)}
        tone="mauve"
      />
    </section>
  );
}

function Metric({
  label,
  icon,
  value,
  tone,
}: {
  label: string;
  icon: JSX.Element;
  value: string;
  tone: MetricTone;
}) {
  const toneClass = {
    sage: "metric-card--sage",
    denim: "metric-card--denim",
    mauve: "metric-card--mauve",
  }[tone];

  return (
    <div className={`metric-card ${toneClass}`}>
      <div className="flex items-center gap-4">
        <span className="metric-icon" aria-hidden="true">
          {icon}
        </span>
        <div className="min-w-0">
          <p className="metric-label">{label}</p>
          <p className="metric-value mt-1 tracking-normal">{value}</p>
        </div>
      </div>
    </div>
  );
}
