// @vitest-environment jsdom

import { type FormEvent } from "react";
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
  setInputValue,
  waitFor,
  waitForText,
} from "./App.testHelpers";
import { checkKey } from "./goalSlots";
import { offsetMonth } from "./monthLogic";
import {
  type MonthControllerMode,
  useMonthController,
} from "./useMonthController";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

describe("useMonthController", () => {
  afterEach(async () => {
    await cleanupAppTest();
  });

  it("keeps preview changes local without calling save APIs", async () => {
    const fetchMock = stubFetch(() => {
      throw new Error("preview mode should not call the API");
    });

    renderApp(<MonthControllerHarness mode="preview" />);
    await waitForText("local");
    await waitForText("no first goal");

    expect(fetchMock).not.toHaveBeenCalled();

    await clickButton("첫 메모 변경");
    await waitForText("preview memo");
    await clickButton("첫 메모 저장");
    await waitForText("저장하려면 로그인해 주세요.");
    expect(fetchMock).not.toHaveBeenCalled();

    await setInputValue("새 목표 제목", "Preview goal");
    await clickButton("새 목표 제출");
    await waitForText("Preview goal");
    expect(fetchMock).not.toHaveBeenCalled();

    await clickButton("첫 체크 토글");
    await waitForText("first check completed");
    expect(fetchMock).not.toHaveBeenCalled();

    await clickButton("목표 제목 수정 시작");
    await setInputValue("목표 수정 제목", "Preview walk");
    await clickButton("목표 제목 수정 제출");
    await waitForText("Preview walk");
    expect(fetchMock).not.toHaveBeenCalled();

    await clickButton("첫 목표 종료");
    await waitForText("저장하려면 로그인해 주세요.");
    expect(fetchMock).not.toHaveBeenCalled();
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

  it("ignores stale month mutation failures after returning to the same month", async () => {
    const mutationResolves: Array<() => void> = [];
    const fetchMock = stubFetch((input) => {
      const path = requestPath(input);
      if (
        path.includes("/ensure") ||
        path.endsWith("/goals") ||
        path === "/api/goals/1" ||
        path === "/api/goals/1/deactivate"
      ) {
        return pendingErrorResponse(500, (resolve) => {
          mutationResolves.push(resolve);
        });
      }

      const month = monthFromRequest(input);
      return jsonResponse(buildLabeledMonthView(month));
    });

    renderApp(<MonthControllerHarness />);
    await waitFor(() => fetchMock.mock.calls.length >= 1);

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    await waitForText(`API ${initialMonth}`);

    await clickButton("목표 이월");
    await waitFor(() => mutationResolves.length >= 1);

    const previousMonth = offsetMonth(initialMonth, -1);
    await clickButton("이전 월 로드");
    await waitForText(`API ${previousMonth}`);

    await clickButton("다음 월 로드");
    await waitForText(`API ${initialMonth}`);

    await resolvePending(mutationResolves[0] ?? null);

    expect(document.body.textContent).toContain(`API ${initialMonth}`);
    expect(document.body.textContent).toContain("no save error");
    expect(document.body.textContent).not.toContain("목표 이월에 실패했습니다.");

    await setInputValue("새 목표 제목", "New API goal");
    await clickButton("새 목표 제출");
    await waitFor(() => mutationResolves.length >= 2);

    await clickButton("이전 월 로드");
    await waitForText(`API ${previousMonth}`);

    await clickButton("다음 월 로드");
    await waitForText(`API ${initialMonth}`);

    await resolvePending(mutationResolves[1] ?? null);

    expect(document.body.textContent).toContain(`API ${initialMonth}`);
    expect(document.body.textContent).toContain("no save error");
    expect(document.body.textContent).not.toContain("목표 추가에 실패했습니다.");

    await clickButton("목표 제목 수정 시작");
    await setInputValue("목표 수정 제목", "Updated API goal");
    await clickButton("목표 제목 수정 제출");
    await waitFor(() => mutationResolves.length >= 3);

    await clickButton("이전 월 로드");
    await waitForText(`API ${previousMonth}`);

    await clickButton("다음 월 로드");
    await waitForText(`API ${initialMonth}`);

    await resolvePending(mutationResolves[2] ?? null);

    expect(document.body.textContent).toContain(`API ${initialMonth}`);
    expect(document.body.textContent).toContain("no save error");
    expect(document.body.textContent).not.toContain("목표 수정에 실패했습니다.");

    await clickButton("첫 목표 종료");
    await waitFor(() => mutationResolves.length >= 4);

    await clickButton("이전 월 로드");
    await waitForText(`API ${previousMonth}`);

    await clickButton("다음 월 로드");
    await waitForText(`API ${initialMonth}`);

    await resolvePending(mutationResolves[3] ?? null);

    expect(document.body.textContent).toContain(`API ${initialMonth}`);
    expect(document.body.textContent).toContain("no save error");
    expect(document.body.textContent).not.toContain("목표 종료에 실패했습니다.");
  });

  it("ignores stale month refresh responses after returning to the same month", async () => {
    const pendingRefreshes = new Map<string, () => void>();
    const fetchMock = stubFetch((input) => {
      const path = requestPath(input);
      if (path.endsWith("/goals")) {
        return pendingResponse((resolve) => {
          pendingRefreshes.set("create", resolve);
        });
      }

      const month = monthFromRequest(input);
      if (pendingRefreshes.has("create") && !pendingRefreshes.has("refresh")) {
        return pendingJsonResponse(
          buildStaleRefreshMonthView(month),
          (resolve) => {
            pendingRefreshes.set("refresh", resolve);
          },
        );
      }

      return jsonResponse(buildLabeledMonthView(month));
    });

    renderApp(<MonthControllerHarness />);
    await waitFor(() => fetchMock.mock.calls.length >= 1);

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    await waitForText(`API ${initialMonth}`);

    await setInputValue("새 목표 제목", "New API goal");
    await clickButton("새 목표 제출");
    await waitFor(() => pendingRefreshes.has("create"));
    await resolvePending(pendingRefreshes.get("create") ?? null);
    await waitFor(() => pendingRefreshes.has("refresh"));

    const previousMonth = offsetMonth(initialMonth, -1);
    await clickButton("이전 월 로드");
    await waitForText(`API ${previousMonth}`);

    await clickButton("다음 월 로드");
    await waitForText(`API ${initialMonth}`);

    await resolvePending(pendingRefreshes.get("refresh") ?? null);

    expect(document.body.textContent).toContain(`API ${initialMonth}`);
    expect(document.body.textContent).toContain("no save error");
    expect(document.body.textContent).not.toContain(
      `STALE REFRESH ${initialMonth}`,
    );
  });

  it("keeps newer month mutation state when stale failures resolve", async () => {
    const createResolves: Array<() => void> = [];
    const fetchMock = stubFetch((input) => {
      if (requestPath(input).endsWith("/goals")) {
        return pendingErrorResponse(500, (resolve) => {
          createResolves.push(resolve);
        });
      }

      const month = monthFromRequest(input);
      return jsonResponse(buildLabeledMonthView(month));
    });

    renderApp(<MonthControllerHarness />);
    await waitFor(() => fetchMock.mock.calls.length >= 1);

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);
    await waitForText(`API ${initialMonth}`);

    await setInputValue("새 목표 제목", "First pending goal");
    await clickButton("새 목표 제출");
    await waitFor(() => createResolves.length >= 1);
    await waitForText("month mutation saving");

    const previousMonth = offsetMonth(initialMonth, -1);
    await clickButton("이전 월 로드");
    await waitForText(`API ${previousMonth}`);

    await clickButton("다음 월 로드");
    await waitForText(`API ${initialMonth}`);
    await waitForText("month mutation idle");

    await setInputValue("새 목표 제목", "Second pending goal");
    await clickButton("새 목표 제출");
    await waitFor(() => createResolves.length >= 2);
    await waitForText("month mutation saving");

    await resolvePending(createResolves[0] ?? null);

    expect(document.body.textContent).toContain(`API ${initialMonth}`);
    expect(document.body.textContent).toContain("month mutation saving");
    expect(document.body.textContent).toContain("no save error");
    expect(document.body.textContent).not.toContain("목표 추가에 실패했습니다.");

    await resolvePending(createResolves[1] ?? null);
    await waitForText("month mutation idle");
  });
});

function MonthControllerHarness({ mode }: { mode?: MonthControllerMode }) {
  const controller = useMonthController({ mode });
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
      <p>{controller.goals[0]?.title ?? "no first goal"}</p>
      <p>{controller.goals.map((goal) => goal.title).join(" | ")}</p>
      <p>{controller.loadStatus}</p>
      <p>{controller.loadError ?? "no load error"}</p>
      <p>{controller.saveError ?? "no save error"}</p>
      <p>{controller.saveMessage ?? "no save message"}</p>
      <p>{firstDay?.memo ?? "no memo"}</p>
      <p>
        {firstCheckCompleted
          ? "first check completed"
          : "first check incomplete"}
      </p>
      <p>{firstCheckSaving ? "first check saving" : "first check idle"}</p>
      <p>{firstMemoSaving ? "first memo saving" : "first memo idle"}</p>
      <p>
        {controller.isMutatingMonth
          ? "month mutation saving"
          : "month mutation idle"}
      </p>
      <input
        aria-label="새 목표 제목"
        value={controller.newGoalTitle}
        onChange={(event) => {
          controller.setNewGoalTitle(event.target.value);
        }}
      />
      {controller.editingGoalID !== null ? (
        <input
          aria-label="목표 수정 제목"
          value={controller.editingGoalTitle}
          onChange={(event) => {
            controller.setEditingGoalTitle(event.target.value);
          }}
        />
      ) : null}
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
      <button
        aria-label="첫 메모 변경"
        type="button"
        disabled={!firstDay}
        onClick={() => {
          if (firstDay) {
            controller.updateMemo(firstDay.date, "preview memo");
          }
        }}
      >
        change first memo
      </button>
      <button
        aria-label="목표 이월"
        type="button"
        onClick={() => {
          void controller.prepareCurrentMonth();
        }}
      >
        prepare month
      </button>
      <button
        aria-label="새 목표 제출"
        type="button"
        onClick={() => {
          void controller.submitNewGoal(testFormEvent());
        }}
      >
        create goal
      </button>
      <button
        aria-label="목표 제목 수정 시작"
        type="button"
        disabled={!firstGoal}
        onClick={() => {
          if (firstGoal) {
            controller.startEditingGoal(firstGoal);
          }
        }}
      >
        start update goal
      </button>
      <button
        aria-label="목표 제목 수정 제출"
        type="button"
        disabled={!firstGoal}
        onClick={() => {
          if (firstGoal) {
            void controller.submitGoalTitle(testFormEvent(), firstGoal.id);
          }
        }}
      >
        update goal
      </button>
      <button
        aria-label="첫 목표 종료"
        type="button"
        disabled={!firstGoal}
        onClick={() => {
          if (firstGoal) {
            void controller.deactivateGoalFromMonth(firstGoal);
          }
        }}
      >
        deactivate first goal
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

function buildStaleRefreshMonthView(month: string) {
  const view = buildMonthView(month);

  return {
    ...view,
    goals: view.goals.map((goal, index) =>
      index === 0 ? { ...goal, title: `STALE REFRESH ${month}` } : goal,
    ),
  };
}

function testFormEvent() {
  return {
    preventDefault() {
      return undefined;
    },
  } as FormEvent<HTMLFormElement>;
}
