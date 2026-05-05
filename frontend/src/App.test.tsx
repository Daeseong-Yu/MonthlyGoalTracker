// @vitest-environment jsdom

import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, it, vi } from "vitest";

import App from "./App";
import { isGoalActiveOnDate, offsetMonth } from "./monthLogic";
import type { Goal, MonthView } from "./types";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

vi.mock("./DailyCompletionChart", async () => {
  const React = await import("react");

  return {
    default: ({ data }: { data: unknown[] }) =>
      React.createElement(
        "div",
        { "data-testid": "completion-chart" },
        `chart points: ${data.length}`,
      ),
  };
});

const roots: Root[] = [];

describe("App", () => {
  afterEach(async () => {
    await act(async () => {
      roots.forEach((root) => root.unmount());
    });
    roots.length = 0;
    document.body.innerHTML = "";
    vi.unstubAllGlobals();
  });

  it("renders API month data after loading", async () => {
    const fetchMock = stubFetch((input) =>
      jsonResponse(buildMonthView(monthFromRequest(input))),
    );

    renderApp();

    await waitForText("API 데이터");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(document.body.textContent).toContain("API walk");
    expect(hasInputValue("api memo")).toBe(true);
    expect(document.body.textContent).toContain("chart points: 2");
  });

  it("falls back to sample data and disables API-only actions when loading fails", async () => {
    stubFetch(() => errorResponse(500));

    renderApp();

    await waitForText("샘플 데이터");

    expect(document.body.textContent).toContain(
      "API 응답을 받지 못해 샘플 데이터를 표시합니다.",
    );
    expect(getButton("목표 이월").disabled).toBe(true);
    expect(getButton("목표 추가").disabled).toBe(true);
  });

  it("loads the previous month from the navigation control", async () => {
    const fetchMock = stubFetch((input) =>
      jsonResponse(buildMonthView(monthFromRequest(input))),
    );

    renderApp();
    await waitForText("API 데이터");

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);

    await act(async () => {
      getButton("이전 달").dispatchEvent(
        new MouseEvent("click", { bubbles: true }),
      );
    });

    await waitFor(() => fetchMock.mock.calls.length >= 2);

    expect(monthFromRequest(fetchMock.mock.calls[1][0])).toBe(
      offsetMonth(initialMonth, -1),
    );
  });

  it("shows feedback after carrying goals into the current month", async () => {
    const fetchMock = stubFetch((input) =>
      jsonResponse(buildMonthView(monthFromRequest(input))),
    );

    renderApp();
    await waitForText("API 데이터");

    await act(async () => {
      getButton("목표 이월").dispatchEvent(
        new MouseEvent("click", { bubbles: true }),
      );
    });

    await waitForText("목표를 이월했습니다.");

    expect(fetchMock.mock.calls.some(([input]) =>
      requestPath(input).endsWith("/ensure"),
    )).toBe(true);
  });

  it("shows feedback after ending a goal", async () => {
    let endedDate: string | null = null;
    const fetchMock = stubFetch((input, init) => {
      const path = requestPath(input);
      if (path.includes("/api/goals/1/deactivate")) {
        endedDate = bodyEndDate(init);
        return okResponse();
      }

      return jsonResponse(
        buildMonthView(monthFromRequest(input), {
          firstGoalEndDate: endedDate,
        }),
      );
    });

    renderApp();
    await waitForText("API 데이터");
    const loadedMonth = monthFromRequest(fetchMock.mock.calls[0][0]);

    await act(async () => {
      getButton("API walk 종료").dispatchEvent(
        new MouseEvent("click", { bubbles: true }),
      );
    });

    await waitForText("목표를 종료했습니다.");

    expect(fetchMock.mock.calls.some(([input]) =>
      requestPath(input).includes("/api/goals/1/deactivate"),
    )).toBe(true);
    expect(queryButton("API walk 종료")).toBeNull();
    expect(getButton(`${shortFirstDayLabel(loadedMonth)} API walk 완료`))
      .toBeTruthy();

    await act(async () => {
      getButton("목표 추가").dispatchEvent(
        new MouseEvent("click", { bubbles: true }),
      );
    });

    expect(getInput("새 목표 시작일").value).toBe(dateAfter(endedDate));
  });

  it("hides an already ended goal from cards while preserving table history", async () => {
    const fetchMock = stubFetch((input) =>
      jsonResponse(
        buildMonthView(monthFromRequest(input), {
          firstGoalEndDate: `${monthFromRequest(input)}-01`,
        }),
      ),
    );

    renderApp();
    await waitForText("API 데이터");
    const loadedMonth = monthFromRequest(fetchMock.mock.calls[0][0]);

    expect(queryButton("API walk 종료")).toBeNull();
    expect(getButton(`${shortFirstDayLabel(loadedMonth)} API walk 완료`))
      .toBeTruthy();

    expect(fetchMock.mock.calls.every(([input]) =>
      !requestPath(input).includes("/api/goals/1/deactivate"),
    )).toBe(true);
  });

  it("keeps the daily table compact and marks weekends", async () => {
    stubFetch(() =>
      jsonResponse(
        buildMonthView("2026-05", {
          goalCount: 5,
          includeSunday: true,
        }),
      ),
    );

    renderApp();
    await waitForText("API 데이터");

    expect(precedes(getHeading("목표"), getHeading("일별 완료 개수"))).toBe(
      true,
    );
    expect(tableHeaders()).toEqual([
      "날짜",
      "메모",
      "API walk",
      "API read",
      "API focus",
      "API strength",
      "API stretch",
      "완료",
    ]);
    expect(getDailyRecordTable().className).toContain("table-fixed");
    expect(getDailyRecordTable().className).toContain("min-w-[48rem]");
    expect(getButton("05.01 API stretch 완료")).toBeTruthy();
    expect(getWeekdayLabel("05.02").className).toContain("text-blue-600");
    expect(getWeekdayLabel("05.03").className).toContain("text-rose-600");
  });
});

