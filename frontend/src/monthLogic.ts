import type {
  ChartPointWithLabel,
  DayEntry,
  Goal,
  GoalCheck,
  MonthView,
} from "./types";

export function applyCheckState(
  view: MonthView,
  goalId: number,
  date: string,
  completed: boolean,
): MonthView {
  if (view.month !== date.slice(0, 7)) {
    return view;
  }

  if (!completed) {
    return {
      ...view,
      checks: view.checks.filter(
        (check) => !(check.goalId === goalId && check.date === date),
      ),
    };
  }

  const exists = view.checks.some(
    (check) => check.goalId === goalId && check.date === date,
  );
  if (exists) {
    return view;
  }

  return {
    ...view,
    checks: [...view.checks, { goalId, date, completed: true }],
  };
}

export function applyMemoState(
  view: MonthView,
  date: string,
  memo: string,
): MonthView {
  if (view.month !== date.slice(0, 7)) {
    return view;
  }

  return {
    ...view,
    days: view.days.map((day) => (day.date === date ? { ...day, memo } : day)),
  };
}

export function applyGoalTitleState(
  view: MonthView,
  goalId: number,
  title: string,
): MonthView {
  return {
    ...view,
    goals: view.goals.map((goal) =>
      goal.id === goalId ? { ...goal, title } : goal,
    ),
  };
}

export function applyGoalEndDateState(
  view: MonthView,
  goalId: number,
  endDate: string,
): MonthView {
  return {
    ...view,
    goals: view.goals.map((goal) =>
      goal.id === goalId ? { ...goal, endDate } : goal,
    ),
  };
}

export function buildChartData(
  days: DayEntry[],
  goals: Goal[],
  checks: GoalCheck[],
): ChartPointWithLabel[] {
  return days.map((day) => {
    const activeGoals = goals.filter((goal) =>
      isGoalActiveOnDate(goal, day.date),
    );
    const completedCount = activeGoals.filter((goal) =>
      checks.some(
        (check) => check.goalId === goal.id && check.date === day.date,
      ),
    ).length;
    const completionRate =
      activeGoals.length === 0 ? 0 : completedCount / activeGoals.length;

    return {
      date: day.date,
      dayLabel: String(new Date(`${day.date}T00:00:00`).getDate()),
      activeGoalCount: activeGoals.length,
      completedCount,
      completionRate,
    };
  });
}

export function isGoalActiveOnDate(goal: Goal, date: string) {
  if (goal.startDate > date) {
    return false;
  }

  return goal.endDate === null || date <= goal.endDate;
}

export function offsetMonth(month: string, offset: number) {
  const date = new Date(`${month}-01T00:00:00`);
  date.setMonth(date.getMonth() + offset);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
}

export function currentMonth(referenceDate = new Date()) {
  return `${referenceDate.getFullYear()}-${String(
    referenceDate.getMonth() + 1,
  ).padStart(2, "0")}`;
}

export function monthStartDate(month: string) {
  return `${month}-01`;
}

export function monthEndDate(month: string) {
  const [year, monthNumber] = month.split("-").map(Number);
  const endDate = new Date(year, monthNumber, 0).getDate();
  return `${month}-${String(endDate).padStart(2, "0")}`;
}

export function currentDate(referenceDate = new Date()) {
  return `${currentMonth(referenceDate)}-${String(
    referenceDate.getDate(),
  ).padStart(2, "0")}`;
}

export function isDateCheckable(date: string, referenceDate = new Date()) {
  return date <= currentDate(referenceDate);
}

export function deactivationDateForGoal(
  goal: Goal,
  month: string,
  referenceDate = new Date(),
) {
  const referenceDay = isCurrentMonth(month, referenceDate)
    ? currentDate(referenceDate)
    : monthStartDate(month);
  const monthEnd = monthEndDate(month);

  if (referenceDay < goal.startDate) {
    return goal.startDate;
  }

  if (referenceDay > monthEnd) {
    return monthEnd;
  }

  return referenceDay;
}

export function isCurrentMonth(month: string, referenceDate = new Date()) {
  return month === currentMonth(referenceDate);
}

export function getReferencePoint(
  month: string,
  chartData: ChartPointWithLabel[],
  referenceDate = new Date(),
) {
  if (chartData.length === 0) {
    return undefined;
  }

  if (isCurrentMonth(month, referenceDate)) {
    return (
      chartData.find((point) => point.date === currentDate(referenceDate)) ??
      chartData[0]
    );
  }

  return chartData[0];
}
