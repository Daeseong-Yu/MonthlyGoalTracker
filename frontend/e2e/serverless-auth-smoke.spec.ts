import { expect, test, type APIRequestContext } from "@playwright/test";

const runAuthenticatedSmoke =
  process.env.PLAYWRIGHT_SERVERLESS_AUTH_SMOKE === "1";
const authEmail = process.env.PLAYWRIGHT_AUTH_EMAIL?.trim();
const authPassword = process.env.PLAYWRIGHT_AUTH_PASSWORD ?? "";
const authLocale = process.env.PLAYWRIGHT_AUTH_LOCALE === "en" ? "en" : "ko";

test.describe("serverless authenticated smoke", () => {
  test.skip(
    !runAuthenticatedSmoke,
    "Set PLAYWRIGHT_SERVERLESS_AUTH_SMOKE=1 to run authenticated serverless smoke.",
  );
  test.skip(
    runAuthenticatedSmoke && (!authEmail || !authPassword),
    "Set PLAYWRIGHT_AUTH_EMAIL and PLAYWRIGHT_AUTH_PASSWORD to run authenticated serverless smoke.",
  );

  test("persists goal, check, memo, and logout through CloudFront API", async ({
    request,
  }) => {
    const smokeDate = authSmokeDate();
    const month = smokeDate.slice(0, 7);
    const runLabel = `serverless-smoke-${Date.now()}`;
    const title = `${runLabel} goal`;
    const updatedTitle = `${runLabel} updated goal`;
    const memo = `${runLabel} memo`;
    let csrfToken: string | null = null;
    let goalID: number | null = null;
    let goalDeactivated = false;
    let loggedOut = false;

    try {
      const loginResponse = await request.post("/api/auth/login", {
        data: {
          email: authEmail,
          password: authPassword,
          locale: authLocale,
        },
      });
      expect(loginResponse.status(), "login status").toBe(200);
      expectLoginCookieSecurity(loginResponse);

      const authPayload = (await loginResponse.json()) as {
        csrfToken?: unknown;
      };
      expect(typeof authPayload.csrfToken).toBe("string");
      csrfToken = authPayload.csrfToken as string;

      const meResponse = await request.get("/api/auth/me");
      expect(meResponse.status(), "authenticated session status").toBe(200);

      const ensureResponse = await request.post(`/api/months/${month}/ensure`, {
        headers: csrfHeaders(csrfToken),
      });
      expect(ensureResponse.status(), "month ensure status").toBe(200);

      const createGoalResponse = await request.post(
        `/api/months/${month}/goals`,
        {
          data: {
            title,
            startDate: smokeDate,
          },
          headers: csrfHeaders(csrfToken),
        },
      );
      expect(createGoalResponse.status(), "goal create status").toBe(201);
      const goalPayload = (await createGoalResponse.json()) as {
        id?: unknown;
      };
      expect(typeof goalPayload.id).toBe("number");
      goalID = goalPayload.id as number;

      const updateGoalResponse = await request.patch(`/api/goals/${goalID}`, {
        data: {
          title: updatedTitle,
        },
        headers: csrfHeaders(csrfToken),
      });
      expect(updateGoalResponse.status(), "goal update status").toBe(200);

      const checkResponse = await request.put("/api/checks", {
        data: {
          goalId: goalID,
          date: smokeDate,
          completed: true,
        },
        headers: csrfHeaders(csrfToken),
      });
      expect(checkResponse.status(), "daily check status").toBe(200);

      const memoResponse = await request.put(`/api/memos/${smokeDate}`, {
        data: {
          memo,
        },
        headers: csrfHeaders(csrfToken),
      });
      expect(memoResponse.status(), "memo save status").toBe(200);

      const monthResponse = await request.get(`/api/months/${month}`);
      expect(monthResponse.status(), "month load status").toBe(200);
      const monthPayload = (await monthResponse.json()) as {
        goals?: Array<{ id: number; title: string }>;
        checks?: Array<{ goalId: number; date: string; completed: boolean }>;
        days?: Array<{ date: string; memo: string }>;
      };
      expect(
        monthPayload.goals?.some(
          (goal) => goal.id === goalID && goal.title === updatedTitle,
        ),
      ).toBe(true);
      expect(
        monthPayload.checks?.some(
          (check) =>
            check.goalId === goalID &&
            check.date === smokeDate &&
            check.completed === true,
        ),
      ).toBe(true);
      expect(
        monthPayload.days?.some(
          (day) => day.date === smokeDate && day.memo === memo,
        ),
      ).toBe(true);

      const deactivateResponse = await request.post(
        `/api/goals/${goalID}/deactivate`,
        {
          data: {
            endDate: smokeDate,
          },
          headers: csrfHeaders(csrfToken),
        },
      );
      expect(deactivateResponse.status(), "goal deactivate status").toBe(200);
      goalDeactivated = true;

      const logoutOthersResponse = await request.post("/api/auth/logout/others", {
        headers: csrfHeaders(csrfToken),
      });
      expect(logoutOthersResponse.status(), "logout other sessions status").toBe(204);

      const logoutResponse = await request.post("/api/auth/logout", {
        headers: csrfHeaders(csrfToken),
      });
      expect(logoutResponse.status(), "logout status").toBe(204);
      loggedOut = true;

      const afterLogoutResponse = await request.get("/api/auth/me");
      expect(afterLogoutResponse.status(), "protected API after logout status").toBe(401);
    } finally {
      await cleanupSmokeState(request, csrfToken, goalID, smokeDate, goalDeactivated, loggedOut);
    }
  });
});

