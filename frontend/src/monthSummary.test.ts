import { describe, expect, it } from "vitest";

import { buildMonthSummary } from "./monthSummary";
import type { MonthView } from "./types";

const makeReferenceDate = (date: string) => new Date(`${date}T12:00:00`);

const makeMonthView = (month: string): MonthView => ({
  checks: [
    { completed: true, date: `${month}-01`, goalId: 1 },
    { completed: true, date: `${month}-02`, goalId: 1 },
    { completed: true, date: `${month}-02`, goalId: 2 },
  ],
  chart: [],
  days: [
    {
      activeGoalCount: 0,
      completedCount: 0,
      completionRate: 0,
      date: `${month}-01`,
      memo: "",
    },
    {
      activeGoalCount: 0,
      completedCount: 0,
      completionRate: 0,
      date: `${month}-02`,
      memo: "",
    },
    {
      activeGoalCount: 0,
      completedCount: 0,
      completionRate: 0,
      date: `${month}-03`,
      memo: "",
    },
  ],
  goals: [
    {
      endDate: null,
      id: 1,
      startDate: `${month}-01`,
      title: "Read",
    },
    {
      endDate: `${month}-03`,
      id: 2,
      startDate: `${month}-02`,
      title: "Walk",
    },
    {
      endDate: null,
      id: 3,
      startDate: `${month}-03`,
      title: "Write",
    },
  ],
  month,
});

describe("buildMonthSummary", () => {
  it("builds current-month metrics from the current reference date", () => {
    const summary = buildMonthSummary(
      makeMonthView("2026-05"),
      makeReferenceDate("2026-05-02"),
    );

    expect(summary.activeMetricGoalCount).toBe(2);
    expect(summary.activeMetricLabel).toBe("오늘 활성 목표");
    expect(summary.averageRate).toBe(67);
    expect(summary.goalListReferenceDate).toBe("2026-05-02");
    expect(summary.totalCompleted).toBe(3);
    expect(summary.chartData).toHaveLength(3);
    expect(summary.dailyRecordGoalSlots).toHaveLength(5);
    expect(summary.visibleGoals.map((goal) => goal.title)).toEqual([
      "Read",
      "Walk",
    ]);
  });

  it("uses the month start as the goal-list reference for past months", () => {
    const summary = buildMonthSummary(
      makeMonthView("2026-04"),
      makeReferenceDate("2026-05-02"),
    );

    expect(summary.activeMetricGoalCount).toBe(1);
    expect(summary.activeMetricLabel).toBe("1일 활성 목표");
    expect(summary.goalListReferenceDate).toBe("2026-04-01");
    expect(summary.visibleGoals.map((goal) => goal.title)).toEqual(["Read"]);
  });

  it("keeps ended goals out of cards while retaining daily table slots", () => {
    const summary = buildMonthSummary(
      {
        ...makeMonthView("2026-04"),
        goals: [
          {
            endDate: "2026-04-01",
            id: 1,
            startDate: "2026-04-01",
            title: "Ended",
          },
          {
            endDate: null,
            id: 2,
            startDate: "2026-04-01",
            title: "Active",
          },
        ],
      },
      makeReferenceDate("2026-05-02"),
    );

    expect(summary.visibleGoals.map((goal) => goal.title)).toEqual(["Active"]);
    expect(
      summary.dailyRecordGoalSlots.flatMap((slotGoals) =>
        slotGoals.map((goal) => goal.title),
      ),
    ).toEqual(["Ended", "Active"]);
  });
});
