import {
  type FormEvent,
  type MouseEvent as ReactMouseEvent,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  ArrowLeft,
  CalendarDays,
  CalendarPlus,
  ChevronLeft,
  ChevronRight,
  KeyRound,
  LayoutDashboard,
  ListChecks,
  LogIn,
  LogOut,
  MailCheck,
  Monitor,
  Moon,
  RefreshCw,
  ShieldCheck,
  Sun,
  Target,
  UserPlus,
  X,
} from "lucide-react";

import {
  bootstrapSession,
  changePassword,
  checkHealth,
  clearAuthCSRFToken,
  isAPIError,
  isAuthResponse,
  login as loginSession,
  logoutOtherSessions,
  logoutSession,
  requestPasswordReset,
  resetPassword,
  signUp as signUpSession,
  updateUserLocale,
  verifyEmail,
} from "./api";
import { formatMonth, statusLabel } from "./appDisplay";
import ChartPanel from "./ChartPanel";
import DailyRecordTable from "./DailyRecordTable";
import GoalPanel from "./GoalPanel";
import { messagesForLocale, normalizeLocale, type AppMessages } from "./i18n";
import MetricSummary from "./MetricSummary";
import { useMonthController } from "./useMonthController";
import type {
  AppLocale,
  AuthResponse,
  ResolvedTheme,
  ThemePreference,
  UserSession,
} from "./types";

type AuthMode = "login" | "signup" | "forgot" | "reset";
type DashboardSection = "dashboard" | "goals" | "records" | "account";
type HealthStatus = "checking" | "healthy" | "unhealthy";
const localeStorageKey = "monthlyGoalTracker.locale";
const themeStorageKey = "monthlyGoalTracker.theme";

