import { isGoalActiveOnDate } from "./monthLogic";
import type { MonthView } from "./types";

export function buildMockMonthView(month: string): MonthView {
  const dayCount = daysInMonth(month);
  const goals = buildGoals(month);
  const checks = buildChecks(month, dayCount);
  const days = buildDays(month, dayCount, goals, checks);

  return {
    month,
    goals,
    days,
    checks,
    chart: days.map(({ date, activeGoalCount, completedCount, completionRate }) => ({
      date,
      activeGoalCount,
      completedCount,
      completionRate,
    })),
  };
}

export function buildEmptyMonthView(month: string): MonthView {
  const days = Array.from({ length: daysInMonth(month) }, (_, index) => {
    const date = dateForDay(month, index + 1);

    return {
      date,
      memo: "",
      activeGoalCount: 0,
      completedCount: 0,
      completionRate: 0,
    };
  });

  return {
    month,
    goals: [],
    days,
    checks: [],
    chart: days.map(({ date, activeGoalCount, completedCount, completionRate }) => ({
      date,
      activeGoalCount,
      completedCount,
      completionRate,
    })),
  };
}

function buildDays(
  month: string,
  dayCount: number,
  goals: MonthView["goals"],
  checks: MonthView["checks"],
) {
  return Array.from({ length: dayCount }, (_, index) => {
    const day = index + 1;
    const date = dateForDay(month, day);
    const activeGoals = goals.filter((goal) => isGoalActiveOnDate(goal, date));
    const completedCount = activeGoals.filter((goal) =>
      checks.some((check) => check.goalId === goal.id && check.date === date),
    ).length;

    return {
      date,
      memo:
        day === 4
          ? "비가 와서 실내 운동으로 변경"
          : day === 12
            ? "저녁 산책 30분"
            : day === 21
              ? "컨디션 회복일"
              : "",
      activeGoalCount: activeGoals.length,
      completedCount,
      completionRate:
        activeGoals.length === 0 ? 0 : completedCount / activeGoals.length,
    };
  });
}

function buildGoals(month: string) {
  return [
    {
      id: 1,
      title: "아침 산책",
      startDate: dateForDay(month, 1),
      endDate: null,
    },
    {
      id: 2,
      title: "독서 20분",
      startDate: dateForDay(month, 1),
      endDate: null,
    },
    {
      id: 3,
      title: "물 6잔",
      startDate: dateForDay(month, 3),
      endDate: null,
    },
    {
      id: 4,
      title: "스트레칭",
      startDate: dateForDay(month, 1),
      endDate: dateForDay(month, 18),
    },
  ];
}

function buildChecks(month: string, dayCount: number) {
  const checkedGoalsByDay = new Map<number, number[]>([
    [1, [1, 2, 4]],
    [2, [1, 4]],
    [3, [1, 2, 3]],
    [4, [1, 2, 3, 4]],
    [5, [2, 3]],
    [6, [1, 2]],
    [7, [1, 4]],
    [8, [1, 2, 3, 4]],
    [9, [1, 3]],
    [10, [2]],
    [11, [1, 2, 3]],
    [12, [1, 4]],
  ]);

  return Array.from(checkedGoalsByDay.entries()).flatMap(([day, goalIds]) => {
    if (day > dayCount) {
      return [];
    }

    return goalIds.map((goalId) => ({
      goalId,
      date: dateForDay(month, day),
      completed: true as const,
    }));
  });
}

function daysInMonth(month: string) {
  const [year, monthNumber] = month.split("-").map(Number);
  return new Date(year, monthNumber, 0).getDate();
}

function dateForDay(month: string, day: number) {
  return `${month}-${String(day).padStart(2, "0")}`;
}
