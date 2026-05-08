import type {
  AppLocale,
  AuthResponse,
  BootstrapResponse,
  MonthView,
  UserSession,
} from "./types";

const apiBaseURL = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");
let csrfToken: string | null = null;

export async function bootstrapSession(): Promise<BootstrapResponse> {
  const response = await requestJSON<BootstrapResponse>(
    "/api/bootstrap",
    "bootstrap request",
  );
  applyCSRFToken(response.csrfToken);

  return {
    ...response,
    authenticated: response.authenticated ?? response.user !== null,
    user: response.user ?? null,
  };
}

export async function login(
  email: string,
  password: string,
  locale: AppLocale,
): Promise<AuthResponse> {
  const response = await requestJSON<AuthResponse>(
    "/api/auth/login",
    "login request",
    {
      method: "POST",
      body: { email, password, locale },
    },
  );
  applyCSRFToken(response.csrfToken);
  return response;
}

export async function signUp(
  email: string,
  password: string,
  locale: AppLocale,
  legacyClaimToken?: string,
): Promise<AuthResponse> {
  const claimToken = legacyClaimToken?.trim();
  const response = await requestJSON<AuthResponse>(
    "/api/auth/signup",
    "signup request",
    {
      method: "POST",
      body: {
        email,
        password,
        locale,
        ...(claimToken ? { claimToken } : {}),
      },
    },
  );
  applyCSRFToken(response.csrfToken);
  return response;
}

export async function logoutSession(): Promise<void> {
  await requestVoid("/api/auth/logout", "logout request", { method: "POST" });
  clearAuthCSRFToken();
}

export async function updateUserLocale(
  locale: AppLocale,
): Promise<UserSession> {
  return requestJSON<UserSession>("/api/auth/me/locale", "locale update request", {
    method: "PATCH",
    body: { locale },
  });
}

export function clearAuthCSRFToken() {
  csrfToken = null;
}

export async function getMonthView(month: string): Promise<MonthView> {
  return requestJSON<MonthView>(
    `/api/months/${encodeURIComponent(month)}`,
    "month request",
  );
}

export async function ensureMonth(month: string): Promise<MonthView> {
  return requestJSON<MonthView>(
    `/api/months/${encodeURIComponent(month)}/ensure`,
    "month ensure request",
    { method: "POST" },
  );
}

export async function createGoal(
  month: string,
  title: string,
  startDate: string,
): Promise<void> {
  await requestVoid(
    `/api/months/${encodeURIComponent(month)}/goals`,
    "goal request",
    { method: "POST", body: { title, startDate } },
  );
}

export async function updateGoalTitle(
  goalId: number,
  title: string,
): Promise<void> {
  await requestVoid(`/api/goals/${goalId}`, "goal update request", {
    method: "PATCH",
    body: { title },
  });
}

export async function deactivateGoal(
  goalId: number,
  endDate: string,
): Promise<void> {
  await requestVoid(`/api/goals/${goalId}/deactivate`, "goal deactivate request", {
    method: "POST",
    body: { endDate },
  });
}

export async function setGoalCompleted(
  goalId: number,
  date: string,
  completed: boolean,
): Promise<void> {
  await requestVoid("/api/checks", "check request", {
    method: "PUT",
    body: { goalId, date, completed },
  });
}

export async function saveMemo(date: string, memo: string): Promise<void> {
  await requestVoid(
    `/api/memos/${encodeURIComponent(date)}`,
    "memo request",
    { method: "PUT", body: { memo } },
  );
}

type RequestOptions = {
  body?: unknown;
  method?: string;
};

async function requestJSON<T>(
  path: string,
  errorLabel: string,
  options?: RequestOptions,
): Promise<T> {
  const response = await request(path, errorLabel, options);
  return (await response.json()) as T;
}

async function requestVoid(
  path: string,
  errorLabel: string,
  options?: RequestOptions,
): Promise<void> {
  await request(path, errorLabel, options);
}

async function request(
  path: string,
  errorLabel: string,
  options?: RequestOptions,
) {
  const response = await fetch(`${apiBaseURL}${path}`, requestInit(options));
  if (!response.ok) {
    throw new Error(`${errorLabel} failed with status ${response.status}`);
  }

  return response;
}

function requestInit(options?: RequestOptions): RequestInit {
  const method = options?.method;
  const headers: Record<string, string> = {
    Accept: "application/json",
  };
  const init: RequestInit = {
    credentials: "include",
    headers,
  };

  if (method) {
    init.method = method;
  }

  if (method && isUnsafeMethod(method) && csrfToken) {
    headers["X-CSRF-Token"] = csrfToken;
  }

  if (options && "body" in options) {
    headers["Content-Type"] = "application/json";
    init.body = JSON.stringify(options.body);
  }

  return init;
}

function applyCSRFToken(token: string | null | undefined) {
  csrfToken = token ?? null;
}

function isUnsafeMethod(method: string) {
  return !["GET", "HEAD", "OPTIONS"].includes(method.toUpperCase());
}
