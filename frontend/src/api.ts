import type { MonthView } from "./types";

const apiBaseURL = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");

export async function getMonthView(month: string): Promise<MonthView> {
  const response = await fetch(
    `${apiBaseURL}/api/months/${encodeURIComponent(month)}`,
    {
      headers: {
        Accept: "application/json",
      },
    },
  );

  if (!response.ok) {
    throw new Error(`month request failed with status ${response.status}`);
  }

  return (await response.json()) as MonthView;
}

export async function createGoal(
  month: string,
  title: string,
  startDate: string,
): Promise<void> {
  const response = await fetch(
    `${apiBaseURL}/api/months/${encodeURIComponent(month)}/goals`,
    {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ title, startDate }),
    },
  );

  if (!response.ok) {
    throw new Error(`goal request failed with status ${response.status}`);
  }
}

export async function updateGoalTitle(
  goalId: number,
  title: string,
): Promise<void> {
  const response = await fetch(`${apiBaseURL}/api/goals/${goalId}`, {
    method: "PATCH",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ title }),
  });

  if (!response.ok) {
    throw new Error(`goal update request failed with status ${response.status}`);
  }
}

export async function setGoalCompleted(
  goalId: number,
  date: string,
  completed: boolean,
): Promise<void> {
  const response = await fetch(`${apiBaseURL}/api/checks`, {
    method: "PUT",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ goalId, date, completed }),
  });

  if (!response.ok) {
    throw new Error(`check request failed with status ${response.status}`);
  }
}

export async function saveMemo(date: string, memo: string): Promise<void> {
  const response = await fetch(
    `${apiBaseURL}/api/memos/${encodeURIComponent(date)}`,
    {
      method: "PUT",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ memo }),
    },
  );

  if (!response.ok) {
    throw new Error(`memo request failed with status ${response.status}`);
  }
}
