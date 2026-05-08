type MetricTone = "teal" | "amber" | "rose";

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
        tone="teal"
      />
      <Metric label={labels.averageRate} value={`${averageRate}%`} tone="amber" />
      <Metric
        label={activeMetricLabel}
        value={labels.goalValue(activeGoalCount)}
        tone="rose"
      />
    </section>
  );
}

function Metric({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone: MetricTone;
}) {
  const toneClass = {
    teal: "border-teal-200 bg-teal-50 text-teal-900",
    amber: "border-amber-200 bg-amber-50 text-amber-900",
    rose: "border-rose-200 bg-rose-50 text-rose-900",
  }[tone];

  return (
    <div className={`rounded-md border px-3 py-2 ${toneClass}`}>
      <p className="text-[0.7rem] font-medium leading-4 opacity-80">{label}</p>
      <p className="mt-0.5 text-xl font-semibold leading-6 tracking-normal">
        {value}
      </p>
    </div>
  );
}
