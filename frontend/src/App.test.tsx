// @vitest-environment jsdom

import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, it, vi } from "vitest";

import App from "./App";
import { isGoalActiveOnDate, offsetMonth } from "./monthLogic";
import type { Goal, GoalCheck, MonthView } from "./types";

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

  it("clears stale success feedback when validation fails", async () => {
    stubFetch((input) => jsonResponse(buildMonthView(monthFromRequest(input))));

    renderApp();
    await waitForText("API 데이터");

    await clickButton("목표 이월");
    await waitForText("목표를 이월했습니다.");

    await clickButton("목표 추가");
    await clickButton("목표 저장");

    await waitForText("목표 제목을 입력해 주세요.");
    expect(document.body.textContent).not.toContain("목표를 이월했습니다.");
  });

  it("disables memo editing while a memo save is pending", async () => {
    let resolveMemoSave: (() => void) | null = null;
    stubFetch((input) => {
      const path = requestPath(input);

      if (path.includes("/api/memos/")) {
        return pendingResponse((resolve) => {
          resolveMemoSave = resolve;
        });
      }

      return jsonResponse(buildMonthView(monthFromRequest(input)));
    });

    renderApp();
    await waitForText("API 데이터");

    await setInputValue("05.01 메모", "saving memo");
    await blurInput("05.01 메모");

    await waitFor(() => getInput("05.01 메모").disabled);

    await act(async () => {
      resolveMemoSave?.();
    });

    await waitFor(() => !getInput("05.01 메모").disabled);
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
          goalCount: 6,
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
      "할일",
      "완료",
    ]);
    expect(getHeading("날짜별 기록").parentElement?.textContent?.trim()).toBe(
      "날짜별 기록",
    );
    expect(getDailyRecordTable().className).toContain("table-fixed");
    expect(getDailyRecordTable().className).not.toContain("min-w-");
    expect(getButton("05.01 API stretch 완료")).toBeTruthy();
    expect(getButton("05.02 API journal 완료")).toBeTruthy();
    expect(getButton("05.01 API walk 완료")).toBeTruthy();
    expect(getWeekdayLabel("05.02").className).toContain("text-blue-600");
    expect(getWeekdayLabel("05.03").className).toContain("text-rose-600");
  });

  it("handles the core goal and daily record workflow", async () => {
    let view: MonthView | null = null;
    let nextGoalID = 3;
    const fetchMock = stubFetch((input, init) => {
      const path = requestPath(input);

      if (path.endsWith("/goals")) {
        const body = requestBody<{ title: string; startDate: string }>(init);
        view = appendGoal(
          view ?? buildMonthView(monthFromGoalCreatePath(path)),
          nextGoalID,
          body.title,
          body.startDate,
        );
        nextGoalID += 1;
        return okResponse();
      }

      if (path === "/api/goals/1") {
        const body = requestBody<{ title: string }>(init);
        view = updateGoalInView(requiredView(view), 1, body.title);
        return okResponse();
      }

      if (path === "/api/checks") {
        const body = requestBody<{
          goalId: number;
          date: string;
          completed: boolean;
        }>(init);
        view = updateCheckInView(
          requiredView(view),
          body.goalId,
          body.date,
          body.completed,
        );
        return okResponse();
      }

      if (path.includes("/api/memos/")) {
        const body = requestBody<{ memo: string }>(init);
        view = updateMemoInView(
          requiredView(view),
          decodeURIComponent(path.split("/").pop() ?? ""),
          body.memo,
        );
        return okResponse();
      }

      const month = monthFromRequest(input);
      if (view === null || view.month !== month) {
        view = buildMonthView(month);
      }

      return jsonResponse(view);
    });

    renderApp();
    await waitForText("API 데이터");
    const loadedMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    const secondDay = `${loadedMonth}-02`;

    await clickButton("목표 추가");
    await setInputValue("새 목표 제목", "API plan");
    await clickButton("목표 저장");

    await waitForText("API plan");
    expect(fetchMock.mock.calls.some(([input, init]) => {
      if (!requestPath(input).endsWith("/goals")) {
        return false;
      }

      const body = requestBody<{ title: string; startDate: string }>(init);
      return body.title === "API plan" && body.startDate === `${loadedMonth}-01`;
    })).toBe(true);

    await clickButton("API walk 수정");
    await setInputValue("API walk 제목 수정", "API walk revised");
    await clickButton("API walk 저장");

    await waitForText("API walk revised");
    expect(fetchMock.mock.calls.some(([input, init]) => {
      if (requestPath(input) !== "/api/goals/1") {
        return false;
      }

      return requestBody<{ title: string }>(init).title === "API walk revised";
    })).toBe(true);

    await clickButton(`${shortDate(secondDay)} API read 완료`);

    await waitFor(() =>
      getButton(`${shortDate(secondDay)} API read 완료`).getAttribute(
        "aria-pressed",
      ) === "true",
    );
    expect(fetchMock.mock.calls.some(([input, init]) => {
      if (requestPath(input) !== "/api/checks") {
        return false;
      }

      const body = requestBody<{
        goalId: number;
        date: string;
        completed: boolean;
      }>(init);
      return body.goalId === 2 && body.date === secondDay && body.completed;
    })).toBe(true);

    await setInputValue(`${shortDate(secondDay)} 메모`, "follow-up");
    await blurInput(`${shortDate(secondDay)} 메모`);

    await waitFor(() =>
      fetchMock.mock.calls.some(([input]) =>
        requestPath(input).includes(`/api/memos/${secondDay}`),
      ),
    );
    expect(fetchMock.mock.calls.some(([input, init]) => {
      if (!requestPath(input).includes(`/api/memos/${secondDay}`)) {
        return false;
      }

      return requestBody<{ memo: string }>(init).memo === "follow-up";
    })).toBe(true);
  });

  it("keeps UI state predictable when workflow saves fail", async () => {
    const fetchMock = stubFetch((input) => {
      const path = requestPath(input);

      if (
        path.endsWith("/goals") ||
        path === "/api/goals/1" ||
        path === "/api/checks" ||
        path.includes("/api/memos/")
      ) {
        return errorResponse(500);
      }

      return jsonResponse(buildMonthView(monthFromRequest(input)));
    });

    renderApp();
    await waitForText("API 데이터");
    const loadedMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    const secondDay = `${loadedMonth}-02`;
    const secondDayReadButton = `${shortDate(secondDay)} API read 완료`;
    const secondDayMemoInput = `${shortDate(secondDay)} 메모`;

    await clickButton("목표 추가");
    await setInputValue("새 목표 제목", "API blocked");
    await clickButton("목표 저장");

    await waitForText("목표 추가에 실패했습니다.");
    expect(getInput("새 목표 제목").value).toBe("API blocked");
    expect(document.body.textContent).not.toContain("API blocked 저장");

    await clickButton("API walk 수정");
    await setInputValue("API walk 제목 수정", "API rejected");
    await clickButton("API walk 저장");

    await waitForText("목표 수정에 실패했습니다.");
    expect(getInput("API walk 제목 수정").value).toBe("API rejected");
    expect(document.body.textContent).not.toContain("API rejected");

    await clickButton(secondDayReadButton);

    await waitForText("체크 저장에 실패했습니다.");
    expect(getButton(secondDayReadButton).getAttribute("aria-pressed")).toBe(
      "false",
    );

    await setInputValue(secondDayMemoInput, "memo kept locally");
    await blurInput(secondDayMemoInput);

    await waitForText("메모 저장에 실패했습니다.");
    expect(getInput(secondDayMemoInput).value).toBe("memo kept locally");
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
  handler: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Response | Promise<Response>,
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
    "API journal",
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

function appendGoal(
  view: MonthView,
  id: number,
  title: string,
  startDate: string,
) {
  return {
    ...view,
    goals: [...view.goals, { id, title, startDate, endDate: null }],
  };
}

function updateGoalInView(view: MonthView, goalId: number, title: string) {
  return {
    ...view,
    goals: view.goals.map((goal) =>
      goal.id === goalId ? { ...goal, title } : goal,
    ),
  };
}

function updateCheckInView(
  view: MonthView,
  goalId: number,
  date: string,
  completed: boolean,
) {
  if (!completed) {
    return {
      ...view,
      checks: view.checks.filter(
        (check) => !(check.goalId === goalId && check.date === date),
      ),
    };
  }

  if (
    view.checks.some((check) => check.goalId === goalId && check.date === date)
  ) {
    return view;
  }

  const check: GoalCheck = { goalId, date, completed: true };

  return {
    ...view,
    checks: [...view.checks, check],
  };
}

function updateMemoInView(view: MonthView, date: string, memo: string) {
  return {
    ...view,
    days: view.days.map((day) =>
      day.date === date ? { ...day, memo } : day,
    ),
  };
}

function requiredView(view: MonthView | null) {
  if (view === null) {
    throw new Error("expected loaded month view");
  }

  return view;
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

function pendingResponse(captureResolve: (resolve: () => void) => void) {
  return new Promise<Response>((resolve) => {
    captureResolve(() => resolve(okResponse()));
  });
}

function monthFromRequest(input: RequestInfo | URL) {
  const path = requestPath(input);
  const match = /\/api\/months\/([^/]+)/.exec(path);

  if (!match) {
    throw new Error(`expected month API request, got ${path}`);
  }

  return decodeURIComponent(match[1]);
}

function monthFromGoalCreatePath(path: string) {
  const match = /\/api\/months\/([^/]+)\/goals$/.exec(path);

  if (!match) {
    throw new Error(`expected goal create API request, got ${path}`);
  }

  return decodeURIComponent(match[1]);
}

function shortFirstDayLabel(month: string) {
  return `${month.slice(5)}.01`;
}

function shortDate(date: string) {
  return date.slice(5).replace("-", ".");
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

async function clickButton(label: string) {
  await act(async () => {
    getButton(label).dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
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

async function setInputValue(label: string, value: string) {
  const input = getInput(label);
  const valueSetter = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype,
    "value",
  )?.set;

  await act(async () => {
    valueSetter?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

async function blurInput(label: string) {
  await act(async () => {
    getInput(label).dispatchEvent(new FocusEvent("focusout", { bubbles: true }));
  });
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
  const body = requestBody<{ endDate?: string }>(init);
  if (body.endDate === undefined) {
    throw new Error("expected endDate in request body");
  }

  return body.endDate;
}

function requestBody<T>(init: RequestInit | undefined) {
  if (typeof init?.body !== "string") {
    throw new Error("expected JSON request body");
  }

  return JSON.parse(init.body) as T;
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
