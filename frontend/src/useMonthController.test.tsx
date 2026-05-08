// @vitest-environment jsdom

import { afterEach, describe, expect, it } from "vitest";

import {
  buildMonthView,
  monthFromRequest,
  pendingErrorResponse,
  pendingJsonResponse,
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
});

function MonthControllerHarness() {
  const controller = useMonthController();

  return (
    <section>
      <p>{controller.month}</p>
      <p>{controller.goals[0]?.title}</p>
      <p>{controller.loadStatus}</p>
      <p>{controller.loadError ?? "no load error"}</p>
      <button
        aria-label="이전 월 로드"
        type="button"
        onClick={() =>
          void controller.loadMonth(offsetMonth(controller.month, -1))
        }
      >
        load previous
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
