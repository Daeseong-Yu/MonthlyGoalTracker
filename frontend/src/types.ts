export type Goal = {
  id: number;
  title: string;
  startDate: string;
  endDate: string | null;
};

export type DayEntry = {
  date: string;
  memo: string;
  activeGoalCount: number;
  completedCount: number;
  completionRate: number;
};

export type GoalCheck = {
  goalId: number;
  date: string;
  completed: true;
};

export type ChartPoint = {
  date: string;
  activeGoalCount: number;
  completedCount: number;
  completionRate: number;
};

export type ChartPointWithLabel = ChartPoint & {
  dayLabel: string;
};

export type MonthView = {
  month: string;
  goals: Goal[];
  days: DayEntry[];
  checks: GoalCheck[];
  chart: ChartPoint[];
};
