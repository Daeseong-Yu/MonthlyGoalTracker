import {
  Ban,
  CalendarDays,
  CalendarPlus,
  Check,
  ChevronLeft,
  ChevronRight,
  Pencil,
  Plus,
  X,
} from "lucide-react";
import {
  type FormEvent,
  lazy,
  Suspense,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  createGoal,
  deactivateGoal,
  ensureMonth,
  getMonthView,
  saveMemo,
  setGoalCompleted,
  updateGoalTitle,
} from "./api";
import ChartErrorBoundary from "./ChartErrorBoundary";
import { buildMockMonthView } from "./mockMonth";
import {
  applyCheckState,
  applyGoalEndDateState,
  applyGoalTitleState,
  buildChartData,
  currentDate,
  currentMonth,
  deactivationDateForGoal,
  getReferencePoint,
  isCurrentMonth,
  isGoalActiveOnDate,
  monthEndDate,
  monthStartDate,
  offsetMonth,
} from "./monthLogic";
import type { Goal } from "./types";

const DailyCompletionChart = lazy(() => import("./DailyCompletionChart"));

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
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const [savingChecks, setSavingChecks] = useState<string[]>([]);
  const [savingMemos, setSavingMemos] = useState<string[]>([]);
  const [goalFormOpen, setGoalFormOpen] = useState(false);
  const [newGoalTitle, setNewGoalTitle] = useState("");
  const [newGoalStartDate, setNewGoalStartDate] = useState(() =>
    monthStartDate(currentMonth()),
  );
  const [savingGoal, setSavingGoal] = useState(false);
  const [editingGoalID, setEditingGoalID] = useState<number | null>(null);
  const [editingGoalTitle, setEditingGoalTitle] = useState("");
  const [savingGoalTitle, setSavingGoalTitle] = useState(false);
  const [deactivatingGoalIDs, setDeactivatingGoalIDs] = useState<number[]>([]);
  const [preparingMonth, setPreparingMonth] = useState(false);
  const { checks, days, goals, month } = monthView;
  const activeMonthRef = useRef(month);
  const deactivatingGoalIDSetRef = useRef(new Set<number>());
  const preparingMonthRef = useRef(false);
  activeMonthRef.current = month;

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
  const canSaveChanges = loadStatus === "api";
  const isMutatingMonth =
    savingGoal ||
    savingGoalTitle ||
    preparingMonth ||
    deactivatingGoalIDs.length > 0;
  const goalListReferenceDate = isCurrentMonth(month)
    ? currentDate()
    : monthStartDate(month);
  const visibleGoals = goals.filter((goal) =>
    isGoalVisibleInGoalList(goal, goalListReferenceDate),
  );

  useEffect(() => {
    void loadMonth(currentMonth());
  }, []);

  function moveMonth(offset: number) {
    loadMonth(offsetMonth(month, offset));
  }

  async function loadMonth(nextMonth: string) {
    setLoadStatus("loading");
    setLoadError(null);
    setSaveError(null);
    setSaveMessage(null);
    setSavingChecks([]);
    setSavingMemos([]);
    setGoalFormOpen(false);
    setNewGoalTitle("");
    setNewGoalStartDate(monthStartDate(nextMonth));
    setEditingGoalID(null);
    setEditingGoalTitle("");
    setSavingGoalTitle(false);
    preparingMonthRef.current = false;
    setPreparingMonth(false);
    deactivatingGoalIDSetRef.current.clear();
    setDeactivatingGoalIDs([]);
    setMonthView(buildMockMonthView(nextMonth));

    try {
      setMonthView(await getMonthView(nextMonth));
      setLoadStatus("api");
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "unknown error");
      setLoadStatus("fallback");
    }
  }

  async function prepareCurrentMonth() {
    if (!canSaveChanges) {
      setSaveFailure("API 데이터에서만 목표를 이월할 수 있습니다.");
      return;
    }

    if (preparingMonthRef.current) {
      return;
    }

    const submittedMonth = month;
    preparingMonthRef.current = true;
    setPreparingMonth(true);
    setSaveError(null);
    setSaveMessage(null);
    setGoalFormOpen(false);
    cancelEditingGoal();

    try {
      const preparedView = await ensureMonth(submittedMonth);
      setMonthView((currentView) =>
        currentView.month === submittedMonth ? preparedView : currentView,
      );
      if (activeMonthRef.current === submittedMonth) {
        setSaveMessage("목표를 이월했습니다.");
      }
    } catch {
      if (activeMonthRef.current === submittedMonth) {
        setSaveFailure("목표 이월에 실패했습니다.");
      }
    } finally {
      preparingMonthRef.current = false;
      setPreparingMonth(false);
    }
  }

  async function toggleCheck(goalId: number, date: string) {
    const goal = goals.find((item) => item.id === goalId);
    if (!goal || !isGoalActiveOnDate(goal, date)) {
      return;
    }

    if (!canSaveChanges) {
      setSaveFailure("API 데이터에서만 체크를 저장할 수 있습니다.");
      return;
    }

    const key = checkKey(goalId, date);
    if (savingChecks.includes(key)) {
      return;
    }

    const completed = !checks.some(
      (check) => check.goalId === goalId && check.date === date,
    );

    setSaveError(null);
    setSaveMessage(null);
    setSavingChecks((currentKeys) => [...currentKeys, key]);
    setMonthView((currentView) =>
      applyCheckState(currentView, goalId, date, completed),
    );

    try {
      await setGoalCompleted(goalId, date, completed);
    } catch {
      setMonthView((currentView) =>
        applyCheckState(currentView, goalId, date, !completed),
      );
      setSaveFailure("체크 저장에 실패했습니다.");
    } finally {
      setSavingChecks((currentKeys) =>
        currentKeys.filter((currentKey) => currentKey !== key),
      );
    }
  }

  function updateMemo(date: string, memo: string) {
    setMonthView((currentView) => ({
      ...currentView,
      days: currentView.days.map((day) =>
        day.date === date ? { ...day, memo } : day,
      ),
    }));
  }

  async function saveMemoForDate(date: string, memo: string) {
    if (!canSaveChanges) {
      setSaveFailure("API 데이터에서만 메모를 저장할 수 있습니다.");
      return;
    }

    if (savingMemos.includes(date)) {
      return;
    }

    setSaveError(null);
    setSaveMessage(null);
    setSavingMemos((currentDates) => [...currentDates, date]);

    try {
      await saveMemo(date, memo);
    } catch {
      setSaveFailure("메모 저장에 실패했습니다.");
    } finally {
      setSavingMemos((currentDates) =>
        currentDates.filter((currentDate) => currentDate !== date),
      );
    }
  }

  async function submitNewGoal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!canSaveChanges) {
      setSaveFailure("API 데이터에서만 목표를 추가할 수 있습니다.");
      return;
    }

    if (isMutatingMonth) {
      setSaveFailure("다른 저장 작업이 끝난 뒤 다시 시도해 주세요.");
      return;
    }

    const trimmedTitle = newGoalTitle.trim();
    if (trimmedTitle === "") {
      setSaveFailure("목표 제목을 입력해 주세요.");
      return;
    }

    if (!newGoalStartDate.startsWith(`${month}-`)) {
      setSaveFailure("시작일은 선택한 월 안에서 골라 주세요.");
      return;
    }

    const submittedMonth = month;
    setSaveError(null);
    setSaveMessage(null);
    setSavingGoal(true);

    try {
      await createGoal(submittedMonth, trimmedTitle, newGoalStartDate);
    } catch {
      setSaveFailure("목표 추가에 실패했습니다.");
      setSavingGoal(false);
      return;
    }

    setNewGoalTitle("");
    setNewGoalStartDate(monthStartDate(submittedMonth));
    setGoalFormOpen(false);

    await refreshMonthView(
      submittedMonth,
      "목표를 추가했지만 화면 갱신에 실패했습니다.",
    );
    setSavingGoal(false);
  }

  function startEditingGoal(goal: Goal) {
    if (!canSaveChanges) {
      setSaveFailure("API 데이터에서만 목표를 수정할 수 있습니다.");
      return;
    }

    setSaveError(null);
    setSaveMessage(null);
    setGoalFormOpen(false);
    setEditingGoalID(goal.id);
    setEditingGoalTitle(goal.title);
  }

  function cancelEditingGoal() {
    setEditingGoalID(null);
    setEditingGoalTitle("");
  }

  async function submitGoalTitle(
    event: FormEvent<HTMLFormElement>,
    goalID: number,
  ) {
    event.preventDefault();

    if (!canSaveChanges) {
      setSaveFailure("API 데이터에서만 목표를 수정할 수 있습니다.");
      return;
    }

    if (isMutatingMonth) {
      setSaveFailure("다른 저장 작업이 끝난 뒤 다시 시도해 주세요.");
      return;
    }

    const trimmedTitle = editingGoalTitle.trim();
    const submittedMonth = month;
    if (trimmedTitle === "") {
      setSaveFailure("목표 제목을 입력해 주세요.");
      return;
    }

    setSaveError(null);
    setSaveMessage(null);
    setSavingGoalTitle(true);

    try {
      await updateGoalTitle(goalID, trimmedTitle);
    } catch {
      setSaveFailure("목표 수정에 실패했습니다.");
      setSavingGoalTitle(false);
      return;
    }

    setMonthView((currentView) =>
      currentView.month === submittedMonth
        ? applyGoalTitleState(currentView, goalID, trimmedTitle)
        : currentView,
    );
    cancelEditingGoal();

    await refreshMonthView(
      submittedMonth,
      "목표를 수정했지만 화면 갱신에 실패했습니다.",
    );
    setSavingGoalTitle(false);
  }

  async function deactivateGoalFromMonth(goal: Goal) {
    if (!canSaveChanges) {
      setSaveFailure("API 데이터에서만 목표를 종료할 수 있습니다.");
      return;
    }

    const submittedMonth = month;
    const endDate = deactivationDateForGoal(goal, submittedMonth);
    if (goal.endDate !== null && goal.endDate <= goalListReferenceDate) {
      setSaveFailure("이미 종료된 목표입니다.");
      return;
    }

    if (!markGoalDeactivating(goal.id)) {
      return;
    }

    setSaveError(null);
    setSaveMessage(null);

    try {
      try {
        await deactivateGoal(goal.id, endDate);
      } catch {
        if (activeMonthRef.current === submittedMonth) {
          setSaveFailure("목표 종료에 실패했습니다.");
        }
        return;
      }

      setMonthView((currentView) =>
        currentView.month === submittedMonth
          ? applyGoalEndDateState(currentView, goal.id, endDate)
          : currentView,
      );
      const replacementStartDate = nextDate(endDate);
      if (replacementStartDate <= monthEndDate(submittedMonth)) {
        setNewGoalStartDate(replacementStartDate);
      }
      const refreshed = await refreshMonthView(
        submittedMonth,
        "목표를 종료했지만 화면 갱신에 실패했습니다.",
      );
      if (refreshed && activeMonthRef.current === submittedMonth) {
        setSaveMessage("목표를 종료했습니다.");
      }
    } finally {
      unmarkGoalDeactivating(goal.id);
    }
  }

  function markGoalDeactivating(goalID: number) {
    if (deactivatingGoalIDSetRef.current.has(goalID)) {
      return false;
    }

    deactivatingGoalIDSetRef.current.add(goalID);
    setDeactivatingGoalIDs([...deactivatingGoalIDSetRef.current]);
    return true;
  }

  function unmarkGoalDeactivating(goalID: number) {
    deactivatingGoalIDSetRef.current.delete(goalID);
    setDeactivatingGoalIDs([...deactivatingGoalIDSetRef.current]);
  }

  async function refreshMonthView(
    submittedMonth: string,
    failureMessage: string,
  ) {
    try {
      const refreshedView = await getMonthView(submittedMonth);
      setMonthView((currentView) =>
        currentView.month === submittedMonth ? refreshedView : currentView,
      );
      return true;
    } catch {
      if (activeMonthRef.current === submittedMonth) {
        setSaveFailure(failureMessage);
      }
      return false;
    }
  }

  function setSaveFailure(message: string) {
    setSaveError(message);
    setSaveMessage(null);
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
              <p
                className="mt-2 text-xs font-medium text-amber-700"
                role="status"
              >
                API 응답을 받지 못해 샘플 데이터를 표시합니다.
              </p>
            ) : null}
            {saveError ? (
              <p className="mt-2 text-xs font-medium text-rose-700" role="alert">
                {saveError}
              </p>
            ) : null}
            {saveMessage ? (
              <p
                className="mt-2 text-xs font-medium text-teal-700"
                role="status"
              >
                {saveMessage}
              </p>
            ) : null}
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <button
              className="icon-button"
              type="button"
              aria-label="이전 달"
              title="이전 달"
              disabled={isLoading || isMutatingMonth}
              onClick={() => void moveMonth(-1)}
            >
              <ChevronLeft size={18} />
            </button>
            <label className="flex h-10 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm font-medium shadow-sm">
              <CalendarDays size={17} className="text-teal-700" />
              <input
                className="w-[8.5rem] bg-transparent text-sm outline-none"
                type="month"
                aria-label="기록할 월"
                value={month}
                disabled={isLoading || isMutatingMonth}
                onChange={(event) => void loadMonth(event.target.value)}
              />
            </label>
            <button
              className="icon-button"
              type="button"
              aria-label="다음 달"
              title="다음 달"
              disabled={isLoading || isMutatingMonth}
              onClick={() => void moveMonth(1)}
            >
              <ChevronRight size={18} />
            </button>
            <button
              className="icon-button"
              type="button"
              aria-label="목표 이월"
              title="목표 이월"
              disabled={!canSaveChanges || isMutatingMonth}
              onClick={() => void prepareCurrentMonth()}
            >
              <CalendarPlus size={18} />
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

        <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_24rem]">
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
              <table className="w-full min-w-[48rem] table-fixed border-collapse text-left text-sm">
                <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
                  <tr>
                    <th className="sticky left-0 z-10 w-20 bg-zinc-50 px-3 py-2 font-semibold">
                      날짜
                    </th>
                    <th className="w-56 px-2 py-2 font-semibold normal-case">
                      메모
                    </th>
                    {goals.map((goal) => (
                      <th
                        key={goal.id}
                        className="w-20 px-2 py-2 font-semibold normal-case text-zinc-600"
                      >
                        <span className="line-clamp-2">{goal.title}</span>
                      </th>
                    ))}
                    <th className="w-16 px-2 py-2 text-right font-semibold normal-case">
                      완료
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-100">
                  {days.map((day) => {
                    const point = chartData.find(
                      (item) => item.date === day.date,
                    );

                    return (
                      <tr key={day.date} className="hover:bg-[#f6fbf8]">
                        <th className="sticky left-0 z-10 bg-white px-3 py-2 font-medium text-zinc-800">
                          <span className="block">{shortDate(day.date)}</span>
                          <span className={weekdayClassName(day.date)}>
                            {weekday(day.date)}
                          </span>
                        </th>
                        <td className="px-2 py-2">
                          <input
                            className={memoInputClassName(
                              savingMemos.includes(day.date),
                            )}
                            aria-label={`${shortDate(day.date)} 메모`}
                            value={day.memo}
                            disabled={
                              !canSaveChanges ||
                              isMutatingMonth ||
                              savingMemos.includes(day.date)
                            }
                            onChange={(event) =>
                              updateMemo(day.date, event.target.value)
                            }
                            onBlur={(event) =>
                              void saveMemoForDate(day.date, event.target.value)
                            }
                          />
                        </td>
                        {goals.map((goal) => {
                          const active = isGoalActiveOnDate(goal, day.date);
                          const checked =
                            active &&
                            checks.some(
                              (check) =>
                                check.goalId === goal.id &&
                                check.date === day.date,
                            );
                          const saving = savingChecks.includes(
                            checkKey(goal.id, day.date),
                          );

                          return (
                            <td key={goal.id} className="px-2 py-2">
                              <button
                                className={
                                  active
                                    ? checked
                                      ? "check-button checked"
                                      : "check-button"
                                    : "check-button inactive"
                                }
                                type="button"
                                aria-label={`${shortDate(day.date)} ${goal.title} 완료`}
                                aria-pressed={checked}
                                disabled={
                                  !active ||
                                  !canSaveChanges ||
                                  saving ||
                                  isMutatingMonth
                                }
                                title={goal.title}
                                onClick={() =>
                                  void toggleCheck(goal.id, day.date)
                                }
                              >
                                {checked ? <Check size={16} /> : null}
                              </button>
                            </td>
                          );
                        })}
                        <td className="px-2 py-2 text-right text-xs font-semibold text-zinc-800">
                          {point?.completedCount ?? 0}/
                          {point?.activeGoalCount ?? 0}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </section>

          <aside className="space-y-6">
            <section className="rounded-lg border border-zinc-200 bg-white p-4 shadow-soft">
              <div className="mb-4 flex items-center justify-between gap-3">
                <h2 className="text-base font-semibold text-zinc-950">목표</h2>
                <button
                  className="icon-button"
                  type="button"
                  aria-label="목표 추가"
                  title="목표 추가"
                  disabled={!canSaveChanges || isMutatingMonth}
                  onClick={() => {
                    cancelEditingGoal();
                    setGoalFormOpen((open) => !open);
                  }}
                >
                  <Plus size={18} />
                </button>
              </div>
              {goalFormOpen ? (
                <form
                  className="mb-4 space-y-2 rounded-md border border-zinc-200 bg-zinc-50 p-3"
                  onSubmit={(event) => void submitNewGoal(event)}
                >
                  <input
                    className="h-9 w-full rounded-md border border-zinc-200 bg-white px-3 text-sm outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100"
                    aria-label="새 목표 제목"
                    value={newGoalTitle}
                    disabled={isMutatingMonth}
                    placeholder="새 목표"
                    onChange={(event) => setNewGoalTitle(event.target.value)}
                  />
                  <div className="flex gap-2">
                    <input
                      className="h-9 min-w-0 flex-1 rounded-md border border-zinc-200 bg-white px-3 text-sm outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100"
                      type="date"
                      aria-label="새 목표 시작일"
                      min={monthStartDate(month)}
                      max={monthEndDate(month)}
                      value={newGoalStartDate}
                      disabled={isMutatingMonth}
                      onChange={(event) =>
                        setNewGoalStartDate(event.target.value)
                      }
                    />
                    <button
                      className="icon-button"
                      type="submit"
                      aria-label="목표 저장"
                      title="목표 저장"
                      disabled={isMutatingMonth}
                    >
                      <Plus size={18} />
                    </button>
                  </div>
                </form>
              ) : null}
              <div className="space-y-3">
                {visibleGoals.length === 0 ? (
                  <p className="rounded-md border border-dashed border-zinc-200 bg-zinc-50 px-3 py-4 text-sm font-medium text-zinc-500">
                    진행 중인 목표가 없습니다.
                  </p>
                ) : null}
                {visibleGoals.map((goal) => {
                  const deactivationDate = deactivationDateForGoal(goal, month);
                  const alreadyDeactivated =
                    goal.endDate !== null &&
                    goal.endDate <= goalListReferenceDate;
                  const deactivating = deactivatingGoalIDs.includes(goal.id);

                  return (
                    <div
                      key={goal.id}
                      className="rounded-md border border-zinc-200 bg-zinc-50 p-3"
                    >
                      {editingGoalID === goal.id ? (
                        <form
                          className="space-y-2"
                          onSubmit={(event) =>
                            void submitGoalTitle(event, goal.id)
                          }
                        >
                          <input
                            className="h-9 w-full rounded-md border border-zinc-200 bg-white px-3 text-sm outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100"
                            aria-label={`${goal.title} 제목 수정`}
                            value={editingGoalTitle}
                            disabled={isMutatingMonth}
                            onChange={(event) =>
                              setEditingGoalTitle(event.target.value)
                            }
                          />
                          <div className="flex items-center justify-between gap-2">
                            <p className="min-w-0 truncate text-xs text-zinc-500">
                              {formatGoalPeriod(goal)}
                            </p>
                            <div className="flex shrink-0 gap-1">
                              <button
                                className="mini-icon-button"
                                type="submit"
                                aria-label={`${goal.title} 저장`}
                                title="목표 저장"
                                disabled={isMutatingMonth}
                              >
                                <Check size={15} />
                              </button>
                              <button
                                className="mini-icon-button"
                                type="button"
                                aria-label={`${goal.title} 수정 취소`}
                                title="수정 취소"
                                disabled={isMutatingMonth}
                                onClick={cancelEditingGoal}
                              >
                                <X size={15} />
                              </button>
                            </div>
                          </div>
                        </form>
                      ) : (
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
                              aria-label={`${goal.title} 수정`}
                              title="목표 수정"
                              disabled={!canSaveChanges || isMutatingMonth}
                              onClick={() => startEditingGoal(goal)}
                            >
                              <Pencil size={15} />
                            </button>
                            <button
                              className="mini-icon-button"
                              type="button"
                              aria-label={
                                alreadyDeactivated
                                  ? `${goal.title} 이미 종료됨`
                                  : `${goal.title} 종료`
                              }
                              title={
                                alreadyDeactivated
                                  ? "이미 종료됨"
                                  : `목표 종료 (${shortDate(deactivationDate)}까지 활성)`
                              }
                              disabled={
                                !canSaveChanges ||
                                isMutatingMonth ||
                                deactivating
                              }
                              onClick={() => void deactivateGoalFromMonth(goal)}
                            >
                              <Ban size={15} />
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </section>

            <section className="rounded-lg border border-zinc-200 bg-white p-4 shadow-soft">
              <div className="mb-4 flex items-center justify-between gap-3">
                <h2 className="text-base font-semibold text-zinc-950">
                  일별 완료 개수
                </h2>
                <span className="rounded-full bg-teal-50 px-3 py-1 text-xs font-medium text-teal-800">
                  {goals.length}개 목표
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
          </aside>
        </section>
      </div>
    </main>
  );
}

function checkKey(goalId: number, date: string) {
  return `${goalId}:${date}`;
}

function memoInputClassName(saving: boolean) {
  const base =
    "h-8 w-full rounded-md border bg-white px-2 text-xs outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100 disabled:cursor-not-allowed disabled:bg-zinc-100 disabled:text-zinc-400";

  if (saving) {
    return `${base} border-amber-300`;
  }

  return `${base} border-zinc-200`;
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
    <div className={`rounded-md border px-3 py-2.5 ${toneClass}`}>
      <p className="text-xs font-medium opacity-80">{label}</p>
      <p className="mt-1 text-2xl font-semibold leading-7 tracking-normal">
        {value}
      </p>
    </div>
  );
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

function weekdayClassName(date: string) {
  const day = new Date(`${date}T00:00:00`).getDay();
  const base = "text-xs font-normal";

  if (day === 6) {
    return `${base} text-blue-600`;
  }

  if (day === 0) {
    return `${base} text-rose-600`;
  }

  return `${base} text-zinc-500`;
}

function isGoalVisibleInGoalList(goal: Goal, referenceDate: string) {
  return goal.endDate === null || referenceDate < goal.endDate;
}

function nextDate(date: string) {
  const next = new Date(`${date}T00:00:00`);
  next.setDate(next.getDate() + 1);
  return `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(
    2,
    "0",
  )}-${String(next.getDate()).padStart(2, "0")}`;
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