function authSmokeDate(): string {
  const configuredDate = process.env.PLAYWRIGHT_AUTH_SMOKE_DATE?.trim();
  if (configuredDate) {
    if (!/^\d{4}-\d{2}-\d{2}$/.test(configuredDate)) {
      throw new Error("PLAYWRIGHT_AUTH_SMOKE_DATE must use YYYY-MM-DD format.");
    }

    return configuredDate;
  }

  return new Date().toISOString().slice(0, 10);
}

function csrfHeaders(csrfToken: string): Record<string, string> {
  return {
    "X-CSRF-Token": csrfToken,
  };
}

function expectLoginCookieSecurity(response: {
  headersArray(): Array<{ name: string; value: string }>;
}) {
  const cookies = response
    .headersArray()
    .filter((header) => header.name.toLowerCase() === "set-cookie")
    .map((header) => parseSetCookie(header.value));

  const sessionCookie = cookies.find((cookie) => cookie.name === "mgt_session");
  const csrfCookie = cookies.find((cookie) => cookie.name === "mgt_csrf");

  expect(sessionCookie, "session cookie is set").toBeDefined();
  expect(csrfCookie, "csrf cookie is set").toBeDefined();

  expect(sessionCookie?.attributes.has("secure"), "session cookie is Secure").toBe(
    true,
  );
  expect(sessionCookie?.attributes.has("httponly"), "session cookie is HttpOnly").toBe(
    true,
  );
  expect(
    sessionCookie?.attributes.get("samesite"),
    "session cookie SameSite",
  ).toBe("lax");

  expect(csrfCookie?.attributes.has("secure"), "csrf cookie is Secure").toBe(
    true,
  );
  expect(
    csrfCookie?.attributes.has("httponly"),
    "csrf cookie is readable by frontend",
  ).toBe(false);
  expect(csrfCookie?.attributes.get("samesite"), "csrf cookie SameSite").toBe(
    "lax",
  );
}

function parseSetCookie(headerValue: string): {
  name: string;
  attributes: Map<string, string | true>;
} {
  const [nameValue, ...rawAttributes] = headerValue
    .split(";")
    .map((part) => part.trim());
  const [name] = nameValue.split("=", 1);
  const attributes = new Map<string, string | true>();

  for (const attribute of rawAttributes) {
    const [rawName, ...rawValue] = attribute.split("=");
    attributes.set(
      rawName.toLowerCase(),
      rawValue.length === 0 ? true : rawValue.join("=").toLowerCase(),
    );
  }

  return { name, attributes };
}

async function cleanupSmokeState(
  request: APIRequestContext,
  csrfToken: string | null,
  goalID: number | null,
  smokeDate: string,
  goalDeactivated: boolean,
  loggedOut: boolean,
) {
  if (csrfToken === null) {
    return;
  }

  if (goalID !== null && !goalDeactivated) {
    try {
      await request.post(`/api/goals/${goalID}/deactivate`, {
        data: {
          endDate: smokeDate,
        },
        headers: csrfHeaders(csrfToken),
      });
    } catch {
      // Best-effort cleanup only; the original smoke failure is more useful.
    }
  }

  if (!loggedOut) {
    try {
      await request.post("/api/auth/logout", {
        headers: csrfHeaders(csrfToken),
      });
    } catch {
      // Best-effort cleanup only; the original smoke failure is more useful.
    }
  }
}
