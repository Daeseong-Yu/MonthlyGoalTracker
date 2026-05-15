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
  messages?: GoalValidationMessages;
};

export type GoalTitleValidationInput = {
  canSaveChanges: boolean;
  isMutatingMonth: boolean;
  title: string;
  unavailableMessage: string;
  messages?: Pick<GoalValidationMessages, "busy" | "titleRequired">;
};

export type GoalValidationMessages = {
  addUnavailable: string;
  busy: string;
  titleRequired: string;
  startDateInMonth: string;
  activeGoalLimit: (limit: number) => string;
};

const defaultMessages: GoalValidationMessages = {
  addUnavailable: "계정 데이터에서만 목표를 추가할 수 있습니다.",
  busy: "다른 저장 작업이 끝난 뒤 다시 시도해 주세요.",
  titleRequired: "목표 제목을 입력해 주세요.",
  startDateInMonth: "시작일은 선택한 월 안에서 골라 주세요.",
  activeGoalLimit: (limit) =>
    `할일은 날짜별로 최대 ${limit}개까지 등록할 수 있습니다.`,
};

export function validateNewGoalDraft({
  canSaveChanges,
  goals,
  isMutatingMonth,
  month,
  startDate,
  title,
  messages = defaultMessages,
}: NewGoalValidationInput): ValidationResult {
  const baseValidation = validateGoalTitleDraft({
    canSaveChanges,
    isMutatingMonth,
    title,
    unavailableMessage: messages.addUnavailable,
    messages,
  });
  if (!baseValidation.ok) {
    return baseValidation;
  }

  if (startDate < monthStartDate(month) || startDate > monthEndDate(month)) {
    return { ok: false, message: messages.startDateInMonth };
  }

  if (activeGoalLimitReachedForNewGoal(goals, startDate, month)) {
    return {
      ok: false,
      message: messages.activeGoalLimit(maxActiveGoalsPerDay),
    };
  }

  return baseValidation;
}

export function validateGoalTitleDraft({
  canSaveChanges,
  isMutatingMonth,
  title,
  unavailableMessage,
  messages = defaultMessages,
}: GoalTitleValidationInput): ValidationResult {
  if (!canSaveChanges) {
    return { ok: false, message: unavailableMessage };
  }

  if (isMutatingMonth) {
    return {
      ok: false,
      message: messages.busy,
    };
  }

  const trimmedTitle = title.trim();
  if (trimmedTitle === "") {
    return { ok: false, message: messages.titleRequired };
  }

  return { ok: true, trimmedTitle };
}
