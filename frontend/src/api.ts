import type { MonthView } from "./types";

const apiBaseURL = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");

export async function getMonthView(month: string): Promise<MonthView> {
  return requestJSON<MonthView>(
    `/api/months/${encodeURIComponent(month)}`,
    "month request",
  );
}

export async function ensureMonth(month: string): Promise<MonthView> {
  return requestJSON<MonthView>(
    `/api/months/${encodeURIComponent(month)}/ensure`,
    "month ensure request",
    { method: "POST" },
  );
}

export async function createGoal(
  month: string,
  title: string,
  startDate: string,
): Promise<void> {
  await requestVoid(
    `/api/months/${encodeURIComponent(month)}/goals`,
    "goal request",
    { method: "POST", body: { title, startDate } },
  );
}

export async function updateGoalTitle(
  goalId: number,
  title: string,
): Promise<void> {
  await requestVoid(`/api/goals/${goalId}`, "goal update request", {
    method: "PATCH",
    body: { title },
  });
}

export async function deactivateGoal(
  goalId: number,
  endDate: string,
): Promise<void> {
  await requestVoid(`/api/goals/${goalId}/deactivate`, "goal deactivate request", {
    method: "POST",
    body: { endDate },
  });
}

export async function setGoalCompleted(
  goalId: number,
  date: string,
  completed: boolean,
): Promise<void> {
  await requestVoid("/api/checks", "check request", {
    method: "PUT",
    body: { goalId, date, completed },
  });
}

export async function saveMemo(date: string, memo: string): Promise<void> {
  await requestVoid(
    `/api/memos/${encodeURIComponent(date)}`,
    "memo request",
    { method: "PUT", body: { memo } },
  );
}

type RequestOptions = {
  body?: unknown;
  method?: string;
};

async function requestJSON<T>(
  path: string,
  errorLabel: string,
  options?: RequestOptions,
): Promise<T> {
  const response = await request(path, errorLabel, options);
  return (await response.json()) as T;
}

async function requestVoid(
  path: string,
  errorLabel: string,
  options?: RequestOptions,
): Promise<void> {
  await request(path, errorLabel, options);
}

async function request(
  path: string,
  errorLabel: string,
  options?: RequestOptions,
) {
  const response = await fetch(`${apiBaseURL}${path}`, requestInit(options));
  if (!response.ok) {
    throw new Error(`${errorLabel} failed with status ${response.status}`);
  }

  return response;
}

function requestInit(options?: RequestOptions): RequestInit {
  const init: RequestInit = {
    headers: {
      Accept: "application/json",
    },
  };

  if (options?.method) {
    init.method = options.method;
  }

  if (options && "body" in options) {
    init.headers = {
      Accept: "application/json",
      "Content-Type": "application/json",
    };
    init.body = JSON.stringify(options.body);
  }

  return init;
}
