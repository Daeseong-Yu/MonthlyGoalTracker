import { describe, expect, it } from "vitest";

import {
  activeGoalInSlot,
  activeGoalLimitReachedForNewGoal,
  buildDailyRecordGoalSlots,
  checkKey,
  goalSlotTitle,
  isGoalVisibleInDisplay,
  maxActiveGoalsPerDay,
  nextDate,
} from "./goalSlots";
import type { Goal } from "./types";

describe("goal slots", () => {
  it("builds fixed daily goal slots and reuses a slot after goals stop overlapping", () => {
    const slots = buildDailyRecordGoalSlots([
      goal(2, "Replacement", "2026-05-02", null),
      goal(1, "Original", "2026-05-01", "2026-05-01"),
      goal(3, "Focus", "2026-05-01", null),
    ]);

    expect(slots).toHaveLength(maxActiveGoalsPerDay);
    expect(goalSlotTitle(slots[0])).toBe("Original / Replacement");
    expect(goalSlotTitle(slots[1])).toBe("Focus");
    expect(activeGoalInSlot(slots[0], "2026-05-01")?.title).toBe("Original");
    expect(activeGoalInSlot(slots[0], "2026-05-02")?.title).toBe(
      "Replacement",
    );
  });

  it("detects whether a new goal would exceed the daily active limit", () => {
    const fullDayGoals = Array.from({ length: maxActiveGoalsPerDay }, (_, id) =>
      goal(id + 1, `Goal ${id + 1}`, "2026-05-01", null),
    );

    expect(
      activeGoalLimitReachedForNewGoal(
        fullDayGoals,
        "2026-05-10",
        "2026-05",
      ),
    ).toBe(true);
    expect(
      activeGoalLimitReachedForNewGoal(
        fullDayGoals.map((item, index) =>
          index === 0 ? { ...item, endDate: "2026-05-09" } : item,
        ),
        "2026-05-10",
        "2026-05",
      ),
    ).toBe(false);
  });

  it("includes end dates for goal card visibility", () => {
    const endingToday = goal(1, "Ending", "2026-05-01", "2026-05-03");
    const endedBefore = goal(2, "Ended", "2026-05-01", "2026-05-02");
    const futureEnd = goal(3, "Visible", "2026-05-01", "2026-05-04");

    expect(isGoalVisibleInDisplay(endingToday, "2026-05-03")).toBe(true);
    expect(isGoalVisibleInDisplay(endedBefore, "2026-05-03")).toBe(false);
    expect(isGoalVisibleInDisplay(futureEnd, "2026-05-03")).toBe(true);
    expect(isGoalVisibleInDisplay(futureEnd, "2026-04-30")).toBe(false);
  });

  it("formats check keys and advances dates across month boundaries", () => {
    expect(checkKey(7, "2026-05-03")).toBe("7:2026-05-03");
    expect(nextDate("2026-05-31")).toBe("2026-06-01");
  });
});

function goal(
  id: number,
  title: string,
  startDate: string,
  endDate: string | null,
): Goal {
  return {
    id,
    title,
    startDate,
    endDate,
  };
}
