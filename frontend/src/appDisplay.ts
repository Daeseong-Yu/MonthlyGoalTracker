import type { Goal } from "./types";

const monthFormatter = new Intl.DateTimeFormat("ko-KR", {
  year: "numeric",
  month: "long",
});

const weekdayFormatter = new Intl.DateTimeFormat("ko-KR", {
  weekday: "short",
});

export type LoadStatus = "loading" | "api" | "fallback";

export function formatMonth(month: string) {
  return monthFormatter.format(new Date(`${month}-01T00:00:00`));
}

export function formatGoalPeriod(goal: Goal) {
  const start = goal.startDate.slice(5).replace("-", ".");
  const end = goal.endDate?.slice(5).replace("-", ".") ?? "계속";
  return `${start} - ${end}`;
}

export function shortDate(date: string) {
  return date.slice(5).replace("-", ".");
}

export function weekday(date: string) {
  return weekdayFormatter.format(new Date(`${date}T00:00:00`));
}

export function weekdayClassName(date: string) {
  const day = new Date(`${date}T00:00:00`).getDay();
  const base = "text-xs font-normal";

  if (day === 6) {
    return `${base} text-blue-600`;
  }

  if (day === 0) {
    return `${base} text-rose-600`;
  }

  return `${base} text-zinc-500`;
}

export function memoInputClassName(saving: boolean) {
  const base =
    "h-8 w-56 max-w-full rounded-md border bg-white px-2 text-xs outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100 disabled:cursor-not-allowed disabled:bg-zinc-100 disabled:text-zinc-400";

  if (saving) {
    return `${base} border-amber-300`;
  }

  return `${base} border-zinc-200`;
}

export function statusLabel(status: LoadStatus) {
  if (status === "loading") {
    return "불러오는 중";
  }

  if (status === "api") {
    return "API 데이터";
  }

  return "샘플 데이터";
}

export function statusClassName(status: LoadStatus) {
  const base = "rounded-full px-2 py-0.5 text-xs font-semibold";
  if (status === "api") {
    return `${base} bg-teal-50 text-teal-800`;
  }

  if (status === "loading") {
    return `${base} bg-zinc-100 text-zinc-600`;
  }

  return `${base} bg-amber-50 text-amber-800`;
}
