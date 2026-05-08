import {
  activeGoalLimitReachedForNewGoal,
  maxActiveGoalsPerDay,
} from "./goalSlots";
import { monthEndDate, monthStartDate } from "./monthLogic";
import type { Goal } from "./types";

type ValidationResult =
  | { ok: true; trimmedTitle: string }
  | { ok: false; message: string };

export type NewGoalValidationInput = {
  canSaveChanges: boolean;
  goals: Goal[];
  isMutatingMonth: boolean;
  month: string;
  startDate: string;
  title: string;
};

export type GoalTitleValidationInput = {
  canSaveChanges: boolean;
  isMutatingMonth: boolean;
  title: string;
  unavailableMessage: string;
};

export function validateNewGoalDraft({
  canSaveChanges,
  goals,
  isMutatingMonth,
  month,
  startDate,
  title,
}: NewGoalValidationInput): ValidationResult {
  const baseValidation = validateGoalTitleDraft({
    canSaveChanges,
    isMutatingMonth,
    title,
    unavailableMessage: "API 데이터에서만 목표를 추가할 수 있습니다.",
  });
  if (!baseValidation.ok) {
    return baseValidation;
  }

  if (startDate < monthStartDate(month) || startDate > monthEndDate(month)) {
    return { ok: false, message: "시작일은 선택한 월 안에서 골라 주세요." };
  }

  if (activeGoalLimitReachedForNewGoal(goals, startDate, month)) {
    return {
      ok: false,
      message: `할일은 날짜별로 최대 ${maxActiveGoalsPerDay}개까지 등록할 수 있습니다.`,
    };
  }

  return baseValidation;
}

export function validateGoalTitleDraft({
  canSaveChanges,
  isMutatingMonth,
  title,
  unavailableMessage,
}: GoalTitleValidationInput): ValidationResult {
  if (!canSaveChanges) {
    return { ok: false, message: unavailableMessage };
  }

  if (isMutatingMonth) {
    return {
      ok: false,
      message: "다른 저장 작업이 끝난 뒤 다시 시도해 주세요.",
    };
  }

  const trimmedTitle = title.trim();
  if (trimmedTitle === "") {
    return { ok: false, message: "목표 제목을 입력해 주세요." };
  }

  return { ok: true, trimmedTitle };
}
