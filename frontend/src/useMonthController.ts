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
import { checkKey, nextDate } from "./goalSlots";
import {
  validateGoalTitleDraft,
  validateNewGoalDraft,
} from "./goalFormValidation";
import { buildMockMonthView } from "./mockMonth";
import { buildMonthSummary } from "./monthSummary";
import {
  applyCheckState,
  applyGoalEndDateState,
  applyGoalTitleState,
  applyMemoState,
  currentDate,
  currentMonth,
  deactivationDateForGoal,
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
  const loadRequestIDRef = useRef(0);
  const preparingMonthRef = useRef(false);
  activeMonthRef.current = month;
  const currentDayKey = currentDate();

  const {
    activeMetricGoalCount,
    activeMetricLabel,
    averageRate,
    chartData,
    dailyRecordGoalSlots,
    goalListReferenceDate,
    totalCompleted,
    visibleGoals,
  } = useMemo(
    () => buildMonthSummary(monthView, new Date(`${currentDayKey}T12:00:00`)),
    [currentDayKey, monthView],
  );
  const isLoading = loadStatus === "loading";
  const canSaveChanges = loadStatus === "api";
  const isMutatingMonth =
    savingGoal ||
    savingGoalTitle ||
    preparingMonth ||
    deactivatingGoalIDs.length > 0;
  useEffect(() => {
    void loadMonth(currentMonth());
  }, []);

  function moveMonth(offset: number) {
    loadMonth(offsetMonth(month, offset));
  }

  async function loadMonth(nextMonth: string) {
    const requestID = loadRequestIDRef.current + 1;
    loadRequestIDRef.current = requestID;
    resetMonthLoadState(nextMonth);

    try {
      const nextMonthView = await getMonthView(nextMonth);
      if (loadRequestIDRef.current !== requestID) {
        return;
      }

      setMonthView(nextMonthView);
      setLoadStatus("api");
    } catch (error) {
      if (loadRequestIDRef.current !== requestID) {
        return;
      }

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
    const submittedLoadRequestID = loadRequestIDRef.current;
    preparingMonthRef.current = true;
    setPreparingMonth(true);
    clearSaveFeedback();
    setGoalFormOpen(false);
    cancelEditingGoal();

    try {
      const preparedView = await ensureMonth(submittedMonth);
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setMonthView(preparedView);
        setSaveMessage("목표를 이월했습니다.");
      }
    } catch {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setSaveFailure("목표 이월에 실패했습니다.");
      }
    } finally {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        preparingMonthRef.current = false;
        setPreparingMonth(false);
      }
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
    const submittedMonth = date.slice(0, 7);
    const submittedLoadRequestID = loadRequestIDRef.current;

    const completed = !checks.some(
      (check) => check.goalId === goalId && check.date === date,
    );

    clearSaveFeedback();
    setSavingChecks((currentKeys) => [...currentKeys, key]);
    setMonthView((currentView) =>
      applyCheckState(currentView, goalId, date, completed),
    );

    try {
      await setGoalCompleted(goalId, date, completed);
    } catch {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setMonthView((currentView) =>
          applyCheckState(currentView, goalId, date, !completed),
        );
        setSaveFailure("체크 저장에 실패했습니다.");
      }
    } finally {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setSavingChecks((currentKeys) =>
          currentKeys.filter((currentKey) => currentKey !== key),
        );
      }
    }
  }

  function updateMemo(date: string, memo: string) {
    setMonthView((currentView) => applyMemoState(currentView, date, memo));
  }

  async function saveMemoForDate(date: string, memo: string) {
    if (!canSaveChanges) {
      setSaveFailure("API 데이터에서만 메모를 저장할 수 있습니다.");
      return;
    }

    if (savingMemos.includes(date)) {
      return;
    }
    const submittedMonth = date.slice(0, 7);
    const submittedLoadRequestID = loadRequestIDRef.current;

    clearSaveFeedback();
    setSavingMemos((currentDates) => [...currentDates, date]);

    try {
      await saveMemo(date, memo);
    } catch {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setSaveFailure("메모 저장에 실패했습니다.");
      }
    } finally {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setSavingMemos((currentDates) =>
          currentDates.filter((currentDate) => currentDate !== date),
        );
      }
    }
  }

  async function submitNewGoal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const validation = validateNewGoalDraft({
      canSaveChanges,
      goals,
      isMutatingMonth,
      month,
      startDate: newGoalStartDate,
      title: newGoalTitle,
    });
    if (!validation.ok) {
      setSaveFailure(validation.message);
      return;
    }

    const submittedMonth = month;
    const submittedLoadRequestID = loadRequestIDRef.current;
    clearSaveFeedback();
    setSavingGoal(true);

    try {
      await createGoal(
        submittedMonth,
        validation.trimmedTitle,
        newGoalStartDate,
      );
    } catch {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setSaveFailure("목표 추가에 실패했습니다.");
        setSavingGoal(false);
      }
      return;
    }

    if (!isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
      return;
    }

    setNewGoalTitle("");
    setNewGoalStartDate(monthStartDate(submittedMonth));
    setGoalFormOpen(false);
    await refreshMonthView(
      submittedMonth,
      submittedLoadRequestID,
      "목표를 추가했지만 화면 갱신에 실패했습니다.",
    );
    if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
      setSavingGoal(false);
    }
  }

  function startEditingGoal(goal: Goal) {
    if (!canSaveChanges) {
      setSaveFailure("API 데이터에서만 목표를 수정할 수 있습니다.");
      return;
    }

    clearSaveFeedback();
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

    const submittedMonth = month;
    const submittedLoadRequestID = loadRequestIDRef.current;
    const validation = validateGoalTitleDraft({
      canSaveChanges,
      isMutatingMonth,
      title: editingGoalTitle,
      unavailableMessage: "API 데이터에서만 목표를 수정할 수 있습니다.",
    });
    if (!validation.ok) {
      setSaveFailure(validation.message);
      return;
    }

    clearSaveFeedback();
    setSavingGoalTitle(true);

    try {
      await updateGoalTitle(goalID, validation.trimmedTitle);
    } catch {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setSaveFailure("목표 수정에 실패했습니다.");
        setSavingGoalTitle(false);
      }
      return;
    }

    if (!isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
      return;
    }

    setMonthView((currentView) =>
      applyGoalTitleState(currentView, goalID, validation.trimmedTitle),
    );
    cancelEditingGoal();

    await refreshMonthView(
      submittedMonth,
      submittedLoadRequestID,
      "목표를 수정했지만 화면 갱신에 실패했습니다.",
    );
    if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
      setSavingGoalTitle(false);
    }
  }

  async function deactivateGoalFromMonth(goal: Goal) {
    if (!canSaveChanges) {
      setSaveFailure("API 데이터에서만 목표를 종료할 수 있습니다.");
      return;
    }

    const submittedMonth = month;
    const submittedLoadRequestID = loadRequestIDRef.current;
    const endDate = deactivationDateForGoal(goal, submittedMonth);
    if (goal.endDate !== null && goal.endDate <= goalListReferenceDate) {
      setSaveFailure("이미 종료된 목표입니다.");
      return;
    }

    if (!markGoalDeactivating(goal.id)) {
      return;
    }

    clearSaveFeedback();

    try {
      try {
        await deactivateGoal(goal.id, endDate);
      } catch {
        if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
          setSaveFailure("목표 종료에 실패했습니다.");
        }
        return;
      }

      if (!isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        return;
      }

      setMonthView((currentView) =>
        applyGoalEndDateState(currentView, goal.id, endDate),
      );
      const replacementStartDate = nextDate(endDate);
      if (replacementStartDate <= monthEndDate(submittedMonth)) {
        setNewGoalStartDate(replacementStartDate);
      }
      const refreshed = await refreshMonthView(
        submittedMonth,
        submittedLoadRequestID,
        "목표를 종료했지만 화면 갱신에 실패했습니다.",
      );
      if (
        refreshed &&
        isCurrentSaveContext(submittedMonth, submittedLoadRequestID)
      ) {
        setSaveMessage("목표를 종료했습니다.");
      }
    } finally {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        unmarkGoalDeactivating(goal.id);
      }
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

  function resetMonthLoadState(nextMonth: string) {
    setLoadStatus("loading");
    setLoadError(null);
    clearSaveFeedback();
    setSavingChecks([]);
    setSavingMemos([]);
    setSavingGoal(false);
    setGoalFormOpen(false);
    setNewGoalTitle("");
    setNewGoalStartDate(monthStartDate(nextMonth));
    cancelEditingGoal();
    setSavingGoalTitle(false);
    preparingMonthRef.current = false;
    setPreparingMonth(false);
    deactivatingGoalIDSetRef.current.clear();
    setDeactivatingGoalIDs([]);
    setMonthView(buildMockMonthView(nextMonth));
  }

  async function refreshMonthView(
    submittedMonth: string,
    submittedLoadRequestID: number,
    failureMessage: string,
  ) {
    try {
      const refreshedView = await getMonthView(submittedMonth);
      if (!isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        return false;
      }

      setMonthView(refreshedView);
      return true;
    } catch {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setSaveFailure(failureMessage);
      }
      return false;
    }
  }

  function clearSaveFeedback() {
    setSaveError(null);
    setSaveMessage(null);
  }

  function isCurrentSaveContext(
    submittedMonth: string,
    submittedLoadRequestID: number,
  ) {
    return (
      activeMonthRef.current === submittedMonth &&
      loadRequestIDRef.current === submittedLoadRequestID
    );
  }

  function setSaveFailure(message: string) {
    setSaveError(message);
    setSaveMessage(null);
  }

  return {
    activeMetricGoalCount,
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
    savingGoal,
    savingGoalTitle,
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
