type MetricTone = "teal" | "amber" | "rose";

type MetricSummaryProps = {
  activeGoalCount: number;
  activeMetricLabel: string;
  averageRate: number;
  totalCompleted: number;
};

export default function MetricSummary({
  activeGoalCount,
  activeMetricLabel,
  averageRate,
  totalCompleted,
}: MetricSummaryProps) {
  return (
    <section className="grid gap-3 sm:grid-cols-3">
      <Metric label="이번 달 완료" value={`${totalCompleted}개`} tone="teal" />
      <Metric label="평균 달성률" value={`${averageRate}%`} tone="amber" />
      <Metric
        label={activeMetricLabel}
        value={`${activeGoalCount}개`}
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
