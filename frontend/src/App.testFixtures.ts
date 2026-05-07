import { nextDate } from "./goalSlots";
import { isGoalActiveOnDate } from "./monthLogic";
import type { Goal, GoalCheck, MonthView } from "./types";

type MonthViewOptions = {
  firstGoalEndDate?: string | null;
  goalCount?: number;
  includeSunday?: boolean;
};

type MonthViewOptionsInput =
  | MonthViewOptions
  | ((month: string) => MonthViewOptions);

type AppFetchHandler = (
  input: RequestInfo | URL,
  init?: RequestInit,
) => Response | Promise<Response>;

type FetchCallSource = {
  mock: {
    calls: Parameters<typeof fetch>[];
  };
};

export function buildMonthView(
  month: string,
  options: MonthViewOptions = {},
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

export function buildMonthViewFromGoals(
  month: string,
  goals: Goal[],
): MonthView {
  const days = buildDays(month, goals, false);

  return {
    month,
    goals,
    days,
    checks: [{ goalId: 1, date: `${month}-01`, completed: true }],
    chart: days,
  };
}

export function buildGoals(
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

export function appendGoal(
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

export function updateGoalInView(
  view: MonthView,
  goalId: number,
  title: string,
) {
  return {
    ...view,
    goals: view.goals.map((goal) =>
      goal.id === goalId ? { ...goal, title } : goal,
    ),
  };
}

export function updateCheckInView(
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

export function updateMemoInView(
  view: MonthView,
  date: string,
  memo: string,
) {
  return {
    ...view,
    days: view.days.map((day) =>
      day.date === date ? { ...day, memo } : day,
    ),
  };
}

export function requiredView(view: MonthView | null) {
  if (view === null) {
    throw new Error("expected loaded month view");
  }

  return view;
}

export function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
    },
  });
}

export function errorResponse(status: number) {
  return new Response(null, { status });
}

export function okResponse() {
  return new Response(null, { status: 204 });
}

export function pendingResponse(captureResolve: (resolve: () => void) => void) {
  return new Promise<Response>((resolve) => {
    captureResolve(() => resolve(okResponse()));
  });
}

export function monthFromRequest(input: RequestInfo | URL) {
  const path = requestPath(input);
  const match = /\/api\/months\/([^/]+)/.exec(path);

  if (!match) {
    throw new Error(`expected month API request, got ${path}`);
  }

  return decodeURIComponent(match[1]);
}

export function monthFromGoalCreatePath(path: string) {
  const match = /\/api\/months\/([^/]+)\/goals$/.exec(path);

  if (!match) {
    throw new Error(`expected goal create API request, got ${path}`);
  }

  return decodeURIComponent(match[1]);
}

export function shortFirstDayLabel(month: string) {
  return `${month.slice(5)}.01`;
}

export function shortDate(date: string) {
  return date.slice(5).replace("-", ".");
}

export function requestPath(input: RequestInfo | URL) {
  return typeof input === "string"
    ? input
    : input instanceof URL
      ? input.pathname
      : input.url;
}

export function bodyEndDate(init: RequestInit | undefined) {
  const body = requestBody<{ endDate?: string }>(init);
  if (body.endDate === undefined) {
    throw new Error("expected endDate in request body");
  }

  return body.endDate;
}

export function requestBody<T>(init: RequestInit | undefined) {
  if (typeof init?.body !== "string") {
    throw new Error("expected JSON request body");
  }

  return JSON.parse(init.body) as T;
}

export function hasFetchedPath(
  fetchMock: FetchCallSource,
  matchesPath: (path: string) => boolean,
) {
  return fetchMock.mock.calls.some(([input]) => matchesPath(requestPath(input)));
}

export function hasNoFetchedPath(
  fetchMock: FetchCallSource,
  matchesPath: (path: string) => boolean,
) {
  return fetchMock.mock.calls.every(
    ([input]) => !matchesPath(requestPath(input)),
  );
}

export function hasFetchedJsonBody<T>(
  fetchMock: FetchCallSource,
  matchesPath: (path: string) => boolean,
  matchesBody: (body: T) => boolean,
) {
  return fetchMock.mock.calls.some(([input, init]) => {
    if (!matchesPath(requestPath(input))) {
      return false;
    }

    return matchesBody(requestBody<T>(init));
  });
}

export function dateAfter(date: string | null) {
  if (date === null) {
    throw new Error("expected date");
  }

  return nextDate(date);
}

export function createMonthViewApiHandler(
  options: MonthViewOptionsInput = {},
): AppFetchHandler {
  return (input) => {
    const month = monthFromRequest(input);
    const resolvedOptions =
      typeof options === "function" ? options(month) : options;

    return jsonResponse(buildMonthView(month, resolvedOptions));
  };
}

export function createWorkflowApiHandler(): AppFetchHandler {
  let view: MonthView | null = null;
  let nextGoalID = 3;

  return (input, init) => {
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
  };
}

export function createFailingWorkflowApiHandler(
  status = 500,
): AppFetchHandler {
  return (input) => {
    const path = requestPath(input);

    if (isWorkflowWritePath(path)) {
      return errorResponse(status);
    }

    return jsonResponse(buildMonthView(monthFromRequest(input)));
  };
}

function isWorkflowWritePath(path: string) {
  return (
    path.endsWith("/goals") ||
    path === "/api/goals/1" ||
    path === "/api/checks" ||
    path.includes("/api/memos/")
  );
}
