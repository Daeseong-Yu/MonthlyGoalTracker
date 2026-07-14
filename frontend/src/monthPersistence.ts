import {
  createGoal,
  deactivateGoal,
  ensureMonth,
  getMonthView,
  saveMemo,
  setGoalCompleted,
  updateGoalTitle,
} from "./api";
import type { LoadStatus } from "./appDisplay";
import { buildEmptyMonthView, buildMockMonthView } from "./mockMonth";
import { applyGoalEndDateState, applyGoalTitleState } from "./monthLogic";
import type { Goal, MonthView } from "./types";

export type MonthControllerMode = "server" | "preview";
export type MonthStorage = "server" | "local";

export type MonthPersistenceState = {
  loadStatus: LoadStatus;
  view: MonthView;
};

export type MonthMutationResult = {
  apply(view: MonthView): MonthView;
  refresh?: () => Promise<MonthView>;
  storage: MonthStorage;
};

export type MonthServerGateway = {
  createGoal: typeof createGoal;
  deactivateGoal: typeof deactivateGoal;
  ensureMonth: typeof ensureMonth;
  getMonthView: typeof getMonthView;
  saveMemo: typeof saveMemo;
  setGoalCompleted: typeof setGoalCompleted;
  updateGoalTitle: typeof updateGoalTitle;
};

export type MonthPersistence = {
  initialState(month: string): MonthPersistenceState;
  loadMonth(month: string): Promise<MonthPersistenceState>;
  prepareMonth(month: string): Promise<MonthMutationResult>;
  saveCheck(
    goalId: number,
    date: string,
    completed: boolean,
  ): Promise<MonthMutationResult>;
  saveMemo(date: string, memo: string): Promise<MonthMutationResult>;
  createGoal(
    month: string,
    title: string,
    startDate: string,
  ): Promise<MonthMutationResult>;
  updateGoalTitle(
    month: string,
    goalId: number,
    title: string,
  ): Promise<MonthMutationResult>;
  deactivateGoal(
    month: string,
    goalId: number,
    endDate: string,
  ): Promise<MonthMutationResult>;
};

const defaultServerGateway: MonthServerGateway = {
  createGoal,
  deactivateGoal,
  ensureMonth,
  getMonthView,
  saveMemo,
  setGoalCompleted,
  updateGoalTitle,
};

export function createMonthPersistence(
  mode: MonthControllerMode,
  gateway: MonthServerGateway = defaultServerGateway,
): MonthPersistence {
  if (mode === "preview") {
    return createPreviewMonthPersistence();
  }

  return createServerMonthPersistence(gateway);
}

function createServerMonthPersistence(
  gateway: MonthServerGateway,
): MonthPersistence {
  const serverResult = (
    apply: MonthMutationResult["apply"],
    refresh?: () => Promise<MonthView>,
  ): MonthMutationResult =>
    refresh === undefined
      ? { apply, storage: "server" }
      : { apply, refresh, storage: "server" };

  return {
    initialState(month) {
      return {
        loadStatus: "loading",
        view: buildMockMonthView(month),
      };
    },
    async loadMonth(month) {
      return {
        loadStatus: "api",
        view: await gateway.getMonthView(month),
      };
    },
    async prepareMonth(month) {
      const preparedView = await gateway.ensureMonth(month);
      return serverResult(() => preparedView);
    },
    async saveCheck(goalId, date, completed) {
      await gateway.setGoalCompleted(goalId, date, completed);
      return serverResult(preserveMonthView);
    },
    async saveMemo(date, memo) {
      await gateway.saveMemo(date, memo);
      return serverResult(preserveMonthView);
    },
    async createGoal(month, title, startDate) {
      return writeWithRefresh(
        () => gateway.createGoal(month, title, startDate),
        preserveMonthView,
        month,
      );
    },
    async updateGoalTitle(month, goalId, title) {
      return writeWithRefresh(
        () => gateway.updateGoalTitle(goalId, title),
        (view) => applyGoalTitleState(view, goalId, title),
        month,
      );
    },
    async deactivateGoal(month, goalId, endDate) {
      return writeWithRefresh(
        () => gateway.deactivateGoal(goalId, endDate),
        (view) => applyGoalEndDateState(view, goalId, endDate),
        month,
      );
    },
  };

  async function writeWithRefresh(
    write: () => Promise<void>,
    apply: MonthMutationResult["apply"],
    month: string,
  ): Promise<MonthMutationResult> {
    await write();
    return serverResult(apply, () => gateway.getMonthView(month));
  }
}

function createPreviewMonthPersistence(): MonthPersistence {
  const localResult = (
    apply: MonthMutationResult["apply"],
  ): MonthMutationResult => ({
    apply,
    storage: "local",
  });

  return {
    initialState(month) {
      return localState(month);
    },
    async loadMonth(month) {
      return localState(month);
    },
    async prepareMonth() {
      return localResult(preserveMonthView);
    },
    async saveCheck() {
      return localResult(preserveMonthView);
    },
    async saveMemo() {
      return localResult(preserveMonthView);
    },
    async createGoal(_month, title, startDate) {
      return localResult((view) =>
        appendPreviewGoalState(view, title, startDate),
      );
    },
    async updateGoalTitle(_month, goalId, title) {
      return localResult((view) => applyGoalTitleState(view, goalId, title));
    },
    async deactivateGoal(_month, goalId, endDate) {
      return localResult((view) =>
        applyGoalEndDateState(view, goalId, endDate),
      );
    },
  };
}

function preserveMonthView(view: MonthView) {
  return view;
}

function localState(month: string): MonthPersistenceState {
  return {
    loadStatus: "local",
    view: buildEmptyMonthView(month),
  };
}

function appendPreviewGoalState(
  view: MonthView,
  title: string,
  startDate: string,
): MonthView {
  const goal: Goal = {
    id: nextPreviewGoalID(view.goals),
    title,
    startDate,
    endDate: null,
  };

  return {
    ...view,
    goals: [...view.goals, goal],
  };
}

function nextPreviewGoalID(goals: Goal[]) {
  return goals.reduce((maxID, goal) => Math.max(maxID, goal.id), 0) + 1;
}
