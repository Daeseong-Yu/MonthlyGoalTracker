// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from "vitest";

import App from "./App";
import {
  bodyEndDate,
  buildGoals,
  buildMonthView,
  buildMonthViewFromGoals,
  createFailingWorkflowApiHandler,
  createMonthViewApiHandler,
  createWorkflowApiHandler,
  dateAfter,
  errorResponse,
  hasFetchedJsonBody,
  hasFetchedPath,
  hasNoFetchedPath,
  jsonResponse,
  monthFromRequest,
  okResponse,
  pendingResponse,
  requestPath,
  shortDate,
  shortFirstDayLabel,
} from "./App.testFixtures";
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
  resolvePending,
  setInputValue,
  stubFetch,
  tableHeaderCells,
  tableHeaders,
  waitFor,
  waitForText,
} from "./App.testHelpers";
import { offsetMonth } from "./monthLogic";
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

describe("App", () => {
  afterEach(async () => {
    await cleanupAppTest();
  });

  it("renders API month data after loading", async () => {
    const fetchMock = stubFetch(createMonthViewApiHandler());

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
    stubFetch(createMonthViewApiHandler({ goalCount: 0 }));

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
    stubFetch(createMonthViewApiHandler({ goalCount: 6 }));

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
    const fetchMock = stubFetch(createMonthViewApiHandler());

    renderApp(<App />);
    await waitForText("API 데이터");

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);

    await clickButton("이전 달");

    await waitFor(() => fetchMock.mock.calls.length >= 2);

    expect(monthFromRequest(fetchMock.mock.calls[1][0])).toBe(
      offsetMonth(initialMonth, -1),
    );
  });

  it("shows feedback after carrying goals into the current month", async () => {
    const fetchMock = stubFetch(createMonthViewApiHandler());

    renderApp(<App />);
    await waitForText("API 데이터");

    await clickButton("목표 이월");

    await waitForText("목표를 이월했습니다.");

    expect(hasFetchedPath(fetchMock, (path) => path.endsWith("/ensure"))).toBe(
      true,
    );
  });

  it("clears stale success feedback when validation fails", async () => {
    const fetchMock = stubFetch(createMonthViewApiHandler());

    renderApp(<App />);
    await waitForText("API 데이터");

    await clickButton("목표 이월");
    await waitForText("목표를 이월했습니다.");

    await clickButton("목표 추가");
    await clickButton("목표 저장");

    await waitForText("목표 제목을 입력해 주세요.");
    expect(document.body.textContent).not.toContain("목표를 이월했습니다.");
    expect(hasNoFetchedPath(fetchMock, (path) => path.endsWith("/goals"))).toBe(
      true,
    );
  });

  it("disables memo editing while a memo save is pending", async () => {
    let resolveMemoSave: (() => void) | null = null;
    const monthViewApiHandler = createMonthViewApiHandler();

    stubFetch((input) => {
      const path = requestPath(input);

      if (path.includes("/api/memos/")) {
        return pendingResponse((resolve) => {
          resolveMemoSave = resolve;
        });
      }

      return monthViewApiHandler(input);
    });

    renderApp(<App />);
    await waitForText("API 데이터");

    await setInputValue("05.01 메모", "saving memo");
    await blurInput("05.01 메모");

    await waitFor(() => getInput("05.01 메모").disabled);

    await resolvePending(resolveMemoSave);

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

    await clickButton("API walk 종료");

    await waitForText("목표를 종료했습니다.");

    expect(hasFetchedPath(fetchMock, (path) =>
      path.includes("/api/goals/1/deactivate"),
    )).toBe(true);
    expect(queryButton("API walk 종료")).toBeNull();
    expect(getButton(`${shortFirstDayLabel(loadedMonth)} API walk 완료`))
      .toBeTruthy();

    await clickButton("목표 추가");

    expect(getInput("새 목표 시작일").value).toBe(dateAfter(endedDate));
  });

  it("hides an already ended goal from cards while preserving table history", async () => {
    const fetchMock = stubFetch(
      createMonthViewApiHandler((month) => ({
        firstGoalEndDate: `${month}-01`,
      })),
    );

    renderApp(<App />);
    await waitForText("API 데이터");
    const loadedMonth = monthFromRequest(fetchMock.mock.calls[0][0]);

    expect(queryButton("API walk 종료")).toBeNull();
    expect(getButton(`${shortFirstDayLabel(loadedMonth)} API walk 완료`))
      .toBeTruthy();

    expect(hasNoFetchedPath(fetchMock, (path) =>
      path.includes("/api/goals/1/deactivate"),
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

    expect(precedes(getHeading("일별 완료 개수"), getHeading("목표"))).toBe(
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
    const fetchMock = stubFetch(createMonthViewApiHandler({ goalCount: 5 }));

    renderApp(<App />);
    await waitForText("API 데이터");

    await clickButton("목표 추가");
    await setInputValue("새 목표 제목", "API overflow");
    await clickButton("목표 저장");

    await waitForText("할일은 날짜별로 최대 5개까지 등록할 수 있습니다.");
    expect(hasNoFetchedPath(fetchMock, (path) => path.endsWith("/goals"))).toBe(
      true,
    );
    expect(getInput("새 목표 제목").value).toBe("API overflow");
  });

  it("blocks blank goal title edits before calling the API", async () => {
    const fetchMock = stubFetch(createMonthViewApiHandler());

    renderApp(<App />);
    await waitForText("API 데이터");

    await clickButton("API walk 수정");
    await setInputValue("API walk 제목 수정", "   ");
    await clickButton("API walk 저장");

    await waitForText("목표 제목을 입력해 주세요.");
    expect(hasNoFetchedPath(fetchMock, (path) => path === "/api/goals/1")).toBe(
      true,
    );
    expect(getInput("API walk 제목 수정").value).toBe("   ");
  });

  it("handles the core goal and daily record workflow", async () => {
    const fetchMock = stubFetch(createWorkflowApiHandler());

    renderApp(<App />);
    await waitForText("API 데이터");
    const loadedMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    const secondDay = `${loadedMonth}-02`;

    await clickButton("목표 추가");
    await setInputValue("새 목표 제목", "API plan");
    await clickButton("목표 저장");

    await waitForText("API plan");
    expect(hasFetchedJsonBody<{ title: string; startDate: string }>(
      fetchMock,
      (path) => path.endsWith("/goals"),
      (body) =>
        body.title === "API plan" && body.startDate === `${loadedMonth}-01`,
    )).toBe(true);

    await clickButton("API walk 수정");
    await setInputValue("API walk 제목 수정", "API walk revised");
    await clickButton("API walk 저장");

    await waitForText("API walk revised");
    expect(hasFetchedJsonBody<{ title: string }>(
      fetchMock,
      (path) => path === "/api/goals/1",
      (body) => body.title === "API walk revised",
    )).toBe(true);

    await clickButton(`${shortDate(secondDay)} API read 완료`);

    await waitFor(() =>
      getButton(`${shortDate(secondDay)} API read 완료`).getAttribute(
        "aria-pressed",
      ) === "true",
    );
    expect(hasFetchedJsonBody<{
      goalId: number;
      date: string;
      completed: boolean;
    }>(
      fetchMock,
      (path) => path === "/api/checks",
      (body) => body.goalId === 2 && body.date === secondDay && body.completed,
    )).toBe(true);

    await setInputValue(`${shortDate(secondDay)} 메모`, "follow-up");
    await blurInput(`${shortDate(secondDay)} 메모`);

    await waitFor(() =>
      hasFetchedPath(fetchMock, (path) =>
        path.includes(`/api/memos/${secondDay}`),
      ),
    );
    expect(hasFetchedJsonBody<{ memo: string }>(
      fetchMock,
      (path) => path.includes(`/api/memos/${secondDay}`),
      (body) => body.memo === "follow-up",
    )).toBe(true);
  });

  it("keeps UI state predictable when workflow saves fail", async () => {
    const fetchMock = stubFetch(createFailingWorkflowApiHandler());

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
