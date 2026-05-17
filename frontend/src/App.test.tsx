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
  jsonErrorResponse,
  jsonResponse,
  monthFromRequest,
  okResponse,
  pendingJsonResponse,
  pendingResponse,
  requestPath,
  shortDate,
  shortFirstDayLabel,
} from "./App.testFixtures";
import {
  blurInput,
  cleanupAppTest,
  clickButton,
  clickElement,
  getButton,
  getDailyRecordTable,
  getDashboardLayout,
  getHeading,
  getInput,
  getWeekdayLabel,
  hasInputValue,
  precedes,
  pressKey,
  queryButton,
  renderApp,
  resolvePending,
  setInputValue,
  stubFetch,
  submitForm,
  tableHeaderCells,
  tableHeaders,
  waitFor,
  waitForText,
} from "./App.testHelpers";
import { currentMonth, monthStartDate, offsetMonth } from "./monthLogic";
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

    await waitForText("계정 데이터");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(document.body.textContent).toContain("API walk");
    expect(hasInputValue("api memo")).toBe(true);
    expect(getInput("05.01 메모").className).toContain("w-56");
    expect(getInput("05.01 메모").className).toContain("max-w-full");
    expect(document.body.textContent).toContain("chart points: 2");
  });

  it("opens a no-save preview for unauthenticated visitors", async () => {
    const fetchMock = stubFetch(createMonthViewApiHandler(), {
      bootstrap: { authenticated: false, locale: "ko", user: null },
    });

    renderApp(<App />);

    await waitForText("체험 모드");

    expect(document.body.textContent).toContain("체험 모드");
    expect(document.body.textContent).toContain("저장하려면 로그인해야 합니다.");
    expect(document.body.textContent).toContain("진행 중인 목표가 없습니다.");
    expect(document.body.textContent).not.toContain("아침 산책");
    expect(document.body.textContent).not.toContain(
      "비가 와서 실내 운동으로 변경",
    );
    expect(getButton("로그인")).toBeTruthy();
    expect(queryButton("로그아웃")).toBeNull();
    expect(document.body.textContent).not.toContain("계정 보안");
    expect(hasNoFetchedPath(fetchMock, (path) => path.includes("/api/months/")))
      .toBe(true);

    await setInputValue("05.01 메모", "preview memo");
    await blurInput("05.01 메모");

    await waitForText("저장하려면 로그인해 주세요.");
    expect(queryButton("로그인하고 저장하기")).toBeNull();
    expect(hasNoFetchedPath(fetchMock, (path) => path.includes("/api/memos/")))
      .toBe(true);

    await clickButton("로그인");
    await waitFor(() => document.querySelector("[role='dialog']") !== null);
    expect(document.body.textContent).toContain(
      "목표를 저장하려면 로그인해 주세요.",
    );
    await pressKey("Escape");
    await waitFor(() => document.querySelector("[role='dialog']") === null);

    await clickButton("목표 추가");
    await setInputValue("새 목표 제목", "Preview goal");
    await clickButton("목표 저장");
    await waitForText("Preview goal");
    expect(hasNoFetchedPath(fetchMock, (path) => path.includes("/api/months/")))
      .toBe(true);
  });

  it("closes the auth modal without leaving the anonymous preview", async () => {
    stubFetch(createMonthViewApiHandler(), {
      bootstrap: { authenticated: false, locale: "ko", user: null },
    });

    renderApp(<App />);

    await waitForText("체험 모드");
    await clickButton("로그인");
    await waitFor(() => document.querySelector("[role='dialog']") !== null);

    await pressKey("Escape");

    await waitFor(() => document.querySelector("[role='dialog']") === null);
    expect(document.body.textContent).toContain("진행 중인 목표가 없습니다.");

    await clickButton("로그인");
    await waitFor(() => document.querySelector("[role='dialog']") !== null);
    const dialog = document.querySelector("[role='dialog']");

    if (!dialog?.parentElement) {
      throw new Error("expected auth modal backdrop");
    }

    await clickElement(dialog.parentElement);

    await waitFor(() => document.querySelector("[role='dialog']") === null);
    expect(document.body.textContent).toContain("진행 중인 목표가 없습니다.");
  });

  it("shows the English login screen from regional bootstrap and loads the dashboard after login", async () => {
    const fetchMock = stubFetch(
      (input) => {
        if (requestPath(input) === "/api/auth/login") {
          return jsonResponse({
            user: {
              id: 4,
              email: "person@example.com",
              locale: "en",
              createdAt: "2026-05-01T00:00:00Z",
            },
            csrfToken: "csrf-login",
            locale: "en",
          });
        }

        return createMonthViewApiHandler()(input);
      },
      {
        bootstrap: {
          authenticated: false,
          locale: "en",
          user: null,
        },
      },
    );

    renderApp(<App />);

    await waitForText("Guest mode");

    expect(document.body.textContent).toContain("Guest mode");
    expect(document.body.textContent?.match(/Guest mode/g) ?? []).toHaveLength(
      1,
    );
    expect(document.body.textContent).toContain("KO");
    expect(document.body.textContent).toContain("No active goals.");
    expect(document.body.textContent).not.toContain("아침 산책");
    expect(hasNoFetchedPath(fetchMock, (path) => path.includes("/api/months/")))
      .toBe(true);

    await clickButton("Log in");

    expect(document.querySelector("[role='dialog']")).not.toBeNull();
    expect(document.body.textContent).toContain("No active goals.");
    expect(document.body.textContent).toContain("Log in to save your goals.");

    await setInputValue("Email", "person@example.com");
    await setInputValue("Password", "secret123");
    await submitForm();

    await waitForText("Account data");

    expect(hasFetchedPath(fetchMock, (path) => path === "/api/auth/login")).toBe(
      true,
    );
    expect(document.body.textContent).toContain(
      "Signed in as person@example.com",
    );
    expect(document.body.textContent).toContain("Goals");
  });

  it("shows rate limit feedback on login", async () => {
    const monthViewApiHandler = createMonthViewApiHandler();
    const fetchMock = stubFetch(
      (input) => {
        if (requestPath(input) === "/api/auth/login") {
          return jsonErrorResponse(429, "too many requests");
        }

        return monthViewApiHandler(input);
      },
      {
        bootstrap: {
          authenticated: false,
          locale: "ko",
          user: null,
        },
      },
    );

    renderApp(<App />);

    await waitForText("체험 모드");
    await clickButton("로그인");
    await waitForText("목표를 저장하려면 로그인해 주세요.");
    await setInputValue("이메일", "person@example.com");
    await setInputValue("비밀번호", "secret123");
    await submitForm();

    await waitForText("요청이 너무 많습니다. 잠시 후 다시 시도해 주세요.");

    expect(hasFetchedPath(fetchMock, (path) => path === "/api/auth/login")).toBe(
      true,
    );
  });

  it.each([
    [
      400,
      "signup failed",
      "회원가입에 실패했습니다. 이메일 또는 비밀번호를 확인해 주세요.",
    ],
    [
      409,
      "email already exists",
      "회원가입에 실패했습니다. 이메일 또는 비밀번호를 확인해 주세요.",
    ],
    [400, "weak password", "비밀번호는 8자 이상이어야 합니다."],
    [
      409,
      "legacy claim required",
      "기존 데이터가 있어 이전 토큰이 필요합니다.",
    ],
  ] as const)(
    "shows %s signup feedback from auth errors",
    async (status, serverError, expectedMessage) => {
      const monthViewApiHandler = createMonthViewApiHandler();
      const fetchMock = stubFetch(
        (input) => {
          if (requestPath(input) === "/api/auth/signup") {
            return jsonErrorResponse(status, serverError);
          }

          return monthViewApiHandler(input);
        },
        {
          bootstrap: {
            authenticated: false,
            locale: "ko",
            user: null,
          },
        },
      );

      renderApp(<App />);

      await waitForText("체험 모드");
      await clickButton("로그인");
      await waitForText("목표를 저장하려면 로그인해 주세요.");
      await clickButton("회원가입 tab");
      await setInputValue("이메일", "person@example.com");
      await setInputValue("비밀번호", "secret123");
      await submitForm();

      await waitForText(expectedMessage);

      expect(
        hasFetchedPath(fetchMock, (path) => path === "/api/auth/signup"),
      ).toBe(true);
    },
  );

  it("shows signup verification feedback without opening a session", async () => {
    const monthViewApiHandler = createMonthViewApiHandler();
    const fetchMock = stubFetch(
      (input) => {
        if (requestPath(input) === "/api/auth/signup") {
          return jsonResponse({
            status: "verification_required",
            locale: "ko",
          });
        }

        return monthViewApiHandler(input);
      },
      {
        bootstrap: {
          authenticated: false,
          locale: "ko",
          user: null,
        },
      },
    );

    renderApp(<App />);

    await waitForText("체험 모드");
    await clickButton("로그인");
    await waitForText("목표를 저장하려면 로그인해 주세요.");
    await clickButton("회원가입 tab");
    await setInputValue("이메일", "person@example.com");
    await setInputValue("비밀번호", "secret123");
    await submitForm();

    await waitForText("인증 메일을 확인해 주세요.");

    expect(hasFetchedPath(fetchMock, (path) => path === "/api/auth/signup")).toBe(
      true,
    );
    expect(hasNoFetchedPath(fetchMock, (path) => path.includes("/api/months/")))
      .toBe(true);
  });

  it("verifies an email token from the URL before loading the dashboard", async () => {
    window.history.pushState({}, "", "/?token=email-token");

    const monthViewApiHandler = createMonthViewApiHandler();
    const fetchMock = stubFetch(
      (input) => {
        if (requestPath(input) === "/api/auth/verify-email") {
          return jsonResponse({
            user: {
              id: 4,
              email: "person@example.com",
              locale: "ko",
              createdAt: "2026-05-01T00:00:00Z",
            },
            csrfToken: "csrf-verified",
            locale: "ko",
          });
        }

        return monthViewApiHandler(input);
      },
      {
        bootstrap: {
          authenticated: false,
          locale: "ko",
          user: null,
        },
      },
    );

    renderApp(<App />);

    await waitForText("계정 데이터");

    expect(hasFetchedJsonBody<{ token: string }>(
      fetchMock,
      (path) => path === "/api/auth/verify-email",
      (body) => body.token === "email-token",
    )).toBe(true);
    expect(window.location.search).not.toContain("token=");
    expect(document.body.textContent).toContain(
      "person@example.com로 로그인됨",
    );
  });

  it("shows feedback when an email verification token is invalid", async () => {
    window.history.pushState({}, "", "/?token=expired-token");

    const monthViewApiHandler = createMonthViewApiHandler();
    const fetchMock = stubFetch(
      (input) => {
        if (requestPath(input) === "/api/auth/verify-email") {
          return jsonErrorResponse(400, "invalid verification token");
        }

        return monthViewApiHandler(input);
      },
      {
        bootstrap: {
          authenticated: false,
          locale: "ko",
          user: null,
        },
      },
    );

    renderApp(<App />);

    await waitForText("인증 링크가 만료되었거나 올바르지 않습니다.");

    expect(hasFetchedPath(
      fetchMock,
      (path) => path === "/api/auth/verify-email",
    )).toBe(true);
    expect(window.location.search).not.toContain("token=");
    expect(hasNoFetchedPath(fetchMock, (path) => path.includes("/api/months/")))
      .toBe(true);
  });

  it("requests a password reset from the login screen", async () => {
    const fetchMock = stubFetch(
      (input) => {
        if (requestPath(input) === "/api/auth/password-reset/request") {
          return jsonResponse({
            status: "password_reset_requested",
            locale: "ko",
          });
        }

        return createMonthViewApiHandler()(input);
      },
      {
        bootstrap: { authenticated: false, locale: "ko", user: null },
      },
    );

    renderApp(<App />);

    await waitForText("체험 모드");
    await clickButton("로그인");
    await waitForText("목표를 저장하려면 로그인해 주세요.");
    await clickButton("비밀번호 재설정");
    await setInputValue("이메일", "person@example.com");
    await submitForm();

    await waitForText("재설정 메일을 확인해 주세요.");

    expect(hasFetchedJsonBody<{ email: string; locale: string }>(
      fetchMock,
      (path) => path === "/api/auth/password-reset/request",
      (body) => body.email === "person@example.com" && body.locale === "ko",
    )).toBe(true);
    expect(hasNoFetchedPath(fetchMock, (path) => path.includes("/api/months/")))
      .toBe(true);
  });

  it("resets a password from a URL token before loading the dashboard", async () => {
    window.history.pushState({}, "", "/#resetToken=password-token");

    const monthViewApiHandler = createMonthViewApiHandler();
    const fetchMock = stubFetch(
      (input) => {
        if (requestPath(input) === "/api/auth/password-reset/confirm") {
          return jsonResponse({
            user: {
              id: 4,
              email: "person@example.com",
              locale: "ko",
              createdAt: "2026-05-01T00:00:00Z",
            },
            csrfToken: "csrf-reset",
            locale: "ko",
          });
        }

        return monthViewApiHandler(input);
      },
      {
        bootstrap: { authenticated: false, locale: "ko", user: null },
      },
    );

    renderApp(<App />);

    await waitForText("새 비밀번호 저장");
    await setInputValue("비밀번호", "new-secret123");
    await submitForm();

    await waitForText("계정 데이터");

    expect(hasFetchedJsonBody<{ token: string; password: string }>(
      fetchMock,
      (path) => path === "/api/auth/password-reset/confirm",
      (body) =>
        body.token === "password-token" && body.password === "new-secret123",
    )).toBe(true);
    expect(window.location.search).not.toContain("resetToken=");
    expect(window.location.hash).not.toContain("resetToken=");
    expect(document.body.textContent).toContain(
      "person@example.com로 로그인됨",
    );
  });

  it("changes the signed-in user's password from the dashboard", async () => {
    const monthViewApiHandler = createMonthViewApiHandler();
    const fetchMock = stubFetch((input) => {
      if (requestPath(input) === "/api/auth/password/change") {
        return jsonResponse({
          user: {
            id: 1,
            email: "tester@example.com",
            locale: "ko",
            createdAt: "2026-05-01T00:00:00Z",
          },
          csrfToken: "csrf-changed",
          locale: "ko",
        });
      }

      return monthViewApiHandler(input);
    });

    renderApp(<App />);

    await waitForText("계정 데이터");
    await setInputValue("현재 비밀번호", "old-secret123");
    await setInputValue("새 비밀번호", "new-secret123");
    await submitForm();

    await waitForText("비밀번호를 변경했습니다.");

    expect(hasFetchedJsonBody<{
      currentPassword: string;
      newPassword: string;
    }>(
      fetchMock,
      (path) => path === "/api/auth/password/change",
      (body) =>
        body.currentPassword === "old-secret123" &&
        body.newPassword === "new-secret123",
    )).toBe(true);
    expect(getInput("현재 비밀번호").value).toBe("");
    expect(getInput("새 비밀번호").value).toBe("");
  });

  it("shows password change feedback when the current password is wrong", async () => {
    const monthViewApiHandler = createMonthViewApiHandler();
    const fetchMock = stubFetch((input) => {
      if (requestPath(input) === "/api/auth/password/change") {
        return jsonErrorResponse(401, "unauthorized");
      }

      return monthViewApiHandler(input);
    });

    renderApp(<App />);

    await waitForText("계정 데이터");
    await setInputValue("현재 비밀번호", "wrong-secret123");
    await setInputValue("새 비밀번호", "new-secret123");
    await submitForm();

    await waitForText("현재 비밀번호를 확인해 주세요.");

    expect(hasFetchedPath(
      fetchMock,
      (path) => path === "/api/auth/password/change",
    )).toBe(true);
  });

  it("logs out other sessions from the dashboard", async () => {
    const monthViewApiHandler = createMonthViewApiHandler();
    const fetchMock = stubFetch((input) => {
      if (requestPath(input) === "/api/auth/logout/others") {
        return okResponse();
      }

      return monthViewApiHandler(input);
    });

    renderApp(<App />);

    await waitForText("계정 데이터");
    await clickButton("다른 기기 로그아웃");

    await waitForText("다른 기기에서 로그아웃했습니다.");

    expect(hasFetchedPath(
      fetchMock,
      (path) => path === "/api/auth/logout/others",
    )).toBe(true);
    expect(document.body.textContent).toContain("계정 데이터");
  });

  it("retries the current month after falling back to sample data", async () => {
    let failedInitialLoad = false;
    let resolveRetryLoad: (() => void) | null = null;
    const fetchMock = stubFetch((input) => {
      if (!failedInitialLoad) {
        failedInitialLoad = true;
        return errorResponse(500);
      }

      return pendingJsonResponse(
        buildMonthView(monthFromRequest(input)),
        (resolve) => {
          resolveRetryLoad = resolve;
        },
      );
    });

    renderApp(<App />);

    await waitForText("샘플 데이터");
    const failedMonth = monthFromRequest(fetchMock.mock.calls[0][0]);

    expect(document.body.textContent).toContain(
      "데이터를 불러오지 못해 샘플 데이터를 표시합니다.",
    );
    expect(getButton("목표 이월").disabled).toBe(true);
    expect(getButton("목표 추가").disabled).toBe(true);

    await clickButton("다시 시도");

    await waitFor(() => fetchMock.mock.calls.length >= 2);

    expect(document.body.textContent).toContain("불러오는 중");
    expect(document.body.textContent).not.toContain(
      "데이터를 불러오지 못해 샘플 데이터를 표시합니다.",
    );
    expect(queryButton("다시 시도")).toBeNull();

    await resolvePending(resolveRetryLoad);

    await waitForText("계정 데이터");

    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(monthFromRequest(fetchMock.mock.calls[1][0])).toBe(failedMonth);
    expect(document.body.textContent).toContain("API walk");
    expect(document.body.textContent).not.toContain(
      "데이터를 불러오지 못해 샘플 데이터를 표시합니다.",
    );
  });

  it("reserves five daily goal columns when no goals exist", async () => {
    stubFetch(createMonthViewApiHandler({ goalCount: 0 }));

    renderApp(<App />);
    await waitForText("계정 데이터");

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
    await waitForText("계정 데이터");

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
    await waitForText("계정 데이터");

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
    await waitForText("계정 데이터");

    const initialMonth = monthFromRequest(fetchMock.mock.calls[0][0]);

    await clickButton("이전 달");

    await waitFor(() => fetchMock.mock.calls.length >= 2);

    expect(monthFromRequest(fetchMock.mock.calls[1][0])).toBe(
      offsetMonth(initialMonth, -1),
    );
  });

  it("keeps future goal completion disabled until the date arrives", async () => {
    const initialMonth = currentMonth();
    const futureMonth = offsetMonth(initialMonth, 1);
    const futureDate = monthStartDate(futureMonth);
    const fetchMock = stubFetch((input) => {
      const month = monthFromRequest(input);
      const goals: Goal[] =
        month === futureMonth
          ? [
              {
                id: 7,
                title: "Future completion",
                startDate: futureDate,
                endDate: null,
              },
            ]
          : [];

      return jsonResponse(buildMonthViewFromGoals(month, goals));
    });

    renderApp(<App />);
    await waitForText("계정 데이터");

    await clickButton("다음 달");
    await waitForText("Future completion");

    const completeButton = getButton(
      `${shortDate(futureDate)} Future completion 완료`,
    );
    expect(completeButton.disabled).toBe(true);

    await clickButton(`${shortDate(futureDate)} Future completion 완료`);

    expect(hasNoFetchedPath(fetchMock, (path) => path === "/api/checks")).toBe(
      true,
    );
  });

  it("shows feedback after carrying goals into the current month", async () => {
    const fetchMock = stubFetch(createMonthViewApiHandler());

    renderApp(<App />);
    await waitForText("계정 데이터");

    await clickButton("목표 이월");

    await waitForText("목표를 이월했습니다.");

    expect(hasFetchedPath(fetchMock, (path) => path.endsWith("/ensure"))).toBe(
      true,
    );
  });

  it("clears stale success feedback when validation fails", async () => {
    const fetchMock = stubFetch(createMonthViewApiHandler());

    renderApp(<App />);
    await waitForText("계정 데이터");

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
    await waitForText("계정 데이터");

    await setInputValue("05.01 메모", "saving memo");
    await blurInput("05.01 메모");

    await waitFor(() => getInput("05.01 메모").disabled);

    await resolvePending(resolveMemoSave);

    await waitFor(() => !getInput("05.01 메모").disabled);
  });

  it("shows pending affordances while goal mutations are saving", async () => {
    let resolveGoalCreate: (() => void) | null = null;
    let resolveGoalTitleSave: (() => void) | null = null;
    let resolveGoalDeactivate: (() => void) | null = null;
    const monthViewApiHandler = createMonthViewApiHandler();

    stubFetch((input) => {
      const path = requestPath(input);

      if (path.endsWith("/goals")) {
        return pendingResponse((resolve) => {
          resolveGoalCreate = resolve;
        });
      }

      if (path === "/api/goals/1") {
        return pendingResponse((resolve) => {
          resolveGoalTitleSave = resolve;
        });
      }

      if (path.includes("/api/goals/1/deactivate")) {
        return pendingResponse((resolve) => {
          resolveGoalDeactivate = resolve;
        });
      }

      return monthViewApiHandler(input);
    });

    renderApp(<App />);
    await waitForText("계정 데이터");

    await clickButton("목표 추가");
    await setInputValue("새 목표 제목", "API plan");
    await clickButton("목표 저장");

    await waitFor(() => getButton("목표 저장 중").disabled);
    expect(getButton("목표 저장 중").title).toBe("목표 저장 중");
    expect(getButton("목표 저장 중").querySelector(".animate-spin"))
      .toBeTruthy();

    await resolvePending(resolveGoalCreate);
    await waitFor(() => queryButton("목표 저장 중") === null);

    await clickButton("API walk 수정");
    await setInputValue("API walk 제목 수정", "API walk revised");
    await clickButton("API walk 저장");

    await waitFor(() => getButton("API walk 저장 중").disabled);
    expect(getButton("API walk 저장 중").title).toBe("목표 저장 중");
    expect(getButton("API walk 저장 중").querySelector(".animate-spin"))
      .toBeTruthy();

    await resolvePending(resolveGoalTitleSave);
    await waitFor(() => queryButton("API walk 저장 중") === null);

    await clickButton("API walk 종료");

    await waitFor(() => getButton("API walk 종료 중").disabled);
    expect(getButton("API walk 종료 중").title).toBe("목표 종료 중");
    expect(getButton("API walk 종료 중").querySelector(".animate-spin"))
      .toBeTruthy();

    await resolvePending(resolveGoalDeactivate);
    await waitFor(() => queryButton("API walk 종료 중") === null);
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
    await waitForText("계정 데이터");
    const loadedMonth = monthFromRequest(fetchMock.mock.calls[0][0]);

    await clickButton("API walk 종료");

    await waitForText("목표를 종료했습니다.");

    expect(hasFetchedPath(fetchMock, (path) =>
      path.includes("/api/goals/1/deactivate"),
    )).toBe(true);
    expect(queryButton("API walk 종료")).toBeNull();
    expect(getButton("API walk 이미 종료됨").disabled).toBe(true);
    expect(document.body.textContent).toContain("종료됨");
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
    await waitForText("계정 데이터");
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
    await waitForText("계정 데이터");

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
    const fetchMock = stubFetch(createMonthViewApiHandler({ goalCount: 5 }));

    renderApp(<App />);
    await waitForText("계정 데이터");

    await clickButton("목표 추가");
    expect(getButton("목표 저장").textContent).toContain("목표 저장");
    await setInputValue("새 목표 제목", "API overflow");
    await clickButton("목표 저장");

    await waitForText("할일은 날짜별로 최대 5개까지 등록할 수 있습니다.");
    expect(getInput("새 목표 제목").closest("form")?.textContent).toContain(
      "할일은 날짜별로 최대 5개까지 등록할 수 있습니다.",
    );
    expect(hasNoFetchedPath(fetchMock, (path) => path.endsWith("/goals"))).toBe(
      true,
    );
    expect(getInput("새 목표 제목").value).toBe("API overflow");
  });

  it("blocks blank goal title edits before calling the API", async () => {
    const fetchMock = stubFetch(createMonthViewApiHandler());

    renderApp(<App />);
    await waitForText("계정 데이터");

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
    await waitForText("계정 데이터");
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

    await clickButton("API plan 수정");
    await setInputValue("API plan 제목 수정", "API plan revised");
    await clickButton("API plan 저장");

    await waitForText("API plan revised");
    expect(document.body.textContent).toContain("API walk");
    expect(hasFetchedJsonBody<{ title: string }>(
      fetchMock,
      (path) => path === "/api/goals/3",
      (body) => body.title === "API plan revised",
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
    await waitForText("계정 데이터");
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
