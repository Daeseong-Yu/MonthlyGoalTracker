// @vitest-environment jsdom

import { afterEach, describe, expect, it } from "vitest";

import {
  buildMonthView,
  jsonResponse,
  monthFromRequest,
  pendingErrorResponse,
  pendingJsonResponse,
  pendingResponse,
  requestPath,
} from "./App.testFixtures";
import {
  cleanupAppTest,
  clickButton,
  renderApp,
  resolvePending,
  stubFetch,
  waitFor,
  waitForText,
} from "./App.testHelpers";
import { checkKey } from "./goalSlots";
import { offsetMonth } from "./monthLogic";
import { useMonthController } from "./useMonthController";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

describe("useMonthController", () => {
  afterEach(async () => {
    await cleanupAppTest();
  });

  it("keeps the latest loaded month when older responses finish late", async () => {
    const pendingLoads = new Map<string, () => void>();
    const fetchMock = stubFetch((input) => {
      const month = monthFromRequest(input);

      return pendingJsonResponse(buildLabeledMonthView(month), (resolve) => {
        pendingLoads.set(month, resolve);
      });
    });

    renderApp(<MonthControllerHarness />);
    await waitFor(() => fetchMock.mock.calls.length >= 1);

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    await resolvePending(pendingLoads.get(initialMonth) ?? null);
    await waitForText(`API ${initialMonth}`);

    const staleMonth = offsetMonth(initialMonth, -1);
    const latestMonth = offsetMonth(initialMonth, -2);

    await clickButton("이전 월 로드");
    await waitFor(() => pendingLoads.has(staleMonth));
    await waitForText(staleMonth);

    await clickButton("이전 월 로드");
    await waitFor(() => pendingLoads.has(latestMonth));

    await resolvePending(pendingLoads.get(latestMonth) ?? null);
    await waitForText(`API ${latestMonth}`);

    await resolvePending(pendingLoads.get(staleMonth) ?? null);

    expect(document.body.textContent).toContain(`API ${latestMonth}`);
    expect(document.body.textContent).not.toContain(`API ${staleMonth}`);
  });

  it("keeps the latest API state when older responses fail late", async () => {
    const pendingLoads = new Map<string, () => void>();
    let failingMonth: string | null = null;
    const fetchMock = stubFetch((input) => {
      const month = monthFromRequest(input);

      if (month === failingMonth) {
        return pendingErrorResponse(500, (resolve) => {
          pendingLoads.set(month, resolve);
        });
      }

      return pendingJsonResponse(buildLabeledMonthView(month), (resolve) => {
        pendingLoads.set(month, resolve);
      });
    });

    renderApp(<MonthControllerHarness />);
    await waitFor(() => fetchMock.mock.calls.length >= 1);

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    await resolvePending(pendingLoads.get(initialMonth) ?? null);
    await waitForText(`API ${initialMonth}`);

    const staleMonth = offsetMonth(initialMonth, -1);
    const latestMonth = offsetMonth(initialMonth, -2);
    failingMonth = staleMonth;

    await clickButton("이전 월 로드");
    await waitFor(() => pendingLoads.has(staleMonth));
    await waitForText(staleMonth);

    await clickButton("이전 월 로드");
    await waitFor(() => pendingLoads.has(latestMonth));

    await resolvePending(pendingLoads.get(latestMonth) ?? null);
    await waitForText(`API ${latestMonth}`);

    await resolvePending(pendingLoads.get(staleMonth) ?? null);

    expect(document.body.textContent).toContain("api");
    expect(document.body.textContent).toContain("no load error");
    expect(document.body.textContent).not.toContain("fallback");
  });

  it("ignores stale check save failures after loading another month", async () => {
    let resolveCheckSave: (() => void) | null = null;
    const fetchMock = stubFetch((input) => {
      if (requestPath(input) === "/api/checks") {
        return pendingErrorResponse(500, (resolve) => {
          resolveCheckSave = resolve;
        });
      }

      const month = monthFromRequest(input);
      return jsonResponse(buildLabeledMonthView(month));
    });

    renderApp(<MonthControllerHarness />);
    await waitFor(() => fetchMock.mock.calls.length >= 1);

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    await waitForText(`API ${initialMonth}`);

    await clickButton("첫 체크 토글");
    await waitFor(() => resolveCheckSave !== null);

    const nextVisibleMonth = offsetMonth(initialMonth, -1);
    await clickButton("이전 월 로드");
    await waitForText(`API ${nextVisibleMonth}`);

    await resolvePending(resolveCheckSave);

    expect(document.body.textContent).toContain(`API ${nextVisibleMonth}`);
    expect(document.body.textContent).toContain("no save error");
    expect(document.body.textContent).not.toContain("체크 저장에 실패했습니다.");
  });

  it("ignores stale check save failures after returning to the same month", async () => {
    const checkSaveResolves: Array<() => void> = [];
    const fetchMock = stubFetch((input) => {
      if (requestPath(input) === "/api/checks") {
        if (checkSaveResolves.length === 0) {
          return pendingErrorResponse(500, (resolve) => {
            checkSaveResolves.push(resolve);
          });
        }

        return pendingResponse((resolve) => {
          checkSaveResolves.push(resolve);
        });
      }

      const month = monthFromRequest(input);
      return jsonResponse(buildLabeledMonthView(month));
    });

    renderApp(<MonthControllerHarness />);
    await waitFor(() => fetchMock.mock.calls.length >= 1);

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    await waitForText(`API ${initialMonth}`);

    await clickButton("첫 체크 토글");
    await waitFor(() => checkSaveResolves.length >= 1);

    const previousMonth = offsetMonth(initialMonth, -1);
    await clickButton("이전 월 로드");
    await waitForText(`API ${previousMonth}`);

    await clickButton("다음 월 로드");
    await waitForText(`API ${initialMonth}`);

    await clickButton("첫 체크 토글");
    await waitFor(() => checkSaveResolves.length >= 2);
    await waitForText("first check incomplete");
    await waitForText("first check saving");

    await resolvePending(checkSaveResolves[0] ?? null);

    expect(document.body.textContent).toContain(`API ${initialMonth}`);
    expect(document.body.textContent).toContain("first check incomplete");
    expect(document.body.textContent).toContain("first check saving");
    expect(document.body.textContent).toContain("no save error");
    expect(document.body.textContent).not.toContain("체크 저장에 실패했습니다.");

    await resolvePending(checkSaveResolves[1] ?? null);
    await waitForText("first check idle");
  });

  it("ignores stale memo save failures after loading another month", async () => {
    let resolveMemoSave: (() => void) | null = null;
    const fetchMock = stubFetch((input) => {
      if (requestPath(input).includes("/api/memos/")) {
        return pendingErrorResponse(500, (resolve) => {
          resolveMemoSave = resolve;
        });
      }

      const month = monthFromRequest(input);
      return jsonResponse(buildLabeledMonthView(month));
    });

    renderApp(<MonthControllerHarness />);
    await waitFor(() => fetchMock.mock.calls.length >= 1);

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    await waitForText(`API ${initialMonth}`);

    await clickButton("첫 메모 저장");
    await waitFor(() => resolveMemoSave !== null);

    const nextVisibleMonth = offsetMonth(initialMonth, -1);
    await clickButton("이전 월 로드");
    await waitForText(`API ${nextVisibleMonth}`);

    await resolvePending(resolveMemoSave);

    expect(document.body.textContent).toContain(`API ${nextVisibleMonth}`);
    expect(document.body.textContent).toContain("no save error");
    expect(document.body.textContent).not.toContain("메모 저장에 실패했습니다.");
  });

  it("ignores stale memo save failures after returning to the same month", async () => {
    const memoSaveResolves: Array<() => void> = [];
    const fetchMock = stubFetch((input) => {
      if (requestPath(input).includes("/api/memos/")) {
        if (memoSaveResolves.length === 0) {
          return pendingErrorResponse(500, (resolve) => {
            memoSaveResolves.push(resolve);
          });
        }

        return pendingResponse((resolve) => {
          memoSaveResolves.push(resolve);
        });
      }

      const month = monthFromRequest(input);
      return jsonResponse(buildLabeledMonthView(month));
    });

    renderApp(<MonthControllerHarness />);
    await waitFor(() => fetchMock.mock.calls.length >= 1);

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    await waitForText(`API ${initialMonth}`);

    await clickButton("첫 메모 저장");
    await waitFor(() => memoSaveResolves.length >= 1);

    const previousMonth = offsetMonth(initialMonth, -1);
    await clickButton("이전 월 로드");
    await waitForText(`API ${previousMonth}`);

    await clickButton("다음 월 로드");
    await waitForText(`API ${initialMonth}`);

    await clickButton("첫 메모 저장");
    await waitFor(() => memoSaveResolves.length >= 2);
    await waitForText("first memo saving");

    await resolvePending(memoSaveResolves[0] ?? null);

    expect(document.body.textContent).toContain(`API ${initialMonth}`);
    expect(document.body.textContent).toContain("first memo saving");
    expect(document.body.textContent).toContain("no save error");
    expect(document.body.textContent).not.toContain("메모 저장에 실패했습니다.");

    await resolvePending(memoSaveResolves[1] ?? null);
    await waitForText("first memo idle");
  });
});

