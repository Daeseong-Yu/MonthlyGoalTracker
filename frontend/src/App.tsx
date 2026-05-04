import {
  Ban,
  CalendarDays,
  Check,
  ChevronLeft,
  ChevronRight,
  Pencil,
  Plus,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { getMonthView } from "./api";
import { buildMockMonthView } from "./mockMonth";
import type { ChartPointWithLabel, DayEntry, Goal, GoalCheck } from "./types";

const monthFormatter = new Intl.DateTimeFormat("ko-KR", {
  year: "numeric",
  month: "long",
});

const weekdayFormatter = new Intl.DateTimeFormat("ko-KR", {
  weekday: "short",
});

export default function App() {
  const [monthView, setMonthView] = useState(() =>
    buildMockMonthView(currentMonth()),
  );
  const [loadStatus, setLoadStatus] = useState<"loading" | "api" | "fallback">(
    "loading",
  );
  const [loadError, setLoadError] = useState<string | null>(null);
  const { checks, days, goals, month } = monthView;

  const chartData = useMemo(
    () => buildChartData(days, goals, checks),
    [checks, days, goals],
  );

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
  const referencePoint = getReferencePoint(month, chartData);
  const activeMetricLabel = isCurrentMonth(month)
    ? "오늘 활성 목표"
    : `${Number(referencePoint?.date.slice(8) ?? "1")}일 활성 목표`;
  const isLoading = loadStatus === "loading";

  useEffect(() => {
    void loadMonth(currentMonth());
  }, []);

  function moveMonth(offset: number) {
    loadMonth(offsetMonth(month, offset));
  }

  async function loadMonth(nextMonth: string) {
    setLoadStatus("loading");
    setLoadError(null);
    setMonthView(buildMockMonthView(nextMonth));

    try {
      setMonthView(await getMonthView(nextMonth));
      setLoadStatus("api");
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "unknown error");
      setLoadStatus("fallback");
    }
  }

  function toggleCheck(goalId: number, date: string) {
    const goal = goals.find((item) => item.id === goalId);
    if (!goal || !isGoalActiveOnDate(goal, date)) {
      return;
    }

    setMonthView((currentView) => {
      const exists = currentView.checks.some(
        (check) => check.goalId === goalId && check.date === date,
      );

      if (exists) {
        return {
          ...currentView,
          checks: currentView.checks.filter(
            (check) => !(check.goalId === goalId && check.date === date),
          ),
        };
      }

      return {
        ...currentView,
        checks: [...currentView.checks, { goalId, date, completed: true }],
      };
    });
  }

  function updateMemo(date: string, memo: string) {
    setMonthView((currentView) => ({
      ...currentView,
      days: currentView.days.map((day) =>
        day.date === date ? { ...day, memo } : day,
      ),
    }));
  }

  return (
    <main className="min-h-screen bg-[#f7f8f5] text-zinc-900">
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 py-5 sm:px-6 lg:px-8">
        <header className="flex flex-col gap-4 border-b border-zinc-200 pb-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 className="text-2xl font-semibold tracking-normal text-zinc-950">
              월간 목표 트래커
            </h1>
            <p className="mt-1 flex flex-wrap items-center gap-2 text-sm text-zinc-600">
              <span>{formatMonth(month)} 기록</span>
              <span className={statusClassName(loadStatus)}>
                {statusLabel(loadStatus)}
              </span>
            </p>
            {loadError ? (
              <p className="mt-2 text-xs font-medium text-amber-700">
                API 응답을 받지 못해 샘플 데이터를 표시합니다.
              </p>
            ) : null}
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <button
              className="icon-button"
              type="button"
              title="이전 달"
              disabled={isLoading}
              onClick={() => void moveMonth(-1)}
            >
              <ChevronLeft size={18} />
            </button>
            <label className="flex h-10 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm font-medium shadow-sm">
              <CalendarDays size={17} className="text-teal-700" />
              <input
                className="w-[8.5rem] bg-transparent text-sm outline-none"
                type="month"
                value={month}
                disabled={isLoading}
                onChange={(event) => void loadMonth(event.target.value)}
              />
            </label>
            <button
              className="icon-button"
              type="button"
              title="다음 달"
              disabled={isLoading}
              onClick={() => void moveMonth(1)}
            >
              <ChevronRight size={18} />
            </button>
          </div>
        </header>

        <section className="grid gap-3 sm:grid-cols-3">
          <Metric label="이번 달 완료" value={`${totalCompleted}개`} tone="teal" />
          <Metric label="평균 달성률" value={`${averageRate}%`} tone="amber" />
          <Metric
            label={activeMetricLabel}
            value={`${referencePoint?.activeGoalCount ?? 0}개`}
            tone="rose"
          />
        </section>

        <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
          <div className="rounded-lg border border-zinc-200 bg-white p-4 shadow-soft">
            <div className="mb-4 flex items-center justify-between gap-3">
              <h2 className="text-base font-semibold text-zinc-950">
                일별 완료 개수
              </h2>
              <span className="rounded-full bg-teal-50 px-3 py-1 text-xs font-medium text-teal-800">
                {goals.length}개 목표
              </span>
            </div>
            <div className="h-72 min-h-72">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={chartData} margin={{ left: -20, right: 12 }}>
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
                        ? `${point.date} · ${Math.round(
                            point.completionRate * 100,
                          )}%`
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
          </div>

          <aside className="rounded-lg border border-zinc-200 bg-white p-4 shadow-soft">
            <div className="mb-4 flex items-center justify-between gap-3">
              <h2 className="text-base font-semibold text-zinc-950">목표</h2>
              <button className="icon-button" type="button" title="목표 추가">
                <Plus size={18} />
              </button>
            </div>
            <div className="space-y-3">
              {goals.map((goal) => (
                <div
                  key={goal.id}
                  className="rounded-md border border-zinc-200 bg-zinc-50 p-3"
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-semibold text-zinc-950">
                        {goal.title}
                      </p>
                      <p className="mt-1 text-xs text-zinc-500">
                        {formatGoalPeriod(goal)}
                      </p>
                    </div>
                    <div className="flex shrink-0 gap-1">
                      <button
                        className="mini-icon-button"
                        type="button"
                        title="목표 수정"
                      >
                        <Pencil size={15} />
                      </button>
                      <button
                        className="mini-icon-button"
                        type="button"
                        title="목표 비활성화"
                      >
                        <Ban size={15} />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </aside>
        </section>

        <section className="rounded-lg border border-zinc-200 bg-white shadow-soft">
          <div className="flex items-center justify-between gap-3 border-b border-zinc-200 px-4 py-3">
            <h2 className="text-base font-semibold text-zinc-950">
              날짜별 기록
            </h2>
            <span className="text-xs font-medium text-zinc-500">
              {days.length}일
            </span>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full min-w-[62rem] border-collapse text-left text-sm">
              <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
                <tr>
                  <th className="sticky left-0 z-10 w-28 bg-zinc-50 px-4 py-3 font-semibold">
                    날짜
                  </th>
                  {goals.map((goal) => (
                    <th
                      key={goal.id}
                      className="w-32 px-3 py-3 font-semibold normal-case text-zinc-600"
                    >
                      <span className="line-clamp-2">{goal.title}</span>
                    </th>
                  ))}
                  <th className="w-72 px-3 py-3 font-semibold normal-case">
                    메모
                  </th>
                  <th className="w-24 px-3 py-3 text-right font-semibold normal-case">
                    완료
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100">
                {days.map((day) => {
                  const point = chartData.find((item) => item.date === day.date);

                  return (
                    <tr key={day.date} className="hover:bg-[#f6fbf8]">
                      <th className="sticky left-0 z-10 bg-white px-4 py-3 font-medium text-zinc-800">
                        <span className="block">{shortDate(day.date)}</span>
                        <span className="text-xs font-normal text-zinc-500">
                          {weekday(day.date)}
                        </span>
                      </th>
                      {goals.map((goal) => {
                        const active = isGoalActiveOnDate(goal, day.date);
                        const checked = checks.some(
                          (check) =>
                            check.goalId === goal.id && check.date === day.date,
                        );

                        return (
                          <td key={goal.id} className="px-3 py-3">
                            <button
                              className={
                                active
                                  ? checked
                                    ? "check-button checked"
                                    : "check-button"
                                  : "check-button inactive"
                              }
                              type="button"
                              disabled={!active}
                              title={goal.title}
                              onClick={() => toggleCheck(goal.id, day.date)}
                            >
                              {checked ? <Check size={16} /> : null}
                            </button>
                          </td>
                        );
                      })}
                      <td className="px-3 py-3">
                        <input
                          className="h-9 w-full rounded-md border border-zinc-200 bg-white px-3 text-sm outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100"
                          value={day.memo}
                          onChange={(event) =>
                            updateMemo(day.date, event.target.value)
                          }
                        />
                      </td>
                      <td className="px-3 py-3 text-right font-semibold text-zinc-800">
                        {point?.completedCount ?? 0}/{point?.activeGoalCount ?? 0}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </main>
  );
}

function Metric({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone: "teal" | "amber" | "rose";
}) {
  const toneClass = {
    teal: "border-teal-200 bg-teal-50 text-teal-900",
    amber: "border-amber-200 bg-amber-50 text-amber-900",
    rose: "border-rose-200 bg-rose-50 text-rose-900",
  }[tone];

  return (
    <div className={`rounded-lg border p-4 ${toneClass}`}>
      <p className="text-sm font-medium opacity-80">{label}</p>
      <p className="mt-2 text-3xl font-semibold tracking-normal">{value}</p>
    </div>
  );
}

function buildChartData(
  days: DayEntry[],
  goals: Goal[],
  checks: GoalCheck[],
): ChartPointWithLabel[] {
  return days.map((day) => {
    const activeGoals = goals.filter((goal) =>
      isGoalActiveOnDate(goal, day.date),
    );
    const completedCount = activeGoals.filter((goal) =>
      checks.some(
        (check) => check.goalId === goal.id && check.date === day.date,
      ),
    ).length;
    const completionRate =
      activeGoals.length === 0 ? 0 : completedCount / activeGoals.length;

    return {
      date: day.date,
      dayLabel: String(new Date(`${day.date}T00:00:00`).getDate()),
      activeGoalCount: activeGoals.length,
      completedCount,
      completionRate,
    };
  });
}

function isGoalActiveOnDate(goal: Goal, date: string) {
  if (goal.startDate > date) {
    return false;
  }

  return goal.endDate === null || date <= goal.endDate;
}

function formatMonth(month: string) {
  return monthFormatter.format(new Date(`${month}-01T00:00:00`));
}

function formatGoalPeriod(goal: Goal) {
  const start = goal.startDate.slice(5).replace("-", ".");
  const end = goal.endDate?.slice(5).replace("-", ".") ?? "계속";
  return `${start} - ${end}`;
}

function shortDate(date: string) {
  return date.slice(5).replace("-", ".");
}

function weekday(date: string) {
  return weekdayFormatter.format(new Date(`${date}T00:00:00`));
}

function offsetMonth(month: string, offset: number) {
  const date = new Date(`${month}-01T00:00:00`);
  date.setMonth(date.getMonth() + offset);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
}

function currentMonth() {
  const today = new Date();
  return `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}`;
}

function currentDate() {
  const today = new Date();
  return `${currentMonth()}-${String(today.getDate()).padStart(2, "0")}`;
}

function isCurrentMonth(month: string) {
  return month === currentMonth();
}

function getReferencePoint(month: string, chartData: ChartPointWithLabel[]) {
  if (chartData.length === 0) {
    return undefined;
  }

  if (isCurrentMonth(month)) {
    return (
      chartData.find((point) => point.date === currentDate()) ?? chartData[0]
    );
  }

  return chartData[0];
}

function statusLabel(status: "loading" | "api" | "fallback") {
  if (status === "loading") {
    return "불러오는 중";
  }

  if (status === "api") {
    return "API 데이터";
  }

  return "샘플 데이터";
}

function statusClassName(status: "loading" | "api" | "fallback") {
  const base = "rounded-full px-2 py-0.5 text-xs font-semibold";
  if (status === "api") {
    return `${base} bg-teal-50 text-teal-800`;
  }

  if (status === "loading") {
    return `${base} bg-zinc-100 text-zinc-600`;
  }

  return `${base} bg-amber-50 text-amber-800`;
}
