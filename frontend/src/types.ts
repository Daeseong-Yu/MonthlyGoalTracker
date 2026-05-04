export type Goal = {
  id: number;
  title: string;
  startDate: string;
  endDate: string | null;
};

export type DayEntry = {
  date: string;
  memo: string;
};

export type GoalCheck = {
  goalId: number;
  date: string;
  completed: true;
};

export type ChartPoint = {
  date: string;
  dayLabel: string;
  activeGoalCount: number;
  completedCount: number;
  completionRate: number;
};

export type MonthView = {
  month: string;
  goals: Goal[];
  days: DayEntry[];
  checks: GoalCheck[];
};
