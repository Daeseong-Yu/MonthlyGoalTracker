import { describe, expect, it } from "vitest";

import {
  applyCheckState,
  applyGoalEndDateState,
  applyGoalTitleState,
  applyMemoState,
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
import type { Goal, MonthView } from "./types";

const goals: Goal[] = [
  {
    id: 1,
    title: "Walk",
    startDate: "2026-05-01",
    endDate: null,
  },
  {
    id: 2,
    title: "Read",
    startDate: "2026-05-02",
    endDate: "2026-05-03",
  },
];

function monthView(overrides: Partial<MonthView> = {}): MonthView {
  return {
    month: "2026-05",
    goals,
    days: [
      {
        date: "2026-05-01",
        memo: "",
        activeGoalCount: 1,
        completedCount: 0,
        completionRate: 0,
      },
      {
        date: "2026-05-02",
        memo: "",
        activeGoalCount: 2,
        completedCount: 0,
        completionRate: 0,
      },
      {
        date: "2026-05-03",
        memo: "",
        activeGoalCount: 2,
        completedCount: 0,
        completionRate: 0,
      },
      {
        date: "2026-05-04",
        memo: "",
        activeGoalCount: 1,
        completedCount: 0,
        completionRate: 0,
      },
    ],
    checks: [],
    chart: [],
    ...overrides,
  };
}

describe("month logic", () => {
  it("adds and removes check state for the selected month", () => {
    const checked = applyCheckState(monthView(), 1, "2026-05-01", true);

    expect(checked.checks).toEqual([
      { goalId: 1, date: "2026-05-01", completed: true },
    ]);

    const unchecked = applyCheckState(checked, 1, "2026-05-01", false);

    expect(unchecked.checks).toEqual([]);
  });

  it("does not duplicate existing checks", () => {
    const view = monthView({
      checks: [{ goalId: 1, date: "2026-05-01", completed: true }],
    });

    expect(applyCheckState(view, 1, "2026-05-01", true)).toBe(view);
  });

  it("ignores check updates outside the month", () => {
    const view = monthView();

    expect(applyCheckState(view, 1, "2026-06-01", true)).toBe(view);
  });

  it("updates memo state for the selected month", () => {
    const updated = applyMemoState(monthView(), "2026-05-02", "Daily note");

    expect(updated.days.map((day) => day.memo)).toEqual([
      "",
      "Daily note",
      "",
      "",
    ]);
  });

  it("ignores memo updates outside the selected month", () => {
    const view = monthView();

    expect(applyMemoState(view, "2026-06-01", "Daily note")).toBe(view);
  });

  it("updates goal title and end date without changing other goals", () => {
    const titled = applyGoalTitleState(monthView(), 2, "Updated");
    const ended = applyGoalEndDateState(titled, 1, "2026-05-04");

    expect(ended.goals).toEqual([
      {
        id: 1,
        title: "Walk",
        startDate: "2026-05-01",
        endDate: "2026-05-04",
      },
      {
        id: 2,
        title: "Updated",
        startDate: "2026-05-02",
        endDate: "2026-05-03",
      },
    ]);
  });

  it("treats goal start and end dates as inclusive", () => {
    const goal = goals[1];

    expect(isGoalActiveOnDate(goal, "2026-05-01")).toBe(false);
    expect(isGoalActiveOnDate(goal, "2026-05-02")).toBe(true);
    expect(isGoalActiveOnDate(goal, "2026-05-03")).toBe(true);
    expect(isGoalActiveOnDate(goal, "2026-05-04")).toBe(false);
  });

  it("builds chart data from active goals and completed checks", () => {
    const chartData = buildChartData(monthView().days, goals, [
      { goalId: 1, date: "2026-05-02", completed: true },
      { goalId: 2, date: "2026-05-02", completed: true },
      { goalId: 2, date: "2026-05-04", completed: true },
    ]);

    expect(chartData).toEqual([
      {
        date: "2026-05-01",
        dayLabel: "1",
        activeGoalCount: 1,
        completedCount: 0,
        completionRate: 0,
      },
      {
        date: "2026-05-02",
        dayLabel: "2",
        activeGoalCount: 2,
        completedCount: 2,
        completionRate: 1,
      },
      {
        date: "2026-05-03",
        dayLabel: "3",
        activeGoalCount: 2,
        completedCount: 0,
        completionRate: 0,
      },
      {
        date: "2026-05-04",
        dayLabel: "4",
        activeGoalCount: 1,
        completedCount: 0,
        completionRate: 0,
      },
    ]);
  });

  it("handles month boundaries and leap years", () => {
    expect(monthStartDate("2026-05")).toBe("2026-05-01");
    expect(monthEndDate("2024-02")).toBe("2024-02-29");
    expect(monthEndDate("2025-02")).toBe("2025-02-28");
    expect(offsetMonth("2026-01", -1)).toBe("2025-12");
    expect(offsetMonth("2026-12", 1)).toBe("2027-01");
  });

  it("derives current date helpers from an injected date", () => {
    const referenceDate = new Date("2026-05-04T12:00:00");

    expect(currentMonth(referenceDate)).toBe("2026-05");
    expect(currentDate(referenceDate)).toBe("2026-05-04");
    expect(isCurrentMonth("2026-05", referenceDate)).toBe(true);
    expect(isCurrentMonth("2026-06", referenceDate)).toBe(false);
  });

  it("chooses a bounded deactivation date", () => {
    const previousMonthGoal: Goal = {
      id: 3,
      title: "Exercise",
      startDate: "2026-03-15",
      endDate: null,
    };

    expect(
      deactivationDateForGoal(goals[0], "2026-05", date("2026-05-04")),
    ).toBe("2026-05-04");
    expect(
      deactivationDateForGoal(goals[1], "2026-05", date("2026-05-01")),
    ).toBe("2026-05-02");
    expect(
      deactivationDateForGoal(
        previousMonthGoal,
        "2026-04",
        date("2026-05-04"),
      ),
    ).toBe("2026-04-01");
  });

  it("selects today's chart point for the current month", () => {
    const chartData = buildChartData(monthView().days, goals, []);

    expect(
      getReferencePoint("2026-05", chartData, date("2026-05-03")),
    ).toEqual(chartData[2]);
    expect(
      getReferencePoint("2026-05", chartData, date("2026-05-31")),
    ).toEqual(chartData[0]);
    expect(
      getReferencePoint("2026-04", chartData, date("2026-05-03")),
    ).toEqual(chartData[0]);
    expect(getReferencePoint("2026-05", [], date("2026-05-03"))).toBe(
      undefined,
    );
  });
});

function date(value: string) {
  return new Date(`${value}T12:00:00`);
}