export default function App() {
  const [locale, setLocale] = useState<AppLocale>(
    () => readStoredLocale() ?? "ko",
  );
  const [user, setUser] = useState<UserSession | null>(null);
  const [bootstrapped, setBootstrapped] = useState(false);
  const [bootstrapFailed, setBootstrapFailed] = useState(false);
  const [healthStatus, setHealthStatus] = useState<HealthStatus>("checking");
  const [emailVerificationFailed, setEmailVerificationFailed] = useState(false);
  const [passwordResetToken, setPasswordResetToken] = useState<string | null>(
    () => readPasswordResetToken(),
  );
  const [showAuthScreen, setShowAuthScreen] = useState(false);
  const [themePreference, setThemePreference] = useState<ThemePreference>(
    () => readStoredThemePreference(),
  );
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>(() =>
    resolveThemePreference(readStoredThemePreference()),
  );
  const messages = useMemo(() => messagesForLocale(locale), [locale]);

  useEffect(() => {
    const mediaQuery = window.matchMedia?.("(prefers-color-scheme: dark)");
    const applyResolvedTheme = () =>
      setResolvedTheme(resolveThemePreference(themePreference));

    applyResolvedTheme();

    if (themePreference !== "system" || !mediaQuery) {
      return;
    }

    mediaQuery.addEventListener("change", applyResolvedTheme);
    return () => mediaQuery.removeEventListener("change", applyResolvedTheme);
  }, [themePreference]);

  useEffect(() => {
    document.documentElement.dataset.theme = resolvedTheme;
  }, [resolvedTheme]);

  useEffect(() => {
    let cancelled = false;

    void checkHealth()
      .then(() => {
        if (!cancelled) {
          setHealthStatus("healthy");
        }
      })
      .catch(() => {
        if (!cancelled) {
          setHealthStatus("unhealthy");
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function bootstrap() {
      try {
        const response = await bootstrapSession();
        if (cancelled) {
          return;
        }

        const nextUser = response.authenticated ? response.user : null;
        const nextLocale = normalizeLocale(
          nextUser?.locale ?? readStoredLocale() ?? response.locale,
        );
        setLocale(nextLocale);
        storeLocale(nextLocale);
        const verificationToken = nextUser ? null : readEmailVerificationToken();
        const resetToken = nextUser ? null : readPasswordResetToken();

        if (nextUser && readEmailVerificationToken()) {
          removeEmailVerificationToken();
        }

        if (nextUser && readPasswordResetToken()) {
          removePasswordResetToken();
        }

        if (verificationToken) {
          try {
            const authResponse = await verifyEmail(verificationToken);
            if (cancelled) {
              return;
            }

            removeEmailVerificationToken();
            removePasswordResetToken();
            const verifiedLocale = normalizeLocale(
              authResponse.user.locale ?? authResponse.locale,
            );
            setLocale(verifiedLocale);
            storeLocale(verifiedLocale);
            setUser(authResponse.user);
            setPasswordResetToken(null);
            setEmailVerificationFailed(false);
            setBootstrapFailed(false);
            return;
          } catch {
            if (cancelled) {
              return;
            }

            clearAuthCSRFToken();
            removeEmailVerificationToken();
            setUser(null);
            setPasswordResetToken(resetToken);
            setEmailVerificationFailed(true);
            setBootstrapFailed(false);
            return;
          }
        }

        setUser(nextUser);
        setPasswordResetToken(resetToken);
        setEmailVerificationFailed(false);
        setBootstrapFailed(false);
      } catch {
        if (cancelled) {
          return;
        }

        clearAuthCSRFToken();
        setUser(null);
        setPasswordResetToken(readPasswordResetToken());
        setEmailVerificationFailed(false);
        setBootstrapFailed(true);
      } finally {
        if (!cancelled) {
          setBootstrapped(true);
        }
      }
    }

    void bootstrap();

    return () => {
      cancelled = true;
    };
  }, []);

  async function handleLocaleChange(nextLocale: AppLocale) {
    if (nextLocale === locale) {
      return;
    }

    setLocale(nextLocale);
    storeLocale(nextLocale);
    if (!user) {
      return;
    }

    setUser({ ...user, locale: nextLocale });
    try {
      const updatedUser = await updateUserLocale(nextLocale);
      setUser(updatedUser);
    } catch {
      setUser((currentUser) =>
        currentUser === null ? currentUser : { ...currentUser, locale: nextLocale },
      );
    }
  }

  function handleThemePreferenceChange(nextPreference: ThemePreference) {
    setThemePreference(nextPreference);
    storeThemePreference(nextPreference);
  }

  function handleAuthenticated(response: AuthResponse) {
    const nextLocale = normalizeLocale(response.user.locale ?? response.locale);
    setLocale(nextLocale);
    storeLocale(nextLocale);
    setUser(response.user);
    setPasswordResetToken(null);
    setShowAuthScreen(false);
    removeEmailVerificationToken();
    removePasswordResetToken();
    setEmailVerificationFailed(false);
    setBootstrapFailed(false);
  }

  function handleLoggedOut() {
    clearAuthCSRFToken();
    setUser(null);
    setPasswordResetToken(readPasswordResetToken());
    setShowAuthScreen(false);
    setEmailVerificationFailed(false);
  }

  function handleCloseAuthFlow() {
    setShowAuthScreen(false);
    setEmailVerificationFailed(false);
    if (passwordResetToken !== null) {
      setPasswordResetToken(null);
      removePasswordResetToken();
    }
  }

  if (!bootstrapped) {
    return <BootstrapScreen messages={messages} />;
  }

  const authFlowOpen =
    !user && (showAuthScreen || passwordResetToken !== null || emailVerificationFailed);

  return (
    <>
      <Dashboard
        key={user ? "server" : "preview"}
        healthStatus={healthStatus}
        locale={locale}
        messages={messages}
        user={user}
        onAuthenticated={handleAuthenticated}
        onLocaleChange={(nextLocale) => void handleLocaleChange(nextLocale)}
        themePreference={themePreference}
        onThemePreferenceChange={handleThemePreferenceChange}
        onOpenAuth={() => setShowAuthScreen(true)}
        onLoggedOut={handleLoggedOut}
      />
      {authFlowOpen ? (
        <AuthScreen
          bootstrapError={bootstrapFailed ? messages.app.bootstrapError : null}
          verificationError={
            emailVerificationFailed ? messages.auth.emailVerificationFailed : null
          }
          locale={locale}
          messages={messages}
          passwordResetToken={passwordResetToken}
          onAuthenticated={handleAuthenticated}
          onLocaleChange={(nextLocale) => void handleLocaleChange(nextLocale)}
          onPasswordResetTokenConsumed={() => {
            setPasswordResetToken(null);
            removePasswordResetToken();
          }}
          onPreviewRequested={handleCloseAuthFlow}
        />
      ) : null}
    </>
  );
}

function BootstrapScreen({ messages }: { messages: AppMessages }) {
  return (
    <main className="app-root flex min-h-screen items-center justify-center px-4">
      <p
        className="panel-card px-4 py-3 text-sm font-semibold"
        role="status"
      >
        {messages.app.bootstrapLoading}
      </p>
    </main>
  );
}

function AuthScreen({
  bootstrapError,
  verificationError,
  locale,
  messages,
  passwordResetToken,
  onAuthenticated,
  onLocaleChange,
  onPasswordResetTokenConsumed,
  onPreviewRequested,
}: {
  bootstrapError: string | null;
  verificationError: string | null;
  locale: AppLocale;
  messages: AppMessages;
  passwordResetToken: string | null;
  onAuthenticated: (response: AuthResponse) => void;
  onLocaleChange: (locale: AppLocale) => void;
  onPasswordResetTokenConsumed: () => void;
  onPreviewRequested?: () => void;
}) {
  const [mode, setMode] = useState<AuthMode>(
    () => (passwordResetToken ? "reset" : "login"),
  );
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [legacyClaimToken, setLegacyClaimToken] = useState("");
  const [legacyClaimRequired, setLegacyClaimRequired] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const authMessages = messages.auth;
  const submitLabel = authSubmitLabel(mode, authMessages);
  const submitIcon = authSubmitIcon(mode);

  useEffect(() => {
    if (!passwordResetToken) {
      return;
    }

    setMode("reset");
    setEmail("");
    setPassword("");
    setLegacyClaimToken("");
    setLegacyClaimRequired(false);
    setError(null);
    setStatus(null);
  }, [passwordResetToken]);

  useEffect(() => {
    if (!onPreviewRequested) {
      return;
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onPreviewRequested?.();
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onPreviewRequested]);

  function moveToMode(nextMode: AuthMode) {
    if (mode === "reset") {
      onPasswordResetTokenConsumed();
    }
    setMode(nextMode);
    setError(null);
    setStatus(null);
    setPassword("");
    setLegacyClaimToken("");
    setLegacyClaimRequired(false);
  }

  function returnToLogin() {
    moveToMode("login");
  }

  function handleBackdropClick(event: ReactMouseEvent<HTMLDivElement>) {
    if (event.target === event.currentTarget) {
      onPreviewRequested?.();
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    setStatus(null);

    try {
      if (mode === "login") {
        const response = await loginSession(email, password, locale);
        onAuthenticated(response);
        return;
      }

      if (mode === "forgot") {
        await requestPasswordReset(email, locale);
        setStatus(authMessages.passwordResetRequested);
        setPassword("");
        setLegacyClaimToken("");
        return;
      }

      if (mode === "reset") {
        if (!passwordResetToken) {
          setError(authMessages.passwordResetTokenFailed);
          return;
        }

        const response = await resetPassword(passwordResetToken, password);
        onPasswordResetTokenConsumed();
        onAuthenticated(response);
        return;
      }

      const response = await signUpSession(
        email,
        password,
        locale,
        legacyClaimToken,
      );
      if (isAuthResponse(response)) {
        onAuthenticated(response);
        return;
      }

      setStatus(authMessages.signupAccepted);
      setPassword("");
      setLegacyClaimToken("");
      setLegacyClaimRequired(false);
    } catch (error) {
      if (
        mode === "signup" &&
        isAPIError(error) &&
        error.code === "legacy claim required"
      ) {
        setLegacyClaimRequired(true);
      }
      setError(authErrorMessage(error, mode, authMessages));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/55 px-4 py-8 sm:items-center"
      onClick={handleBackdropClick}
    >
      <section
        className="panel-card panel-card-padded max-h-[calc(100vh-2rem)] w-full max-w-md overflow-y-auto"
        role="dialog"
        aria-modal="true"
        aria-labelledby="auth-dialog-title"
      >
        <div className="flex flex-col gap-4">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <h1
                id="auth-dialog-title"
                className="text-2xl font-semibold tracking-normal" style={{ color: "var(--text-primary)" }}
              >
                {authMessages.title}
              </h1>
              <p className="mt-1 text-sm" style={{ color: "var(--text-secondary)" }}>
                {authMessages.subtitle}
              </p>
            </div>
            {onPreviewRequested ? (
              <button
                className="icon-button shrink-0"
                type="button"
                aria-label={authMessages.previewBackButton}
                title={authMessages.previewBackButton}
                onClick={onPreviewRequested}
              >
                <X size={18} />
              </button>
            ) : null}
          </div>

          <LanguageToggle
            label={messages.app.languageLabel}
            locale={locale}
            onChange={onLocaleChange}
          />

          {mode === "login" || mode === "signup" ? (
            <div
              className="surface-muted grid grid-cols-2 gap-1 p-1"
              role="tablist"
            >
              <button
                className={authModeClassName(mode === "login")}
                type="button"
                aria-label={`${authMessages.loginTab} tab`}
                aria-pressed={mode === "login"}
                onClick={() => moveToMode("login")}
              >
                {authMessages.loginTab}
              </button>
              <button
                className={authModeClassName(mode === "signup")}
                type="button"
                aria-label={`${authMessages.signupTab} tab`}
                aria-pressed={mode === "signup"}
                onClick={() => moveToMode("signup")}
              >
                {authMessages.signupTab}
              </button>
            </div>
          ) : (
            <button
              className="secondary-action h-9 w-fit px-3 text-sm"
              type="button"
              aria-label={authMessages.backToLoginButton}
              onClick={returnToLogin}
            >
              <ArrowLeft size={15} />
              {authMessages.backToLoginButton}
            </button>
          )}

          <form className="space-y-3" onSubmit={handleSubmit}>
            {mode !== "reset" ? (
              <label className="block text-sm font-semibold" style={{ color: "var(--text-secondary)" }}>
                <span>{authMessages.emailLabel}</span>
                <input
                  className="field-control mt-1 h-10 w-full rounded-md px-3 text-sm"
                  type="email"
                  aria-label={authMessages.emailLabel}
                  autoComplete="email"
                  placeholder={authMessages.emailPlaceholder}
                  required
                  value={email}
                  disabled={busy}
                  onChange={(event) => setEmail(event.target.value)}
                />
              </label>
            ) : null}
            {mode !== "forgot" ? (
              <label className="block text-sm font-semibold" style={{ color: "var(--text-secondary)" }}>
                <span>{authMessages.passwordLabel}</span>
                <input
                  className="field-control mt-1 h-10 w-full rounded-md px-3 text-sm"
                  type="password"
                  aria-label={authMessages.passwordLabel}
                  autoComplete={
                    mode === "login" ? "current-password" : "new-password"
                  }
                  placeholder={authMessages.passwordPlaceholder}
                  required
                  value={password}
                  disabled={busy}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </label>
            ) : null}
            {mode === "signup" && legacyClaimRequired ? (
              <label className="block text-sm font-semibold" style={{ color: "var(--text-secondary)" }}>
                <span>{authMessages.legacyClaimTokenLabel}</span>
                <input
                  className="field-control mt-1 h-10 w-full rounded-md px-3 text-sm"
                  type="text"
                  aria-label={authMessages.legacyClaimTokenLabel}
                  autoComplete="off"
                  placeholder={authMessages.legacyClaimTokenPlaceholder}
                  value={legacyClaimToken}
                  disabled={busy}
                  onChange={(event) => setLegacyClaimToken(event.target.value)}
                />
              </label>
            ) : null}

            {bootstrapError ? (
              <p className="text-xs font-semibold feedback-warning" role="status">
                {bootstrapError}
              </p>
            ) : null}
            {verificationError ? (
              <p className="text-xs font-semibold feedback-error" role="alert">
                {verificationError}
              </p>
            ) : null}
            {status ? (
              <p className="text-xs font-semibold feedback-success" role="status">
                {status}
              </p>
            ) : null}
            {error ? (
              <p className="text-xs font-semibold feedback-error" role="alert">
                {error}
              </p>
            ) : null}

            <button
              className="primary-action h-10 w-full px-3 text-sm disabled:cursor-not-allowed disabled:opacity-60"
              type="submit"
              aria-label={submitLabel}
              disabled={busy}
            >
              {submitIcon}
              {busy ? authMessages.submitBusy : submitLabel}
            </button>
            {mode === "login" ? (
              <button
                className="inline-flex h-9 items-center gap-2 text-sm font-semibold transition" style={{ color: "var(--accent-denim-strong)" }}
                type="button"
                aria-label={authMessages.forgotPasswordButton}
                onClick={() => moveToMode("forgot")}
              >
                <KeyRound size={14} />
                {authMessages.forgotPasswordButton}
              </button>
            ) : null}
          </form>
        </div>
      </section>
    </div>
  );
}

function authErrorMessage(
  error: unknown,
  mode: AuthMode,
  authMessages: AppMessages["auth"],
) {
  if (isAPIError(error)) {
    if (error.status === 429 || error.code === "too many requests") {
      return authMessages.authRateLimited;
    }

    if (error.code === "invalid verification token") {
      return authMessages.emailVerificationFailed;
    }

    if (error.code === "invalid password reset token") {
      return authMessages.passwordResetTokenFailed;
    }

    if (mode === "login" && error.code === "email not verified") {
      return authMessages.loginEmailNotVerified;
    }

    if (mode === "forgot") {
      switch (error.code) {
        case "invalid email":
          return authMessages.signupInvalidEmail;
        case "invalid locale":
          return authMessages.signupInvalidLocale;
      }
    }

    if (mode === "reset") {
      if (error.code === "weak password") {
        return authMessages.signupWeakPassword;
      }

      return authMessages.resetPasswordFailed;
    }

    if (mode === "signup") {
      switch (error.code) {
        case "weak password":
          return authMessages.signupWeakPassword;
        case "invalid email":
          return authMessages.signupInvalidEmail;
        case "invalid locale":
          return authMessages.signupInvalidLocale;
        case "invalid legacy claim":
          return authMessages.signupInvalidLegacyClaim;
        case "legacy claim required":
          return authMessages.signupLegacyClaimRequired;
      }
    }
  }

  switch (mode) {
    case "login":
      return authMessages.loginFailed;
    case "signup":
      return authMessages.signupFailed;
    case "forgot":
      return authMessages.passwordResetRequestFailed;
    case "reset":
      return authMessages.resetPasswordFailed;
  }
}

function authSubmitLabel(mode: AuthMode, authMessages: AppMessages["auth"]) {
  switch (mode) {
    case "login":
      return authMessages.loginButton;
    case "signup":
      return authMessages.signupButton;
    case "forgot":
      return authMessages.requestPasswordResetButton;
    case "reset":
      return authMessages.resetPasswordButton;
  }
}

function authSubmitIcon(mode: AuthMode) {
  switch (mode) {
    case "login":
      return <LogIn size={17} />;
    case "signup":
      return <UserPlus size={17} />;
    case "forgot":
      return <MailCheck size={17} />;
    case "reset":
      return <KeyRound size={17} />;
  }
}

function Dashboard({
  healthStatus,
  locale,
  messages,
  themePreference,
  user,
  onAuthenticated,
  onLocaleChange,
  onThemePreferenceChange,
  onOpenAuth,
  onLoggedOut,
}: {
  healthStatus: HealthStatus;
  locale: AppLocale;
  messages: AppMessages;
  themePreference: ThemePreference;
  user: UserSession | null;
  onAuthenticated: (response: AuthResponse) => void;
  onLocaleChange: (locale: AppLocale) => void;
  onThemePreferenceChange: (preference: ThemePreference) => void;
  onOpenAuth: () => void;
  onLoggedOut: () => void;
}) {
  const previewMode = user === null;
  const [activeSection, setActiveSection] =
    useState<DashboardSection>("dashboard");
  const monthController = useMonthController({
    messages,
    mode: previewMode ? "preview" : "server",
  });
  const monthLabel = messages.app.monthRecord(
    formatMonth(monthController.month, locale),
  );
  const navigationItems = [
    {
      section: "dashboard" as const,
      label: messages.app.navDashboard,
      icon: <LayoutDashboard size={16} />,
    },
    {
      section: "goals" as const,
      label: messages.app.navGoals,
      icon: <Target size={16} />,
    },
    {
      section: "records" as const,
      label: messages.app.navRecords,
      icon: <ListChecks size={16} />,
    },
    ...(user
      ? [
          {
            section: "account" as const,
            label: messages.app.navAccount,
            icon: <ShieldCheck size={16} />,
          },
        ]
      : []),
  ];
  const healthLabel =
    healthStatus === "healthy"
      ? messages.app.healthHealthy
      : healthStatus === "unhealthy"
        ? messages.app.healthUnhealthy
        : messages.app.healthChecking;
  const healthClassName =
    healthStatus === "healthy"
      ? "status-pill status-pill--api"
      : healthStatus === "unhealthy"
        ? "status-pill status-pill--fallback"
        : "status-pill status-pill--loading";

  function handleSectionNavigation(
    event: ReactMouseEvent<HTMLAnchorElement>,
    section: DashboardSection,
  ) {
    event.preventDefault();
    setActiveSection(section);
    document.getElementById(section)?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
  }

  async function handleLogout() {
    if (!user) {
      return;
    }

    try {
      await logoutSession();
    } finally {
      onLoggedOut();
    }
  }

  return (
    <main className="app-root" id="dashboard">
      <div className="app-shell">
        <aside className="app-sidebar" aria-label={messages.app.title}>
          <div className="sidebar-brand">
            <span className="sidebar-brand-icon" aria-hidden="true">
              <Target size={22} />
            </span>
            <p className="sidebar-title">Monthly Goal Tracker</p>
          </div>

          <div className="sidebar-divider" />

          <div className="sidebar-section sidebar-month-section space-y-3">
            <p className="sidebar-section-title">
              {locale === "ko" ? "월 선택" : "Month"}
            </p>
            <div className="sidebar-month-row">
              <button
                className="icon-button"
                type="button"
                aria-label={messages.app.previousMonth}
                title={messages.app.previousMonth}
                disabled={
                  monthController.isLoading || monthController.isMutatingMonth
                }
                onClick={() => void monthController.moveMonth(-1)}
              >
                <ChevronLeft size={18} />
              </button>
              <label className="field-control sidebar-month-input flex h-10 min-w-0 items-center gap-2 rounded-md px-2 text-sm font-semibold">
                <span className="month-input-label">
                  {formatMonth(monthController.month, locale)}
                </span>
                <CalendarDays
                  size={16}
                  style={{ color: "var(--accent-denim)" }}
                />
                <input
                  className="sr-only"
                  type="month"
                  aria-label={messages.app.monthInput}
                  value={monthController.month}
                  disabled={
                    monthController.isLoading || monthController.isMutatingMonth
                  }
                  onChange={(event) =>
                    void monthController.loadMonth(event.target.value)
                  }
                />
              </label>
              <button
                className="icon-button"
                type="button"
                aria-label={messages.app.nextMonth}
                title={messages.app.nextMonth}
                disabled={
                  monthController.isLoading || monthController.isMutatingMonth
                }
                onClick={() => void monthController.moveMonth(1)}
              >
                <ChevronRight size={18} />
              </button>
            </div>
            <button
              className="secondary-action h-10 w-full px-3 text-sm"
              type="button"
              aria-label={messages.app.prepareMonth}
              title={messages.app.prepareMonth}
              disabled={
                !monthController.canSaveChanges ||
                monthController.isMutatingMonth
              }
              onClick={() => void monthController.prepareCurrentMonth()}
            >
              <CalendarPlus size={17} />
              {messages.app.prepareMonth}
            </button>
          </div>

          <nav className="space-y-1" aria-label={messages.app.navDashboard}>
            {navigationItems.map((item) => (
              <a
                key={item.section}
                className={`sidebar-nav-link ${
                  activeSection === item.section ? "is-active" : ""
                }`}
                href={`#${item.section}`}
                aria-current={
                  activeSection === item.section ? "location" : undefined
                }
                onClick={(event) =>
                  handleSectionNavigation(event, item.section)
                }
              >
                {item.icon}
                {item.label}
              </a>
            ))}
          </nav>

          <div className="sidebar-divider" />

          <ThemePreferenceToggle
            displayLabel={locale === "ko" ? "테마" : messages.app.themeLabel}
            label={messages.app.themeLabel}
            preference={themePreference}
            systemLabel={messages.app.themeSystem}
            lightLabel={messages.app.themeLight}
            darkLabel={messages.app.themeDark}
            onChange={onThemePreferenceChange}
          />

          {previewMode ? (
            <div className="preview-card">
              <div className="flex items-center justify-between gap-3">
                <p className="text-sm font-semibold" style={{ color: "var(--text-primary)" }}>
                  {messages.app.previewMode}
                </p>
                <span className="preview-info" aria-hidden="true">
                  i
                </span>
              </div>
              <p className="mt-3 text-xs font-semibold leading-relaxed" style={{ color: "var(--text-muted)" }}>
                {messages.app.previewNotice}
              </p>
              <button
                className="secondary-action mt-4 h-10 w-full px-3 text-sm"
                type="button"
                aria-label={messages.app.previewSaveLogin}
                onClick={onOpenAuth}
              >
                {messages.app.previewSaveLogin}
              </button>
            </div>
          ) : null}
        </aside>

        <div className="app-content">
          <div className="app-content-inner">
            <header className="app-topbar">
              <h1 className="sr-only">{monthLabel}</h1>
              <div className="topbar-status">
                <span
                  aria-live="polite"
                  className={healthClassName}
                  role="status"
                >
                  {healthLabel}
                </span>
                {previewMode ? (
                  <span className="status-pill status-pill--preview">
                    {messages.app.previewMode}
                  </span>
                ) : null}
                <span className="sr-only">
                  {statusLabel(monthController.loadStatus, messages.status)}
                </span>
                {monthController.loadError ? (
                  <div className="mt-2 flex flex-wrap items-center gap-2">
                    <p className="text-xs font-semibold feedback-warning" role="status">
                      {messages.app.fallbackNotice}
                    </p>
                    <button
                      className="secondary-action h-8 px-2.5 text-xs"
                      type="button"
                      aria-label={messages.app.retry}
                      disabled={monthController.isLoading}
                      onClick={() =>
                        void monthController.loadMonth(monthController.month)
                      }
                    >
                      <RefreshCw size={14} />
                      {messages.app.retry}
                    </button>
                  </div>
                ) : null}
                {monthController.saveError &&
                monthController.saveFeedbackScope !== "goal" ? (
                  <p className="mt-2 text-xs font-semibold feedback-error" role="alert">
                    {monthController.saveError}
                  </p>
                ) : null}
                {monthController.saveMessage &&
                monthController.saveFeedbackScope !== "goal" ? (
                  <p className="mt-2 text-xs font-semibold feedback-success" role="status">
                    {monthController.saveMessage}
                  </p>
                ) : null}
              </div>

              <div className="app-toolbar">
                <LanguageToggle
                  compact
                  label={messages.app.languageLabel}
                  locale={locale}
                  onChange={onLocaleChange}
                />
                {user ? (
                  <>
                    <p
                      className="max-w-[13rem] truncate text-xs font-semibold"
                      style={{ color: "var(--text-secondary)" }}
                      title={messages.app.signedInAs(user.email)}
                    >
                      {messages.app.signedInAs(user.email)}
                    </p>
                    <button
                      className="icon-button"
                      type="button"
                      aria-label={messages.app.logout}
                      title={messages.app.logout}
                      onClick={() => void handleLogout()}
                    >
                      <LogOut size={18} />
                    </button>
                  </>
                ) : (
                  <button
                    className="secondary-action h-10 px-3 text-sm"
                    type="button"
                    aria-label={messages.app.login}
                    onClick={onOpenAuth}
                  >
                    <LogIn size={17} />
                    {messages.app.login}
                  </button>
                )}
              </div>
            </header>

            <MetricSummary
              activeGoalCount={monthController.activeMetricGoalCount}
              activeMetricLabel={monthController.activeMetricLabel}
              averageRate={monthController.averageRate}
              labels={messages.summary}
              totalCompleted={monthController.totalCompleted}
            />

            <section className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_26rem]">
              <div id="records" className="min-w-0">
                <DailyRecordTable
                  canSaveChanges={monthController.canSaveChanges}
                  chartData={monthController.chartData}
                  checkableThroughDate={monthController.checkableThroughDate}
                  checks={monthController.checks}
                  dailyRecordGoalSlots={monthController.dailyRecordGoalSlots}
                  days={monthController.days}
                  isMutatingMonth={monthController.isMutatingMonth}
                  labels={messages.dailyRecord}
                  locale={locale}
                  savingChecks={monthController.savingChecks}
                  savingMemos={monthController.savingMemos}
                  onMemoBlur={(date, memo) =>
                    void monthController.saveMemoForDate(date, memo)
                  }
                  onMemoChange={monthController.updateMemo}
                  onToggleCheck={(goalId, date) =>
                    void monthController.toggleCheck(goalId, date)
                  }
                />
              </div>

              <aside className="dashboard-side-panel space-y-3">
                <div id="goals">
                  <GoalPanel
                    canSaveChanges={monthController.canSaveChanges}
                    deactivatingGoalIDs={monthController.deactivatingGoalIDs}
                    editingGoalID={monthController.editingGoalID}
                    editingGoalTitle={monthController.editingGoalTitle}
                    goalFormOpen={monthController.goalFormOpen}
                    goalListReferenceDate={monthController.goalListReferenceDate}
                    isMutatingMonth={monthController.isMutatingMonth}
                    labels={messages.goalPanel}
                    month={monthController.month}
                    newGoalStartDate={monthController.newGoalStartDate}
                    newGoalTitle={monthController.newGoalTitle}
                    saveError={
                      monthController.saveFeedbackScope === "goal"
                        ? monthController.saveError
                        : null
                    }
                    saveMessage={
                      monthController.saveFeedbackScope === "goal"
                        ? monthController.saveMessage
                        : null
                    }
                    savingGoal={monthController.savingGoal}
                    savingGoalTitle={monthController.savingGoalTitle}
                    visibleGoals={monthController.visibleGoals}
                    goalProgress={buildGoalProgress(
                      monthController.visibleGoals,
                      monthController.days,
                      monthController.checks,
                    )}
                    onCancelEditingGoal={monthController.cancelEditingGoal}
                    onDeactivateGoal={(goal) =>
                      void monthController.deactivateGoalFromMonth(goal)
                    }
                    onEditingGoalTitleChange={monthController.setEditingGoalTitle}
                    onNewGoalStartDateChange={monthController.setNewGoalStartDate}
                    onNewGoalTitleChange={monthController.setNewGoalTitle}
                    onStartEditingGoal={monthController.startEditingGoal}
                    onSubmitGoalTitle={(event, goalID) =>
                      void monthController.submitGoalTitle(event, goalID)
                    }
                    onSubmitNewGoal={(event) =>
                      void monthController.submitNewGoal(event)
                    }
                    onToggleGoalForm={monthController.toggleGoalForm}
                  />
                </div>
                <ChartPanel
                  chartData={monthController.chartData}
                  goalCount={monthController.goals.length}
                  labels={messages.chart}
                  month={monthController.month}
                />
                {user ? (
                  <div id="account">
                    <AccountSecurityPanel
                      labels={messages.account}
                      onPasswordChanged={onAuthenticated}
                    />
                  </div>
                ) : null}
              </aside>
            </section>
          </div>
        </div>
      </div>
    </main>
  );
}

function ThemePreferenceToggle({
  darkLabel,
  displayLabel,
  label,
  lightLabel,
  preference,
  systemLabel,
  onChange,
}: {
  darkLabel: string;
  displayLabel?: string;
  label: string;
  lightLabel: string;
  preference: ThemePreference;
  systemLabel: string;
  onChange: (preference: ThemePreference) => void;
}) {
  const options: Array<{
    icon: JSX.Element;
    label: string;
    value: ThemePreference;
  }> = [
    { icon: <Monitor size={14} />, label: systemLabel, value: "system" },
    { icon: <Sun size={14} />, label: lightLabel, value: "light" },
    { icon: <Moon size={14} />, label: darkLabel, value: "dark" },
  ];

  return (
    <div className="theme-toggle-panel sidebar-section space-y-2">
      <p className="sidebar-section-title">{displayLabel ?? label}</p>
      <div className="theme-toggle" role="radiogroup" aria-label={label}>
        {options.map((option) => (
          <button
            key={option.value}
            className={`theme-toggle-button ${
              preference === option.value ? "is-active" : ""
            }`}
            type="button"
            role="radio"
            aria-checked={preference === option.value}
            aria-label={`${label} ${option.label}`}
            onClick={() => onChange(option.value)}
          >
            {option.icon}
            <span>{option.label}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

function buildGoalProgress(
  goals: Array<{ id: number; startDate: string; endDate: string | null }>,
  days: Array<{ date: string }>,
  checks: Array<{ goalId: number; date: string }>,
) {
  return Object.fromEntries(
    goals.map((goal) => {
      const activeDays = days.filter(
        (day) =>
          goal.startDate <= day.date && (goal.endDate === null || day.date <= goal.endDate),
      );
      const completed = checks.filter((check) => check.goalId === goal.id).length;
      const total = activeDays.length;
      const percent = total > 0 ? Math.round((completed / total) * 100) : 0;

      return [goal.id, { completed, percent, total }];
    }),
  );
}

function AccountSecurityPanel({
  labels,
  onPasswordChanged,
}: {
  labels: AppMessages["account"];
  onPasswordChanged: (response: AuthResponse) => void;
}) {
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [sessionBusy, setSessionBusy] = useState(false);
  const [sessionError, setSessionError] = useState<string | null>(null);
  const [sessionStatus, setSessionStatus] = useState<string | null>(null);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    setStatus(null);

    try {
      const response = await changePassword(currentPassword, newPassword);
      onPasswordChanged(response);
      setCurrentPassword("");
      setNewPassword("");
      setStatus(labels.passwordChanged);
    } catch (error) {
      setError(passwordChangeErrorMessage(error, labels));
    } finally {
      setBusy(false);
    }
  }

  async function handleLogoutOtherSessions() {
    setSessionBusy(true);
    setSessionError(null);
    setSessionStatus(null);

    try {
      await logoutOtherSessions();
      setSessionStatus(labels.otherSessionsLoggedOut);
    } catch {
      setSessionError(labels.otherSessionsLogoutFailed);
    } finally {
      setSessionBusy(false);
    }
  }

  return (
    <section className="panel-card panel-card-padded">
      <form className="space-y-3" onSubmit={handleSubmit}>
        <h2 className="flex items-center gap-2 text-base font-semibold tracking-normal" style={{ color: "var(--text-primary)" }}>
          <KeyRound size={17} style={{ color: "var(--accent-denim)" }} />
          {labels.heading}
        </h2>
        <label className="block text-sm font-semibold" style={{ color: "var(--text-secondary)" }}>
          <span>{labels.currentPasswordLabel}</span>
          <input
            className="field-control mt-1 h-10 w-full rounded-md px-3 text-sm"
            type="password"
            aria-label={labels.currentPasswordLabel}
            autoComplete="current-password"
            placeholder={labels.currentPasswordPlaceholder}
            required
            value={currentPassword}
            disabled={busy}
            onChange={(event) => setCurrentPassword(event.target.value)}
          />
        </label>
        <label className="block text-sm font-semibold" style={{ color: "var(--text-secondary)" }}>
          <span>{labels.newPasswordLabel}</span>
          <input
            className="field-control mt-1 h-10 w-full rounded-md px-3 text-sm"
            type="password"
            aria-label={labels.newPasswordLabel}
            autoComplete="new-password"
            placeholder={labels.newPasswordPlaceholder}
            required
            value={newPassword}
            disabled={busy}
            onChange={(event) => setNewPassword(event.target.value)}
          />
        </label>
        {status ? (
          <p className="text-xs font-semibold feedback-success" role="status">
            {status}
          </p>
        ) : null}
        {error ? (
          <p className="text-xs font-semibold feedback-error" role="alert">
            {error}
          </p>
        ) : null}
        <button
          className="primary-action h-10 w-full px-3 text-sm disabled:cursor-not-allowed disabled:opacity-60"
          type="submit"
          aria-label={busy ? labels.changingPassword : labels.changePasswordButton}
          disabled={busy}
        >
          <KeyRound size={17} />
          {busy ? labels.changingPassword : labels.changePasswordButton}
        </button>
      </form>
      <div className="mt-4 border-t pt-3" style={{ borderColor: "var(--border-subtle)" }}>
        {sessionStatus ? (
          <p className="text-xs font-semibold feedback-success" role="status">
            {sessionStatus}
          </p>
        ) : null}
        {sessionError ? (
          <p className="text-xs font-semibold feedback-error" role="alert">
            {sessionError}
          </p>
        ) : null}
        <button
          className="secondary-action mt-3 h-10 w-full px-3 text-sm disabled:cursor-not-allowed disabled:opacity-60"
          type="button"
          aria-label={
            sessionBusy
              ? labels.loggingOutOtherSessions
              : labels.logoutOtherSessionsButton
          }
          disabled={sessionBusy}
          onClick={() => void handleLogoutOtherSessions()}
        >
          <ShieldCheck size={17} style={{ color: "var(--accent-denim)" }} />
          {sessionBusy
            ? labels.loggingOutOtherSessions
            : labels.logoutOtherSessionsButton}
        </button>
      </div>
    </section>
  );
}

function passwordChangeErrorMessage(
  error: unknown,
  labels: AppMessages["account"],
) {
  if (isAPIError(error)) {
    if (error.code === "weak password") {
      return labels.passwordChangeWeakPassword;
    }

    if (error.status === 401 || error.code === "unauthorized") {
      return labels.passwordChangeUnauthorized;
    }
  }

  return labels.passwordChangeFailed;
}

function LanguageToggle({
  compact = false,
  label,
  locale,
  onChange,
}: {
  compact?: boolean;
  label: string;
  locale: AppLocale;
  onChange: (locale: AppLocale) => void;
}) {
  const koreanLabel = compact ? "KO" : locale === "en" ? "Korean" : "한국어";
  const englishLabel = compact ? "EN" : "English";

  return (
    <div
      className={`inline-flex shrink-0 items-center ${
        compact
          ? "surface-muted h-10 w-[7.25rem] px-1.5"
          : ""
      }`}
      aria-label={label}
    >
      <div className="segmented-control inline-flex w-full p-0.5">
        <button
          className={languageButtonClassName(locale === "ko", compact)}
          type="button"
          aria-label={`${label} ${koreanLabel}`}
          aria-pressed={locale === "ko"}
          onClick={() => onChange("ko")}
        >
          {koreanLabel}
        </button>
        <button
          className={languageButtonClassName(locale === "en", compact)}
          type="button"
          aria-label={`${label} ${englishLabel}`}
          aria-pressed={locale === "en"}
          onClick={() => onChange("en")}
        >
          {englishLabel}
        </button>
      </div>
    </div>
  );
}

function authModeClassName(active: boolean) {
  const base = "segmented-button h-9 rounded-md px-3 text-sm";
  return active
    ? `${base} segmented-button-active`
    : `${base} segmented-button-inactive`;
}

function languageButtonClassName(active: boolean, compact: boolean) {
  const base = "segmented-button h-8 whitespace-nowrap rounded px-2 text-xs";
  const width = compact ? "w-12" : "min-w-[4rem]";
  return active
    ? `${base} ${width} segmented-button-active`
    : `${base} ${width} segmented-button-inactive`;
}

function readStoredThemePreference(): ThemePreference {
  try {
    const storedTheme = window.localStorage.getItem(themeStorageKey);
    return storedTheme === "light" ||
      storedTheme === "dark" ||
      storedTheme === "system"
      ? storedTheme
      : "dark";
  } catch {
    return "dark";
  }
}

function storeThemePreference(preference: ThemePreference) {
  try {
    window.localStorage.setItem(themeStorageKey, preference);
  } catch {
    // Theme preference is optional; system mode remains the fallback.
  }
}

function resolveThemePreference(preference: ThemePreference): ResolvedTheme {
  if (preference === "light" || preference === "dark") {
    return preference;
  }

  return preferredSystemTheme();
}

function preferredSystemTheme(): ResolvedTheme {
  if (typeof window.matchMedia !== "function") {
    return "light";
  }

  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function readStoredLocale(): AppLocale | null {
  try {
    const storedLocale = window.localStorage.getItem(localeStorageKey);
    return storedLocale === "ko" || storedLocale === "en"
      ? storedLocale
      : null;
  } catch {
    return null;
  }
}

function storeLocale(locale: AppLocale) {
  try {
    window.localStorage.setItem(localeStorageKey, locale);
  } catch {
    // Local storage is optional; the server session remains authoritative.
  }
}

function readEmailVerificationToken() {
  const params = new URLSearchParams(window.location.search);
  return (
    params.get("token")?.trim() ||
    params.get("verifyToken")?.trim() ||
    null
  );
}

function removeEmailVerificationToken() {
  removeSearchParams(["token", "verifyToken"]);
}

function readPasswordResetToken() {
  const params = hashParams();
  return (
    params.get("resetToken")?.trim() ||
    params.get("passwordResetToken")?.trim() ||
    null
  );
}

function removePasswordResetToken() {
  removeSearchParams(["resetToken", "passwordResetToken"]);
  removeHashParams(["resetToken", "passwordResetToken"]);
}

function hashParams() {
  return new URLSearchParams(window.location.hash.replace(/^#/, ""));
}

function removeHashParams(keys: string[]) {
  const url = new URL(window.location.href);
  const params = new URLSearchParams(url.hash.replace(/^#/, ""));
  let changed = false;

  for (const key of keys) {
    if (params.has(key)) {
      params.delete(key);
      changed = true;
    }
  }

  if (changed) {
    url.hash = params.toString();
    window.history.replaceState({}, "", `${url.pathname}${url.search}${url.hash}`);
  }
}

function removeSearchParams(keys: string[]) {
  const url = new URL(window.location.href);
  let changed = false;

  for (const key of keys) {
    if (url.searchParams.has(key)) {
      url.searchParams.delete(key);
      changed = true;
    }
  }

  if (changed) {
    window.history.replaceState({}, "", `${url.pathname}${url.search}${url.hash}`);
  }
}
