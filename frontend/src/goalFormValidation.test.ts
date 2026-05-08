import { describe, expect, it } from "vitest";

import {
  validateGoalTitleDraft,
  validateNewGoalDraft,
} from "./goalFormValidation";
import type { Goal } from "./types";

describe("goal form validation", () => {
  it("accepts a valid new goal draft and trims its title", () => {
    const result = validateNewGoalDraft({
      canSaveChanges: true,
      goals: [],
      isMutatingMonth: false,
      month: "2026-05",
      startDate: "2026-05-03",
      title: "  Read docs  ",
    });

    expect(result).toEqual({ ok: true, trimmedTitle: "Read docs" });
  });

  it("rejects new goals outside the selected month", () => {
    const result = validateNewGoalDraft({
      canSaveChanges: true,
      goals: [],
      isMutatingMonth: false,
      month: "2026-05",
      startDate: "2026-06-01",
      title: "Read",
    });

    expect(result).toEqual({
      ok: false,
      message: "시작일은 선택한 월 안에서 골라 주세요.",
    });
  });

  it("rejects new goals when API data cannot be saved", () => {
    const result = validateNewGoalDraft({
      canSaveChanges: false,
      goals: [],
      isMutatingMonth: false,
      month: "2026-05",
      startDate: "2026-05-03",
      title: "Read",
    });

    expect(result).toEqual({
      ok: false,
      message: "API 데이터에서만 목표를 추가할 수 있습니다.",
    });
  });

  it("rejects goal drafts while another month mutation is in progress", () => {
    const result = validateNewGoalDraft({
      canSaveChanges: true,
      goals: [],
      isMutatingMonth: true,
      month: "2026-05",
      startDate: "2026-05-03",
      title: "Read",
    });

    expect(result).toEqual({
      ok: false,
      message: "다른 저장 작업이 끝난 뒤 다시 시도해 주세요.",
    });
  });

  it("rejects new goals when the daily active limit is full", () => {
    const result = validateNewGoalDraft({
      canSaveChanges: true,
      goals: Array.from({ length: 5 }, (_, index) =>
        goal(index + 1, `Goal ${index + 1}`, "2026-05-01", null),
      ),
      isMutatingMonth: false,
      month: "2026-05",
      startDate: "2026-05-03",
      title: "Extra goal",
    });

    expect(result).toEqual({
      ok: false,
      message: "할일은 날짜별로 최대 5개까지 등록할 수 있습니다.",
    });
  });

  it("validates reusable goal title state", () => {
    expect(
      validateGoalTitleDraft({
        canSaveChanges: false,
        isMutatingMonth: false,
        title: "Read",
        unavailableMessage: "API 데이터에서만 목표를 수정할 수 있습니다.",
      }),
    ).toEqual({
      ok: false,
      message: "API 데이터에서만 목표를 수정할 수 있습니다.",
    });

    expect(
      validateGoalTitleDraft({
        canSaveChanges: true,
        isMutatingMonth: false,
        title: "  ",
        unavailableMessage: "API 데이터에서만 목표를 수정할 수 있습니다.",
      }),
    ).toEqual({
      ok: false,
      message: "목표 제목을 입력해 주세요.",
    });
  });
});

function goal(
  id: number,
  title: string,
  startDate: string,
  endDate: string | null,
): Goal {
  return {
    endDate,
    id,
    startDate,
    title,
  };
}