function MonthControllerHarness() {
  const controller = useMonthController();
  const firstGoal = controller.goals[0];
  const firstDay = controller.days[0];
  const firstCheckKey =
    firstGoal && firstDay ? checkKey(firstGoal.id, firstDay.date) : null;
  const firstCheckCompleted =
    firstGoal && firstDay
      ? controller.checks.some(
          (check) =>
            check.goalId === firstGoal.id && check.date === firstDay.date,
        )
      : false;
  const firstCheckSaving =
    firstCheckKey !== null && controller.savingChecks.includes(firstCheckKey);
  const firstMemoSaving =
    firstDay !== undefined && controller.savingMemos.includes(firstDay.date);

  return (
    <section>
      <p>{controller.month}</p>
      <p>{controller.goals[0]?.title}</p>
      <p>{controller.loadStatus}</p>
      <p>{controller.loadError ?? "no load error"}</p>
      <p>{controller.saveError ?? "no save error"}</p>
      <p>
        {firstCheckCompleted
          ? "first check completed"
          : "first check incomplete"}
      </p>
      <p>{firstCheckSaving ? "first check saving" : "first check idle"}</p>
      <p>{firstMemoSaving ? "first memo saving" : "first memo idle"}</p>
      <button
        aria-label="이전 월 로드"
        type="button"
        onClick={() =>
          void controller.loadMonth(offsetMonth(controller.month, -1))
        }
      >
        load previous
      </button>
      <button
        aria-label="다음 월 로드"
        type="button"
        onClick={() =>
          void controller.loadMonth(offsetMonth(controller.month, 1))
        }
      >
        load next
      </button>
      <button
        aria-label="첫 체크 토글"
        type="button"
        disabled={!firstGoal || !firstDay}
        onClick={() => {
          if (firstGoal && firstDay) {
            void controller.toggleCheck(firstGoal.id, firstDay.date);
          }
        }}
      >
        toggle first check
      </button>
      <button
        aria-label="첫 메모 저장"
        type="button"
        disabled={!firstDay}
        onClick={() => {
          if (firstDay) {
            void controller.saveMemoForDate(firstDay.date, "stale memo");
          }
        }}
      >
        save first memo
      </button>
    </section>
  );
}

function buildLabeledMonthView(month: string) {
  const view = buildMonthView(month);

  return {
    ...view,
    goals: view.goals.map((goal, index) =>
      index === 0 ? { ...goal, title: `API ${month}` } : goal,
    ),
  };
}
