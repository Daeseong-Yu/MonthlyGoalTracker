// @vitest-environment jsdom

import { act } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import App from "./App";
import {
  blurInput,
  cleanupAppTest,
  clickButton,
  getButton,
  getDailyRecordTable,
  getDashboardLayout,
  getHeading,
  getInput,
  getWeekdayLabel,
  hasInputValue,
  precedes,
  queryButton,
  renderApp,
  setInputValue,
  stubFetch,
  tableHeaderCells,
  tableHeaders,
  waitFor,
  waitForText,
} from "./App.testHelpers";
import { nextDate } from "./goalSlots";
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

describe("App", () => {
  afterEach(async () => {
    await cleanupAppTest();
  });

  it("renders API month data after loading", async () => {
    const fetchMock = stubFetch((input) =>
      jsonResponse(buildMonthView(monthFromRequest(input))),
    );

    renderApp(<App />);

    await waitForText("API 데이터");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(document.body.textContent).toContain("API walk");
    expect(hasInputValue("api memo")).toBe(true);
    expect(getInput("05.01 메모").className).toContain("w-56");
    expect(getInput("05.01 메모").className).toContain("max-w-full");
    expect(document.body.textContent).toContain("chart points: 2");
  });

  it("falls back to sample data and disables API-only actions when loading fails", async () => {
    stubFetch(() => errorResponse(500));

    renderApp(<App />);

    await waitForText("샘플 데이터");

    expect(document.body.textContent).toContain(
      "API 응답을 받지 못해 샘플 데이터를 표시합니다.",
    );
    expect(getButton("목표 이월").disabled).toBe(true);
    expect(getButton("목표 추가").disabled).toBe(true);
  });

  it("reserves five daily goal columns when no goals exist", async () => {
    stubFetch((input) =>
      jsonResponse(
        buildMonthView(monthFromRequest(input), {
          goalCount: 0,
        }),
      ),
    );

    renderApp(<App />);
    await waitForText("API 데이터");

    expect(tableHeaders()).toEqual([
      "날짜",
      "메모",
      "",
      "",
      "",
      "",
      "",
      "완료",
    ]);
    expect(getDailyRecordTable().className).toContain("min-w-[48rem]");
  });

  it("does not expand past five daily goal columns", async () => {
    stubFetch((input) =>
      jsonResponse(
        buildMonthView(monthFromRequest(input), {
          goalCount: 6,
        }),
      ),
    );

    renderApp(<App />);
    await waitForText("API 데이터");

    expect(tableHeaders()).toEqual([
      "날짜",
      "메모",
      "API walk",
      "API focus",
      "API strength",
      "API stretch",
      "API journal",
      "완료",
    ]);
    expect(tableHeaders()).not.toContain("API read");
  });

  it("reuses a daily goal column for a later replacement goal", async () => {
    stubFetch((input) => {
      const month = monthFromRequest(input);
      const goals = buildGoals(month, {
        firstGoalEndDate: `${month}-01`,
        goalCount: 5,
      });
      const replacementGoal: Goal = {
        id: 6,
        title: "API journal",
        startDate: `${month}-02`,
        endDate: null,
      };

      return jsonResponse(buildMonthViewFromGoals(month, [
        replacementGoal,
        ...goals,
      ]));
    });

    renderApp(<App />);
    await waitForText("API 데이터");

    expect(tableHeaders()).toEqual([
      "날짜",
      "메모",
      "API walk / API read",
      "API focus",
      "API strength",
      "API stretch",
      "API journal",
      "완료",
    ]);
    expect(getButton("05.01 API walk 완료")).toBeTruthy();
    expect(getButton("05.02 API journal 완료")).toBeTruthy();
  });

  it("loads the previous month from the navigation control", async () => {
    const fetchMock = stubFetch((input) =>
      jsonResponse(buildMonthView(monthFromRequest(input))),
    );

    renderApp(<App />);
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

    renderApp(<App />);
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

    renderApp(<App />);
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

    renderApp(<App />);
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

    renderApp(<App />);
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

    renderApp(<App />);
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

    renderApp(<App />);
    await waitForText("API 데이터");

    expect(precedes(getHeading("목표"), getHeading("일별 완료 개수"))).toBe(
      true,
    );
    expect(tableHeaders()).toEqual([
      "날짜",
      "메모",
      "API walk",
      "API focus",
      "API strength",
      "API stretch",
      "API read",
      "완료",
    ]);
    expect(getHeading("날짜별 기록").parentElement?.textContent?.trim()).toBe(
      "날짜별 기록",
    );
    expect(getDashboardLayout().className).toContain(
      "xl:grid-cols-[minmax(0,1fr)_20.5rem]",
    );
    expect(getDailyRecordTable().className).toContain("table-fixed");
    expect(getDailyRecordTable().className).toContain("min-w-[48rem]");
    expect(tableHeaderCells()[2].className).toContain("pl-4");
    expect(tableHeaderCells()[7].className).toContain("pr-4");
    expect(getButton("05.01 API stretch 완료")).toBeTruthy();
    expect(getButton("05.01 API walk 완료")).toBeTruthy();
    expect(getWeekdayLabel("05.02").className).toContain("text-blue-600");
    expect(getWeekdayLabel("05.03").className).toContain("text-rose-600");
  });

  it("blocks adding a sixth active goal", async () => {
    const fetchMock = stubFetch((input) =>
      jsonResponse(
        buildMonthView(monthFromRequest(input), {
          goalCount: 5,
        }),
      ),
    );

    renderApp(<App />);
    await waitForText("API 데이터");

    await clickButton("목표 추가");
    await setInputValue("새 목표 제목", "API overflow");
    await clickButton("목표 저장");

    await waitForText("할일은 날짜별로 최대 5개까지 등록할 수 있습니다.");
    expect(fetchMock.mock.calls.every(([input]) =>
      !requestPath(input).endsWith("/goals"),
    )).toBe(true);
    expect(getInput("새 목표 제목").value).toBe("API overflow");
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

    renderApp(<App />);
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

    renderApp(<App />);
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

function buildMonthViewFromGoals(month: string, goals: Goal[]): MonthView {
  const days = buildDays(month, goals, false);

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

  return nextDate(date);
}
