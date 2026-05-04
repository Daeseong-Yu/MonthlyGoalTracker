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
