export const publicApiPaths = [
  "GET /api/health",
  "GET /api/bootstrap",
  "POST /api/auth/signup",
  "POST /api/auth/login",
  "POST /api/auth/verify-email",
  "POST /api/auth/password-reset/request",
  "POST /api/auth/password-reset/confirm",
] as const;

export const protectedApiPaths = [
  "GET /api/auth/me",
  "POST /api/auth/logout",
  "POST /api/auth/logout/others",
  "PATCH /api/auth/me/locale",
  "POST /api/auth/password/change",
  "POST /api/months/:month/ensure",
  "GET /api/months/:month",
  "POST /api/months/:month/goals",
  "PATCH /api/goals/:id",
  "POST /api/goals/:id/deactivate",
  "PUT /api/memos/:date",
  "PUT /api/checks",
] as const;

export const serverlessInvariants = [
  "CloudFront serves frontend and /api plus /api/* under one site boundary.",
  "Public/protected API paths remain compatible with the EC2 deployment.",
  "Session tokens remain HttpOnly cookies.",
  "Unsafe HTTP methods require CSRF tokens.",
  "Anonymous preview mode remains frontend-only.",
  "PostgreSQL remains the system of record.",
] as const;
