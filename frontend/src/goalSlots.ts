import { isGoalActiveOnDate, monthEndDate } from "./monthLogic";
import type { Goal } from "./types";

export const maxActiveGoalsPerDay = 5;

export function checkKey(goalId: number, date: string) {
  return `${goalId}:${date}`;
}

export function activeGoalLimitReachedForNewGoal(
  goals: Goal[],
  startDate: string,
  month: string,
) {
  const endDate = monthEndDate(month);

  for (let date = startDate; date <= endDate; date = nextDate(date)) {
    const activeGoalCount = goals.filter((goal) =>
      isGoalActiveOnDate(goal, date),
    ).length;

    if (activeGoalCount >= maxActiveGoalsPerDay) {
      return true;
    }
  }

  return false;
}

export function buildDailyRecordGoalSlots(goals: Goal[]) {
  const slots = Array.from({ length: maxActiveGoalsPerDay }, () => [] as Goal[]);
  const sortedGoals = [...goals].sort(
    (first, second) =>
      first.startDate.localeCompare(second.startDate) || first.id - second.id,
  );

  for (const goal of sortedGoals) {
    const availableSlot = slots.find((slotGoals) =>
      slotGoals.every((slotGoal) => !goalsOverlap(goal, slotGoal)),
    );

    if (availableSlot) {
      availableSlot.push(goal);
    }
  }

  return slots;
}

export function activeGoalInSlot(slotGoals: Goal[], date: string) {
  return slotGoals.find((goal) => isGoalActiveOnDate(goal, date)) ?? null;
}

export function goalSlotTitle(slotGoals: Goal[]) {
  return slotGoals.map((goal) => goal.title).join(" / ");
}

export function nextDate(date: string) {
  const next = new Date(`${date}T00:00:00`);
  next.setDate(next.getDate() + 1);
  return `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(
    2,
    "0",
  )}-${String(next.getDate()).padStart(2, "0")}`;
}

function goalsOverlap(first: Goal, second: Goal) {
  const firstEndDate = first.endDate ?? "9999-12-31";
  const secondEndDate = second.endDate ?? "9999-12-31";

  return first.startDate <= secondEndDate && second.startDate <= firstEndDate;
}