function renderApp() {
  const container = document.createElement("div");
  document.body.append(container);

  const root = createRoot(container);
  roots.push(root);

  act(() => {
    root.render(<App />);
  });

  return container;
}

function stubFetch(
  handler: (input: RequestInfo | URL, init?: RequestInit) => Response,
) {
  const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (input, init) =>
    handler(input, init),
  );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function buildMonthView(
  month: string,
  options: {
    firstGoalEndDate?: string | null;
    goalCount?: number;
    includeSunday?: boolean;
  } = {},
): MonthView {
  const goals = buildGoals(month, options);
  const days = buildDays(month, goals, options.includeSunday === true);

  return {
    month,
    goals,
    days,
    checks: [{ goalId: 1, date: `${month}-01`, completed: true }],
    chart: days,
  };
}

function buildGoals(
  month: string,
  options: { firstGoalEndDate?: string | null; goalCount?: number },
) {
  const titles = [
    "API walk",
    "API read",
    "API focus",
    "API strength",
    "API stretch",
  ];
  const goalCount = options.goalCount ?? 2;

  return titles.slice(0, goalCount).map<Goal>((title, index) => ({
    id: index + 1,
    title,
    startDate: `${month}-${index === 1 ? "02" : "01"}`,
    endDate: index === 0 ? options.firstGoalEndDate ?? null : null,
  }));
}

function buildDays(month: string, goals: Goal[], includeSunday: boolean) {
  const dayNumbers = includeSunday ? [1, 2, 3] : [1, 2];

  return dayNumbers.map((dayNumber) => {
    const date = `${month}-${String(dayNumber).padStart(2, "0")}`;
    const activeGoalCount = goals.filter((goal) =>
      isGoalActiveOnDate(goal, date),
    ).length;
    const completedCount = dayNumber === 1 ? 1 : 0;

    return {
      date,
      memo: dayNumber === 1 ? "api memo" : "",
      activeGoalCount,
      completedCount,
      completionRate:
        activeGoalCount === 0 ? 0 : completedCount / activeGoalCount,
    };
  });
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

function okResponse() {
  return new Response(null, { status: 204 });
}

function monthFromRequest(input: RequestInfo | URL) {
  const path = requestPath(input);
  const match = /\/api\/months\/([^/]+)/.exec(path);

  if (!match) {
    throw new Error(`expected month API request, got ${path}`);
  }

  return decodeURIComponent(match[1]);
}

function shortFirstDayLabel(month: string) {
  return `${month.slice(5)}.01`;
}

function requestPath(input: RequestInfo | URL) {
  return typeof input === "string"
    ? input
    : input instanceof URL
      ? input.pathname
      : input.url;
}

async function waitForText(text: string) {
  await waitFor(() => document.body.textContent?.includes(text) === true);
}

async function waitFor(assertion: () => boolean) {
  const start = Date.now();

  while (Date.now() - start < 1000) {
    if (assertion()) {
      return;
    }

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 10));
    });
  }

  throw new Error("timed out waiting for app state");
}

function getButton(label: string) {
  const button = queryButton(label);

  if (!button) {
    throw new Error(`expected button with aria-label ${label}`);
  }

  return button;
}

function queryButton(label: string) {
  return document.querySelector<HTMLButtonElement>(
    `button[aria-label="${label}"]`,
  );
}

function getInput(label: string) {
  const input = document.querySelector<HTMLInputElement>(
    `input[aria-label="${label}"]`,
  );

  if (!input) {
    throw new Error(`expected input with aria-label ${label}`);
  }

  return input;
}

function getHeading(text: string) {
  const heading = Array.from(document.querySelectorAll("h2")).find(
    (item) => item.textContent === text,
  );

  if (!heading) {
    throw new Error(`expected heading ${text}`);
  }

  return heading;
}

function precedes(first: Element, second: Element) {
  return Boolean(
    first.compareDocumentPosition(second) & Node.DOCUMENT_POSITION_FOLLOWING,
  );
}

function getDailyRecordTable() {
  const table = document.querySelector("table");

  if (!table) {
    throw new Error("expected daily record table");
  }

  return table;
}

function tableHeaders() {
  return Array.from(getDailyRecordTable().querySelectorAll("thead th")).map(
    (header) => header.textContent?.trim() ?? "",
  );
}

function getWeekdayLabel(shortDay: string) {
  const dateCell = Array.from(
    getDailyRecordTable().querySelectorAll("tbody th"),
  ).find((cell) => cell.textContent?.includes(shortDay));

  if (!dateCell) {
    throw new Error(`expected date cell ${shortDay}`);
  }

  const weekdayLabel = dateCell.querySelector("span:nth-child(2)");

  if (!weekdayLabel) {
    throw new Error(`expected weekday label for ${shortDay}`);
  }

  return weekdayLabel;
}

function hasInputValue(value: string) {
  return Array.from(document.querySelectorAll("input")).some(
    (input) => input.value === value,
  );
}

function bodyEndDate(init: RequestInit | undefined) {
  if (typeof init?.body !== "string") {
    throw new Error("expected JSON request body");
  }

  const body = JSON.parse(init.body) as { endDate?: string };
  if (body.endDate === undefined) {
    throw new Error("expected endDate in request body");
  }

  return body.endDate;
}

function dateAfter(date: string | null) {
  if (date === null) {
    throw new Error("expected date");
  }

  const next = new Date(`${date}T00:00:00`);
  next.setDate(next.getDate() + 1);
  return `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(
    2,
    "0",
  )}-${String(next.getDate()).padStart(2, "0")}`;
}
