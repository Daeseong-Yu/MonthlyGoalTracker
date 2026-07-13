import type {
  AppLocale,
  AuthResponse,
  BootstrapResponse,
  MonthView,
  PasswordResetAcceptedResponse,
  SignupResponse,
  UserSession,
} from "./types";

const apiBaseURL = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");
let csrfToken: string | null = null;

export class APIError extends Error {
  readonly code: string | null;
  readonly status: number;

  constructor(errorLabel: string, status: number, code: string | null) {
    super(`${errorLabel} failed with status ${status}${code ? `: ${code}` : ""}`);
    this.name = "APIError";
    this.code = code;
    this.status = status;
  }
}

export function isAPIError(error: unknown): error is APIError {
  return error instanceof APIError;
}

export function isAuthResponse(response: SignupResponse): response is AuthResponse {
  return "csrfToken" in response && "user" in response;
}

export async function checkHealth(): Promise<void> {
  await requestVoid("/api/health", "health request");
}

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
): Promise<SignupResponse> {
  const claimToken = legacyClaimToken?.trim();
  const response = await requestJSON<SignupResponse>(
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
  if (isAuthResponse(response)) {
    applyCSRFToken(response.csrfToken);
  }
  return response;
}

export async function verifyEmail(token: string): Promise<AuthResponse> {
  const response = await requestJSON<AuthResponse>(
    "/api/auth/verify-email",
    "email verification request",
    {
      method: "POST",
      body: { token },
    },
  );
  applyCSRFToken(response.csrfToken);
  return response;
}

export async function requestPasswordReset(
  email: string,
  locale: AppLocale,
): Promise<PasswordResetAcceptedResponse> {
  return requestJSON<PasswordResetAcceptedResponse>(
    "/api/auth/password-reset/request",
    "password reset request",
    {
      method: "POST",
      body: { email, locale },
    },
  );
}

export async function resetPassword(
  token: string,
  password: string,
): Promise<AuthResponse> {
  const response = await requestJSON<AuthResponse>(
    "/api/auth/password-reset/confirm",
    "password reset confirmation request",
    {
      method: "POST",
      body: { token, password },
    },
  );
  applyCSRFToken(response.csrfToken);
  return response;
}

export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<AuthResponse> {
  const response = await requestJSON<AuthResponse>(
    "/api/auth/password/change",
    "password change request",
    {
      method: "POST",
      body: { currentPassword, newPassword },
    },
  );
  applyCSRFToken(response.csrfToken);
  return response;
}

export async function logoutSession(): Promise<void> {
  await requestVoid("/api/auth/logout", "logout request", { method: "POST" });
  clearAuthCSRFToken();
}

export async function logoutOtherSessions(): Promise<void> {
  await requestVoid("/api/auth/logout/others", "other sessions logout request", {
    method: "POST",
  });
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
    throw await toAPIError(response, errorLabel);
  }

  return response;
}

async function toAPIError(response: Response, errorLabel: string): Promise<APIError> {
  const contentType = response.headers.get("Content-Type") ?? "";
  if (contentType.includes("application/json")) {
    try {
      const body = (await response.json()) as { error?: unknown };
      if (typeof body.error === "string") {
        return new APIError(errorLabel, response.status, body.error);
      }
    } catch {
      // Preserve the old status-only behavior when a failed JSON response is invalid.
    }
  }

  return new APIError(errorLabel, response.status, null);
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
