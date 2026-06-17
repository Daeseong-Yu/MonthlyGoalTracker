import type { Goal } from "./types";

export type LoadStatus = "loading" | "api" | "fallback" | "local";
export type StatusLabels = Record<LoadStatus, string>;

const defaultStatusLabels: StatusLabels = {
  loading: "불러오는 중",
  api: "계정 데이터",
  fallback: "샘플 데이터",
  local: "둘러보기",
};

export function formatMonth(month: string, locale = "ko") {
  return monthFormatter(locale).format(new Date(`${month}-01T00:00:00`));
}

export function formatGoalPeriod(goal: Goal, continuedLabel = "계속") {
  const start = goal.startDate.slice(5).replace("-", ".");
  const end = goal.endDate?.slice(5).replace("-", ".") ?? continuedLabel;
  return `${start} - ${end}`;
}

export function shortDate(date: string) {
  return date.slice(5).replace("-", ".");
}

export function weekday(date: string, locale = "ko") {
  return weekdayFormatter(locale).format(new Date(`${date}T00:00:00`));
}

export function weekdayClassName(date: string) {
  const day = new Date(`${date}T00:00:00`).getDay();
  const base = "weekday-label text-xs font-normal";

  if (day === 6) {
    return `${base} weekend-sat text-blue-600`;
  }

  if (day === 0) {
    return `${base} weekend-sun text-rose-600`;
  }

  return `${base} text-zinc-500`;
}

export function memoInputClassName(saving: boolean) {
  const base =
    "field-control h-8 w-56 max-w-full rounded-md px-2 text-xs disabled:cursor-not-allowed";

  if (saving) {
    return `${base} border-[color:var(--warning-text)]`;
  }

  return base;
}

export function statusLabel(
  status: LoadStatus,
  labels: StatusLabels = defaultStatusLabels,
) {
  return labels[status];
}

export function statusClassName(status: LoadStatus) {
  const base = "status-pill";
  if (status === "api") {
    return `${base} status-pill--api`;
  }

  if (status === "loading") {
    return `${base} status-pill--loading`;
  }

  if (status === "local") {
    return `${base} status-pill--local`;
  }

  return `${base} status-pill--fallback`;
}

function intlLocale(locale: string) {
  return locale === "en" ? "en-US" : "ko-KR";
}

function monthFormatter(locale: string) {
  return new Intl.DateTimeFormat(intlLocale(locale), {
    year: "numeric",
    month: "long",
  });
}

function weekdayFormatter(locale: string) {
  return new Intl.DateTimeFormat(intlLocale(locale), {
    weekday: "short",
  });
}
