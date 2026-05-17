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
import { messages as localizedMessages, type AppMessages } from "./i18n";
import { buildEmptyMonthView, buildMockMonthView } from "./mockMonth";
import { buildMonthSummary } from "./monthSummary";
import {
  applyCheckState,
  applyGoalEndDateState,
  applyGoalTitleState,
  applyMemoState,
  currentDate,
  currentMonth,
  deactivationDateForGoal,
  isDateCheckable,
  isGoalActiveOnDate,
  monthEndDate,
  monthStartDate,
  offsetMonth,
} from "./monthLogic";
import type { Goal, MonthView } from "./types";

export type MonthControllerMode = "server" | "preview";
type SaveFeedbackScope = "global" | "goal";

type UseMonthControllerOptions = {
  mode?: MonthControllerMode;
  messages?: Pick<AppMessages, "controller" | "summary" | "validation">;
};

export function useMonthController(options: UseMonthControllerOptions = {}) {
  const mode = options.mode ?? "server";
  const isPreviewMode = mode === "preview";
  const localeMessages = options.messages ?? localizedMessages.ko;
  const controllerMessages = localeMessages.controller;
  const summaryMessages = localeMessages.summary;
  const validationMessages = localeMessages.validation;
  const [monthView, setMonthView] = useState(() =>
    buildMonthViewForMode(currentMonth(), isPreviewMode),
  );
  const [loadStatus, setLoadStatus] = useState<LoadStatus>(() =>
    isPreviewMode ? "local" : "loading",
  );
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const [saveFeedbackScope, setSaveFeedbackScope] =
    useState<SaveFeedbackScope | null>(null);
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
    () =>
      buildMonthSummary(
        monthView,
        new Date(`${currentDayKey}T12:00:00`),
        summaryMessages,
      ),
    [currentDayKey, monthView, summaryMessages],
  );
  const isLoading = loadStatus === "loading";
  const canSaveChanges = isPreviewMode || loadStatus === "api";
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

    if (isPreviewMode) {
      setLoadStatus("local");
      return;
    }

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
      setSaveFailure(controllerMessages.prepareUnavailable);
      return;
    }

    if (isPreviewMode) {
      clearSaveFeedback();
      setGoalFormOpen(false);
      cancelEditingGoal();
      setSaveSuccess(controllerMessages.previewSaveNotice);
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
        setSaveSuccess(controllerMessages.prepareSuccess);
      }
    } catch {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setSaveFailure(controllerMessages.prepareFailure);
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
    if (!goal || !isGoalActiveOnDate(goal, date) || !isDateCheckable(date)) {
      return;
    }

    if (!canSaveChanges) {
      setSaveFailure(controllerMessages.checkUnavailable);
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

    if (isPreviewMode) {
      clearSaveFeedback();
      setMonthView((currentView) =>
        applyCheckState(currentView, goalId, date, completed),
      );
      setSaveSuccess(controllerMessages.previewSaveNotice);
      return;
    }

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
        setSaveFailure(controllerMessages.checkFailure);
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
      setSaveFailure(controllerMessages.memoUnavailable);
      return;
    }

    if (isPreviewMode) {
      clearSaveFeedback();
      setSaveSuccess(controllerMessages.previewSaveNotice);
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
        setSaveFailure(controllerMessages.memoFailure);
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
      messages: validationMessages,
    });
    if (!validation.ok) {
      setSaveFailure(validation.message, "goal");
      return;
    }

    const submittedMonth = month;
    const submittedLoadRequestID = loadRequestIDRef.current;
    clearSaveFeedback();

    if (isPreviewMode) {
      setMonthView((currentView) =>
        appendPreviewGoalState(
          currentView,
          validation.trimmedTitle,
          newGoalStartDate,
        ),
      );
      setNewGoalTitle("");
      setNewGoalStartDate(monthStartDate(submittedMonth));
      setGoalFormOpen(false);
      setSaveSuccess(controllerMessages.previewSaveNotice, "goal");
      return;
    }

    setSavingGoal(true);

    try {
      await createGoal(
        submittedMonth,
        validation.trimmedTitle,
        newGoalStartDate,
      );
    } catch {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setSaveFailure(controllerMessages.createFailure, "goal");
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
      controllerMessages.createRefreshFailure,
      "goal",
    );
    if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
      setSavingGoal(false);
    }
  }

  function startEditingGoal(goal: Goal) {
    if (!canSaveChanges) {
      setSaveFailure(controllerMessages.editUnavailable, "goal");
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
      unavailableMessage: controllerMessages.editUnavailable,
      messages: validationMessages,
    });
    if (!validation.ok) {
      setSaveFailure(validation.message, "goal");
      return;
    }

    if (isPreviewMode) {
      clearSaveFeedback();
      setMonthView((currentView) =>
        applyGoalTitleState(currentView, goalID, validation.trimmedTitle),
      );
      cancelEditingGoal();
      setSaveSuccess(controllerMessages.previewSaveNotice, "goal");
      return;
    }

    clearSaveFeedback();
    setSavingGoalTitle(true);

    try {
      await updateGoalTitle(goalID, validation.trimmedTitle);
    } catch {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setSaveFailure(controllerMessages.editFailure, "goal");
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
      controllerMessages.editRefreshFailure,
      "goal",
    );
    if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
      setSavingGoalTitle(false);
    }
  }

  async function deactivateGoalFromMonth(goal: Goal) {
    if (!canSaveChanges) {
      setSaveFailure(controllerMessages.deactivateUnavailable, "goal");
      return;
    }

    const submittedMonth = month;
    const submittedLoadRequestID = loadRequestIDRef.current;
    const endDate = deactivationDateForGoal(goal, submittedMonth);
    if (goal.endDate !== null && goal.endDate <= goalListReferenceDate) {
      setSaveFailure(controllerMessages.alreadyEnded, "goal");
      return;
    }

    if (isPreviewMode) {
      clearSaveFeedback();
      setMonthView((currentView) =>
        applyGoalEndDateState(currentView, goal.id, endDate),
      );
      const replacementStartDate = nextDate(endDate);
      if (replacementStartDate <= monthEndDate(submittedMonth)) {
        setNewGoalStartDate(replacementStartDate);
      }
      setSaveSuccess(controllerMessages.previewSaveNotice, "goal");
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
          setSaveFailure(controllerMessages.deactivateFailure, "goal");
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
        controllerMessages.deactivateRefreshFailure,
        "goal",
      );
      if (
        refreshed &&
        isCurrentSaveContext(submittedMonth, submittedLoadRequestID)
      ) {
        setSaveSuccess(controllerMessages.deactivateSuccess, "goal");
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
    setMonthView(buildMonthViewForMode(nextMonth, isPreviewMode));
  }

  async function refreshMonthView(
    submittedMonth: string,
    submittedLoadRequestID: number,
    failureMessage: string,
    failureScope: SaveFeedbackScope = "global",
  ) {
    if (isPreviewMode) {
      return true;
    }

    try {
      const refreshedView = await getMonthView(submittedMonth);
      if (!isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        return false;
      }

      setMonthView(refreshedView);
      return true;
    } catch {
      if (isCurrentSaveContext(submittedMonth, submittedLoadRequestID)) {
        setSaveFailure(failureMessage, failureScope);
      }
      return false;
    }
  }

  function clearSaveFeedback() {
    setSaveError(null);
    setSaveMessage(null);
    setSaveFeedbackScope(null);
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

  function setSaveFailure(message: string, scope: SaveFeedbackScope = "global") {
    setSaveError(message);
    setSaveMessage(null);
    setSaveFeedbackScope(scope);
  }

  function setSaveSuccess(message: string, scope: SaveFeedbackScope = "global") {
    setSaveError(null);
    setSaveMessage(message);
    setSaveFeedbackScope(scope);
  }

  return {
    activeMetricGoalCount,
    activeMetricLabel,
    averageRate,
    canSaveChanges,
    chartData,
    checkableThroughDate: currentDayKey,
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
    saveFeedbackScope,
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

function buildMonthViewForMode(month: string, isPreviewMode: boolean) {
  return isPreviewMode ? buildEmptyMonthView(month) : buildMockMonthView(month);
}

function appendPreviewGoalState(
  view: MonthView,
  title: string,
  startDate: string,
): MonthView {
  const goal: Goal = {
    id: nextPreviewGoalID(view.goals),
    title,
    startDate,
    endDate: null,
  };

  return {
    ...view,
    goals: [...view.goals, goal],
  };
}

function nextPreviewGoalID(goals: Goal[]) {
  return goals.reduce((maxID, goal) => Math.max(maxID, goal.id), 0) + 1;
}
