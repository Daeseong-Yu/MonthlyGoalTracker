import { afterEach, describe, expect, it, vi } from "vitest";

import {
  bootstrapSession,
  clearAuthCSRFToken,
  createGoal,
  deactivateGoal,
  ensureMonth,
  getMonthView,
  login,
  logoutSession,
  saveMemo,
  setGoalCompleted,
  signUp,
  updateGoalTitle,
  updateUserLocale,
  verifyEmail,
} from "./api";
import type { MonthView, UserSession } from "./types";

const monthView: MonthView = {
  month: "2026-05",
  goals: [],
  days: [],
  checks: [],
  chart: [],
};

const userSession: UserSession = {
  id: 3,
  email: "tester@example.com",
  locale: "ko",
  createdAt: "2026-05-01T00:00:00Z",
};

describe("api client", () => {
  afterEach(() => {
    clearAuthCSRFToken();
    vi.unstubAllGlobals();
  });

  it("loads a month view with an encoded month", async () => {
    const fetchMock = stubFetch(jsonResponse(monthView));

    await expect(getMonthView("2026 05")).resolves.toEqual(monthView);

    expect(fetchMock).toHaveBeenCalledWith("/api/months/2026%2005", {
      credentials: "include",
      headers: {
        Accept: "application/json",
      },
    });
  });

  it("bootstraps the current session and stores CSRF for unsafe requests", async () => {
    const fetchMock = stubFetch(
      jsonResponse({
        authenticated: true,
        locale: "ko",
        csrfToken: "csrf-bootstrap",
        user: userSession,
      }),
    );

    await expect(bootstrapSession()).resolves.toEqual({
      authenticated: true,
      locale: "ko",
      csrfToken: "csrf-bootstrap",
      user: userSession,
    });

    expect(fetchMock).toHaveBeenNthCalledWith(1, "/api/bootstrap", {
      credentials: "include",
      headers: {
        Accept: "application/json",
      },
    });

    fetchMock.mockResolvedValueOnce(okResponse());
    await createGoal("2026-05", "Read", "2026-05-01");

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/months/2026-05/goals",
      {
        method: "POST",
        credentials: "include",
        headers: jsonHeaders("csrf-bootstrap"),
        body: JSON.stringify({ title: "Read", startDate: "2026-05-01" }),
      },
    );
  });

  it("logs in and sends the saved CSRF token when changing locale", async () => {
    const fetchMock = stubFetch(
      jsonResponse({
        user: userSession,
        csrfToken: "csrf-login",
        locale: "ko",
      }),
    );

    await expect(login("tester@example.com", "secret123", "ko")).resolves.toEqual(
      {
        user: userSession,
        csrfToken: "csrf-login",
        locale: "ko",
      },
    );

    expect(fetchMock).toHaveBeenNthCalledWith(1, "/api/auth/login", {
      method: "POST",
      credentials: "include",
      headers: jsonHeaders(),
      body: JSON.stringify({
        email: "tester@example.com",
        password: "secret123",
        locale: "ko",
      }),
    });

    const updatedUser = { ...userSession, locale: "en" as const };
    fetchMock.mockResolvedValueOnce(jsonResponse(updatedUser));

    await expect(updateUserLocale("en")).resolves.toEqual(updatedUser);

    expect(fetchMock).toHaveBeenNthCalledWith(2, "/api/auth/me/locale", {
      method: "PATCH",
      credentials: "include",
      headers: jsonHeaders("csrf-login"),
      body: JSON.stringify({ locale: "en" }),
    });
  });

  it("signs up with the selected locale", async () => {
    const fetchMock = stubFetch(
      jsonResponse({
        user: { ...userSession, locale: "en" },
        csrfToken: "csrf-signup",
        locale: "en",
      }),
    );

    await expect(signUp("new@example.com", "secret123", "en")).resolves.toEqual({
      user: { ...userSession, locale: "en" },
      csrfToken: "csrf-signup",
      locale: "en",
    });

    expect(fetchMock).toHaveBeenCalledWith("/api/auth/signup", {
      method: "POST",
      credentials: "include",
      headers: jsonHeaders(),
      body: JSON.stringify({
        email: "new@example.com",
        password: "secret123",
        locale: "en",
      }),
    });
  });

  it("keeps signup accepted responses out of the CSRF session", async () => {
    const fetchMock = stubFetch(
      jsonResponse({
        status: "verification_required",
        locale: "ko",
      }),
    );

    await expect(signUp("new@example.com", "secret123", "ko")).resolves.toEqual({
      status: "verification_required",
      locale: "ko",
    });

    expect(fetchMock).toHaveBeenNthCalledWith(1, "/api/auth/signup", {
      method: "POST",
      credentials: "include",
      headers: jsonHeaders(),
      body: JSON.stringify({
        email: "new@example.com",
        password: "secret123",
        locale: "ko",
      }),
    });

    fetchMock.mockResolvedValueOnce(okResponse());
    await createGoal("2026-05", "Read", "2026-05-01");

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/months/2026-05/goals",
      {
        method: "POST",
        credentials: "include",
        headers: jsonHeaders(),
        body: JSON.stringify({ title: "Read", startDate: "2026-05-01" }),
      },
    );
  });

  it("verifies email and stores CSRF for later unsafe requests", async () => {
    const fetchMock = stubFetch(
      jsonResponse({
        user: userSession,
        csrfToken: "csrf-verified",
        locale: "ko",
      }),
    );

    await expect(verifyEmail("email-token")).resolves.toEqual({
      user: userSession,
      csrfToken: "csrf-verified",
      locale: "ko",
    });

    expect(fetchMock).toHaveBeenNthCalledWith(1, "/api/auth/verify-email", {
      method: "POST",
      credentials: "include",
      headers: jsonHeaders(),
      body: JSON.stringify({ token: "email-token" }),
    });

    fetchMock.mockResolvedValueOnce(okResponse());
    await createGoal("2026-05", "Read", "2026-05-01");

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/months/2026-05/goals",
      {
        method: "POST",
        credentials: "include",
        headers: jsonHeaders("csrf-verified"),
        body: JSON.stringify({ title: "Read", startDate: "2026-05-01" }),
      },
    );
  });

  it("sends a trimmed legacy claim token during signup", async () => {
    const fetchMock = stubFetch(
      jsonResponse({
        user: { ...userSession, locale: "ko" },
        csrfToken: "csrf-signup",
        locale: "ko",
      }),
    );

    await expect(
      signUp("owner@example.com", "secret123", "ko", " owner-token-123 "),
    ).resolves.toEqual({
      user: { ...userSession, locale: "ko" },
      csrfToken: "csrf-signup",
      locale: "ko",
    });

    expect(fetchMock).toHaveBeenCalledWith("/api/auth/signup", {
      method: "POST",
      credentials: "include",
      headers: jsonHeaders(),
      body: JSON.stringify({
        email: "owner@example.com",
        password: "secret123",
        locale: "ko",
        claimToken: "owner-token-123",
      }),
    });
  });

  it("clears the saved CSRF token after logout", async () => {
    const fetchMock = stubFetch(
      jsonResponse({
        authenticated: true,
        locale: "ko",
        csrfToken: "csrf-bootstrap",
        user: userSession,
      }),
    );

    await bootstrapSession();

    fetchMock.mockResolvedValueOnce(okResponse());
    await logoutSession();

    expect(fetchMock).toHaveBeenNthCalledWith(2, "/api/auth/logout", {
      method: "POST",
      credentials: "include",
      headers: {
        Accept: "application/json",
        "X-CSRF-Token": "csrf-bootstrap",
      },
    });

    fetchMock.mockResolvedValueOnce(okResponse());
    await deactivateGoal(7, "2026-05-03");

    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      "/api/goals/7/deactivate",
      {
        method: "POST",
        credentials: "include",
        headers: jsonHeaders(),
        body: JSON.stringify({ endDate: "2026-05-03" }),
      },
    );
  });

  it("prepares a month with a POST request", async () => {
    const fetchMock = stubFetch(jsonResponse(monthView));

    await expect(ensureMonth("2026-05")).resolves.toEqual(monthView);

    expect(fetchMock).toHaveBeenCalledWith("/api/months/2026-05/ensure", {
      method: "POST",
      credentials: "include",
      headers: {
        Accept: "application/json",
      },
    });
  });

  it("creates a goal with JSON body", async () => {
    const fetchMock = stubFetch(okResponse());

    await createGoal("2026-05", "Read", "2026-05-01");

    expect(fetchMock).toHaveBeenCalledWith("/api/months/2026-05/goals", {
      method: "POST",
      credentials: "include",
      headers: jsonHeaders(),
      body: JSON.stringify({ title: "Read", startDate: "2026-05-01" }),
    });
  });

  it("updates a goal title with PATCH", async () => {
    const fetchMock = stubFetch(okResponse());

    await updateGoalTitle(7, "Updated");

    expect(fetchMock).toHaveBeenCalledWith("/api/goals/7", {
      method: "PATCH",
      credentials: "include",
      headers: jsonHeaders(),
      body: JSON.stringify({ title: "Updated" }),
    });
  });

  it("deactivates a goal with an end date", async () => {
    const fetchMock = stubFetch(okResponse());

    await deactivateGoal(7, "2026-05-03");

    expect(fetchMock).toHaveBeenCalledWith("/api/goals/7/deactivate", {
      method: "POST",
      credentials: "include",
      headers: jsonHeaders(),
      body: JSON.stringify({ endDate: "2026-05-03" }),
    });
  });

  it("saves a check with completed false", async () => {
    const fetchMock = stubFetch(okResponse());

    await setGoalCompleted(7, "2026-05-03", false);

    expect(fetchMock).toHaveBeenCalledWith("/api/checks", {
      method: "PUT",
      credentials: "include",
      headers: jsonHeaders(),
      body: JSON.stringify({
        goalId: 7,
        date: "2026-05-03",
        completed: false,
      }),
    });
  });

  it("saves memo text with an encoded date", async () => {
    const fetchMock = stubFetch(okResponse());

    await saveMemo("2026 05 03", "memo");

    expect(fetchMock).toHaveBeenCalledWith("/api/memos/2026%2005%2003", {
      method: "PUT",
      credentials: "include",
      headers: jsonHeaders(),
      body: JSON.stringify({ memo: "memo" }),
    });
  });

  it("throws when an API response is not ok", async () => {
    stubFetch(errorResponse(500));

    await expect(getMonthView("2026-05")).rejects.toThrow(
      "month request failed with status 500",
    );
  });

  it("preserves API error status and server code", async () => {
    stubFetch(jsonErrorResponse(429, "too many requests"));

    let caughtError: unknown;
    try {
      await login("tester@example.com", "secret123", "ko");
    } catch (error) {
      caughtError = error;
    }

    expect(caughtError).toMatchObject({
      code: "too many requests",
      status: 429,
    });
    expect(caughtError).toBeInstanceOf(Error);
    expect((caughtError as Error).message).toBe(
      "login request failed with status 429: too many requests",
    );
  });
});

function stubFetch(response: Response) {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(response);
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function okResponse() {
  return new Response(null, { status: 204 });
}

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
    },
  });
}

function jsonErrorResponse(status: number, error: string) {
  return new Response(JSON.stringify({ error }), {
    status,
    headers: {
      "Content-Type": "application/json",
    },
  });
}

function errorResponse(status: number) {
  return new Response(null, { status });
}

function jsonHeaders(csrfToken?: string) {
  return {
    Accept: "application/json",
    "Content-Type": "application/json",
    ...(csrfToken ? { "X-CSRF-Token": csrfToken } : {}),
  };
}
