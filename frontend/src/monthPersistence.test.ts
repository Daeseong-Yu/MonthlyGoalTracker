import { describe, expect, it, vi } from "vitest";

import {
  createMonthPersistence,
  type MonthServerGateway,
} from "./monthPersistence";
import { buildEmptyMonthView } from "./mockMonth";

describe("month persistence", () => {
  it("keeps preview month mutations local without using the server gateway", async () => {
    const gateway: MonthServerGateway = {
      createGoal: vi.fn(),
      deactivateGoal: vi.fn(),
      ensureMonth: vi.fn(),
      getMonthView: vi.fn(),
      saveMemo: vi.fn(),
      setGoalCompleted: vi.fn(),
      updateGoalTitle: vi.fn(),
    };
    const persistence = createMonthPersistence("preview", gateway);

    const initial = persistence.initialState("2026-04");
    expect(initial.loadStatus).toBe("local");
    expect(initial.view.goals).toEqual([]);

    const loaded = await persistence.loadMonth("2026-05");
    expect(loaded.loadStatus).toBe("local");
    expect(loaded.view.month).toBe("2026-05");

    const created = await persistence.createGoal(
      "2026-05",
      "Preview walk",
      "2026-05-02",
    );
    const createdView = created.apply(loaded.view);
    expect(created.storage).toBe("local");
    expect(createdView.goals).toEqual([
      {
        id: 1,
        title: "Preview walk",
        startDate: "2026-05-02",
        endDate: null,
      },
    ]);

    const updated = await persistence.updateGoalTitle(
      "2026-05",
      1,
      "Preview run",
    );
    const updatedView = updated.apply(createdView);
    expect(updatedView.goals[0]?.title).toBe("Preview run");

    const deactivated = await persistence.deactivateGoal(
      "2026-05",
      1,
      "2026-05-10",
    );
    const deactivatedView = deactivated.apply(updatedView);
    expect(deactivatedView.goals[0]?.endDate).toBe("2026-05-10");

    await persistence.prepareMonth("2026-05");
    await persistence.saveCheck(1, "2026-05-03", true);
    await persistence.saveMemo("2026-05-03", "preview memo");

    for (const operation of Object.values(gateway)) {
      expect(operation).not.toHaveBeenCalled();
    }
  });

  it("refreshes the server month after creating a goal", async () => {
    const currentView = buildEmptyMonthView("2026-04");
    const refreshedView = {
      ...currentView,
      goals: [
        {
          id: 7,
          title: "Server walk",
          startDate: "2026-04-02",
          endDate: null,
        },
      ],
    };
    const gateway: MonthServerGateway = {
      createGoal: vi.fn().mockResolvedValue(undefined),
      deactivateGoal: vi.fn().mockResolvedValue(undefined),
      ensureMonth: vi.fn().mockResolvedValue(refreshedView),
      getMonthView: vi
        .fn()
        .mockResolvedValueOnce(currentView)
        .mockResolvedValueOnce(refreshedView),
      saveMemo: vi.fn().mockResolvedValue(undefined),
      setGoalCompleted: vi.fn().mockResolvedValue(undefined),
      updateGoalTitle: vi.fn().mockResolvedValue(undefined),
    };
    const persistence = createMonthPersistence("server", gateway);

    const initial = persistence.initialState("2026-04");
    expect(initial.loadStatus).toBe("loading");

    const loaded = await persistence.loadMonth("2026-04");
    expect(loaded).toEqual({ loadStatus: "api", view: currentView });

    const created = await persistence.createGoal(
      "2026-04",
      "Server walk",
      "2026-04-02",
    );
    expect(gateway.createGoal).toHaveBeenCalledWith(
      "2026-04",
      "Server walk",
      "2026-04-02",
    );
    expect(gateway.getMonthView).toHaveBeenCalledTimes(1);
    expect(created.storage).toBe("server");
    expect(created.apply(loaded.view)).toEqual(currentView);
    expect(created.refresh).toBeTypeOf("function");

    const refreshed = await created.refresh?.();

    expect(gateway.getMonthView).toHaveBeenLastCalledWith("2026-04");
    expect(refreshed).toEqual(refreshedView);
  });

  it("routes direct server saves through the gateway", async () => {
    const currentView = buildEmptyMonthView("2026-04");
    const preparedView = {
      ...currentView,
      goals: [
        {
          id: 3,
          title: "Carried goal",
          startDate: "2026-04-01",
          endDate: null,
        },
      ],
    };
    const gateway: MonthServerGateway = {
      createGoal: vi.fn().mockResolvedValue(undefined),
      deactivateGoal: vi.fn().mockResolvedValue(undefined),
      ensureMonth: vi.fn().mockResolvedValue(preparedView),
      getMonthView: vi.fn().mockResolvedValue(currentView),
      saveMemo: vi.fn().mockResolvedValue(undefined),
      setGoalCompleted: vi.fn().mockResolvedValue(undefined),
      updateGoalTitle: vi.fn().mockResolvedValue(undefined),
    };
    const persistence = createMonthPersistence("server", gateway);

    const prepared = await persistence.prepareMonth("2026-04");
    const preparedMonthView = prepared.apply(currentView);
    const checked = await persistence.saveCheck(
      3,
      "2026-04-02",
      true,
    );
    const checkedMonthView = checked.apply(preparedMonthView);
    const memoSaved = await persistence.saveMemo(
      "2026-04-02",
      "server memo",
    );

    expect(gateway.ensureMonth).toHaveBeenCalledWith("2026-04");
    expect(gateway.setGoalCompleted).toHaveBeenCalledWith(
      3,
      "2026-04-02",
      true,
    );
    expect(gateway.saveMemo).toHaveBeenCalledWith(
      "2026-04-02",
      "server memo",
    );
    expect(preparedMonthView).toEqual(preparedView);
    expect(checkedMonthView).toEqual(preparedView);
    expect(memoSaved.apply(checkedMonthView)).toEqual(preparedView);
    expect(prepared.storage).toBe("server");
    expect(checked.storage).toBe("server");
    expect(memoSaved.storage).toBe("server");
  });

  it("refreshes the server month after editing or deactivating a goal", async () => {
    const currentView = {
      ...buildEmptyMonthView("2026-04"),
      goals: [
        {
          id: 4,
          title: "Original goal",
          startDate: "2026-04-01",
          endDate: null,
        },
      ],
    };
    const updatedView = {
      ...currentView,
      goals: [{ ...currentView.goals[0], title: "Updated goal" }],
    };
    const deactivatedView = {
      ...updatedView,
      goals: [{ ...updatedView.goals[0], endDate: "2026-04-12" }],
    };
    const gateway: MonthServerGateway = {
      createGoal: vi.fn().mockResolvedValue(undefined),
      deactivateGoal: vi.fn().mockResolvedValue(undefined),
      ensureMonth: vi.fn().mockResolvedValue(currentView),
      getMonthView: vi
        .fn()
        .mockResolvedValueOnce(updatedView)
        .mockResolvedValueOnce(deactivatedView),
      saveMemo: vi.fn().mockResolvedValue(undefined),
      setGoalCompleted: vi.fn().mockResolvedValue(undefined),
      updateGoalTitle: vi.fn().mockResolvedValue(undefined),
    };
    const persistence = createMonthPersistence("server", gateway);

    const updated = await persistence.updateGoalTitle(
      "2026-04",
      4,
      "Updated goal",
    );
    const committedUpdatedView = updated.apply(currentView);
    expect(committedUpdatedView).toEqual(updatedView);
    expect(gateway.getMonthView).not.toHaveBeenCalled();
    expect(await updated.refresh?.()).toEqual(updatedView);

    const deactivated = await persistence.deactivateGoal(
      "2026-04",
      4,
      "2026-04-12",
    );
    expect(deactivated.apply(committedUpdatedView)).toEqual(deactivatedView);
    expect(await deactivated.refresh?.()).toEqual(deactivatedView);

    expect(gateway.updateGoalTitle).toHaveBeenCalledWith(4, "Updated goal");
    expect(gateway.deactivateGoal).toHaveBeenCalledWith(4, "2026-04-12");
    expect(gateway.getMonthView).toHaveBeenNthCalledWith(1, "2026-04");
    expect(gateway.getMonthView).toHaveBeenNthCalledWith(2, "2026-04");
    expect(updated.storage).toBe("server");
    expect(deactivated.storage).toBe("server");
  });

  it("keeps server write failures separate from lazy refresh failures", async () => {
    const currentView = buildEmptyMonthView("2026-04");
    const gateway: MonthServerGateway = {
      createGoal: vi
        .fn()
        .mockRejectedValueOnce(new Error("write failed"))
        .mockResolvedValueOnce(undefined),
      deactivateGoal: vi.fn().mockResolvedValue(undefined),
      ensureMonth: vi.fn().mockResolvedValue(currentView),
      getMonthView: vi.fn().mockRejectedValue(new Error("refresh failed")),
      saveMemo: vi.fn().mockResolvedValue(undefined),
      setGoalCompleted: vi.fn().mockResolvedValue(undefined),
      updateGoalTitle: vi.fn().mockResolvedValue(undefined),
    };
    const persistence = createMonthPersistence("server", gateway);

    await expect(
      persistence.createGoal(
        "2026-04",
        "Server walk",
        "2026-04-02",
      ),
    ).rejects.toThrow("write failed");
    expect(gateway.getMonthView).not.toHaveBeenCalled();

    const written = await persistence.createGoal(
      "2026-04",
      "Server walk",
      "2026-04-02",
    );
    expect(gateway.getMonthView).not.toHaveBeenCalled();
    expect(written.refresh).toBeTypeOf("function");
    const refresh = written.refresh;
    if (refresh === undefined) {
      throw new Error("expected a server refresh operation");
    }

    await expect(refresh()).rejects.toThrow("refresh failed");
    expect(gateway.getMonthView).toHaveBeenCalledWith("2026-04");
  });
});
