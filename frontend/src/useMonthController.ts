import { type FormEvent, useEffect, useMemo, useRef, useState } from "react";

import {
  createGoal,
  deactivateGoal,
  ensureMonth,
  getMonthView,
  saveMemo,
  setGoalCompleted,
  updateGoalTitle,
} from "./api";
import type { LoadStatus } from "./appDisplay";
import {
  activeGoalLimitReachedForNewGoal,
  buildDailyRecordGoalSlots,
  checkKey,
  maxActiveGoalsPerDay,
  nextDate,
} from "./goalSlots";
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

export function useMonthController() {
  const [monthView, setMonthView] = useState(() =>
    buildMockMonthView(currentMonth()),
  );
  const [loadStatus, setLoadStatus] = useState<LoadStatus>("loading");
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
    isGoalActiveInDisplay(goal, goalListReferenceDate),
  );
  const dailyRecordGoalSlots = useMemo(
    () => buildDailyRecordGoalSlots(goals),
    [goals],
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

    if (
      newGoalStartDate < monthStartDate(month) ||
      newGoalStartDate > monthEndDate(month)
    ) {
      setSaveFailure("시작일은 선택한 월 안에서 골라 주세요.");
      return;
    }

    if (activeGoalLimitReachedForNewGoal(goals, newGoalStartDate, month)) {
      setSaveFailure(
        `할일은 날짜별로 최대 ${maxActiveGoalsPerDay}개까지 등록할 수 있습니다.`,
      );
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

  function toggleGoalForm() {
    cancelEditingGoal();
    setGoalFormOpen((open) => !open);
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

  return {
    activeMetricGoalCount: referencePoint?.activeGoalCount ?? 0,
    activeMetricLabel,
    averageRate,
    canSaveChanges,
    chartData,
    checks,
    dailyRecordGoalSlots,
    days,
    deactivatingGoalIDs,
    editingGoalID,
    editingGoalTitle,
    goalFormOpen,
    goalListReferenceDate,
    goals,
    isLoading,
    isMutatingMonth,
    loadError,
    loadMonth,
    loadStatus,
    month,
    moveMonth,
    newGoalStartDate,
    newGoalTitle,
    prepareCurrentMonth,
    saveError,
    saveMemoForDate,
    saveMessage,
    savingChecks,
    savingMemos,
    setEditingGoalTitle,
    setNewGoalStartDate,
    setNewGoalTitle,
    startEditingGoal,
    submitGoalTitle,
    submitNewGoal,
    toggleCheck,
    toggleGoalForm,
    totalCompleted,
    updateMemo,
    visibleGoals,
    deactivateGoalFromMonth,
    cancelEditingGoal,
  };
}

function isGoalActiveInDisplay(goal: Goal, referenceDate: string) {
  return (
    goal.startDate <= referenceDate &&
    (goal.endDate === null || referenceDate < goal.endDate)
  );
}
