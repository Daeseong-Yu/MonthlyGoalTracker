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

export type AppLocale = "ko" | "en";

export type UserSession = {
  id: number;
  email: string;
  locale: AppLocale;
  createdAt: string;
};

export type BootstrapResponse = {
  authenticated: boolean;
  locale: AppLocale;
  user: UserSession | null;
  csrfToken?: string | null;
};

export type AuthResponse = {
  user: UserSession;
  csrfToken: string;
  locale: AppLocale;
};
