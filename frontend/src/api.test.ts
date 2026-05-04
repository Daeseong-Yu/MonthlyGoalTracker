import { afterEach, describe, expect, it, vi } from "vitest";

import {
  createGoal,
  deactivateGoal,
  ensureMonth,
  getMonthView,
  saveMemo,
  setGoalCompleted,
  updateGoalTitle,
} from "./api";
import type { MonthView } from "./types";

const monthView: MonthView = {
  month: "2026-05",
  goals: [],
  days: [],
  checks: [],
  chart: [],
};

describe("api client", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads a month view with an encoded month", async () => {
    const fetchMock = stubFetch(jsonResponse(monthView));

    await expect(getMonthView("2026 05")).resolves.toEqual(monthView);

    expect(fetchMock).toHaveBeenCalledWith("/api/months/2026%2005", {
      headers: {
        Accept: "application/json",
      },
    });
  });

  it("prepares a month with a POST request", async () => {
    const fetchMock = stubFetch(jsonResponse(monthView));

    await expect(ensureMonth("2026-05")).resolves.toEqual(monthView);

    expect(fetchMock).toHaveBeenCalledWith("/api/months/2026-05/ensure", {
      method: "POST",
      headers: {
        Accept: "application/json",
      },
    });
  });

  it("creates a goal with JSON body", async () => {
    const fetchMock = stubFetch(okResponse());

    await createGoal("2026-05", "Read", "2026-05-01");

    expect(fetchMock).toHaveBeenCalledWith("/api/months/2026-05/goals", {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify({ title: "Read", startDate: "2026-05-01" }),
    });
  });

  it("updates a goal title with PATCH", async () => {
    const fetchMock = stubFetch(okResponse());

    await updateGoalTitle(7, "Updated");

    expect(fetchMock).toHaveBeenCalledWith("/api/goals/7", {
      method: "PATCH",
      headers: jsonHeaders(),
      body: JSON.stringify({ title: "Updated" }),
    });
  });

  it("deactivates a goal with an end date", async () => {
    const fetchMock = stubFetch(okResponse());

    await deactivateGoal(7, "2026-05-03");

    expect(fetchMock).toHaveBeenCalledWith("/api/goals/7/deactivate", {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify({ endDate: "2026-05-03" }),
    });
  });

  it("saves a check with completed false", async () => {
    const fetchMock = stubFetch(okResponse());

    await setGoalCompleted(7, "2026-05-03", false);

    expect(fetchMock).toHaveBeenCalledWith("/api/checks", {
      method: "PUT",
      headers: jsonHeaders(),
      body: JSON.stringify({
        goalId: 7,
        date: "2026-05-03",
        completed: false,
      }),
    });
  });

  it("saves memo text with an encoded date", async () => {
    const fetchMock = stubFetch(okResponse());

    await saveMemo("2026 05 03", "memo");

    expect(fetchMock).toHaveBeenCalledWith("/api/memos/2026%2005%2003", {
      method: "PUT",
      headers: jsonHeaders(),
      body: JSON.stringify({ memo: "memo" }),
    });
  });

  it("throws when an API response is not ok", async () => {
    stubFetch(errorResponse(500));

    await expect(getMonthView("2026-05")).rejects.toThrow(
      "month request failed with status 500",
    );
  });
});

function stubFetch(response: Response) {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(response);
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function okResponse() {
  return new Response(null, { status: 204 });
}

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
    },
  });
}

function errorResponse(status: number) {
  return new Response(null, { status });
}

function jsonHeaders() {
  return {
    Accept: "application/json",
    "Content-Type": "application/json",
  };
}
