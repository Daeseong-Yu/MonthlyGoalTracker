import { type FormEvent, useEffect, useMemo, useState } from "react";
import {
  CalendarDays,
  CalendarPlus,
  ChevronLeft,
  ChevronRight,
  Languages,
  LogIn,
  LogOut,
  RefreshCw,
  UserPlus,
} from "lucide-react";

import {
  bootstrapSession,
  clearAuthCSRFToken,
  login as loginSession,
  logoutSession,
  signUp as signUpSession,
  updateUserLocale,
} from "./api";
import { formatMonth, statusClassName, statusLabel } from "./appDisplay";
import ChartPanel from "./ChartPanel";
import DailyRecordTable from "./DailyRecordTable";
import GoalPanel from "./GoalPanel";
import { messagesForLocale, normalizeLocale, type AppMessages } from "./i18n";
import MetricSummary from "./MetricSummary";
import { useMonthController } from "./useMonthController";
import type { AppLocale, AuthResponse, UserSession } from "./types";

type AuthMode = "login" | "signup";
const localeStorageKey = "monthlyGoalTracker.locale";

export default function App() {
  const [locale, setLocale] = useState<AppLocale>(
    () => readStoredLocale() ?? "ko",
  );
  const [user, setUser] = useState<UserSession | null>(null);
  const [bootstrapped, setBootstrapped] = useState(false);
  const [bootstrapFailed, setBootstrapFailed] = useState(false);
  const messages = useMemo(() => messagesForLocale(locale), [locale]);

  useEffect(() => {
    let cancelled = false;

    bootstrapSession()
      .then((response) => {
        if (cancelled) {
          return;
        }

        const nextUser = response.authenticated ? response.user : null;
        const nextLocale = normalizeLocale(
          nextUser?.locale ?? readStoredLocale() ?? response.locale,
        );
        setLocale(nextLocale);
        storeLocale(nextLocale);
        setUser(nextUser);
        setBootstrapFailed(false);
      })
      .catch(() => {
        if (cancelled) {
          return;
        }

        clearAuthCSRFToken();
        setUser(null);
        setBootstrapFailed(true);
      })
      .finally(() => {
        if (!cancelled) {
          setBootstrapped(true);
        }
      });

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

  function handleAuthenticated(response: AuthResponse) {
    const nextLocale = normalizeLocale(response.user.locale ?? response.locale);
    setLocale(nextLocale);
    storeLocale(nextLocale);
    setUser(response.user);
    setBootstrapFailed(false);
  }

  function handleLoggedOut() {
    clearAuthCSRFToken();
    setUser(null);
  }

  if (!bootstrapped) {
    return <BootstrapScreen messages={messages} />;
  }

  if (!user) {
    return (
      <AuthScreen
        bootstrapError={
          bootstrapFailed ? messages.app.bootstrapError : null
        }
        locale={locale}
        messages={messages}
        onAuthenticated={handleAuthenticated}
        onLocaleChange={(nextLocale) => void handleLocaleChange(nextLocale)}
      />
    );
  }

  return (
    <Dashboard
      locale={locale}
      messages={messages}
      user={user}
      onLocaleChange={(nextLocale) => void handleLocaleChange(nextLocale)}
      onLoggedOut={handleLoggedOut}
    />
  );
}

function BootstrapScreen({ messages }: { messages: AppMessages }) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-[#f7f8f5] px-4 text-zinc-900">
      <p
        className="rounded-md border border-zinc-200 bg-white px-4 py-3 text-sm font-medium text-zinc-600 shadow-soft"
        role="status"
      >
        {messages.app.bootstrapLoading}
      </p>
    </main>
  );
}

function AuthScreen({
  bootstrapError,
  locale,
  messages,
  onAuthenticated,
  onLocaleChange,
}: {
  bootstrapError: string | null;
  locale: AppLocale;
  messages: AppMessages;
  onAuthenticated: (response: AuthResponse) => void;
  onLocaleChange: (locale: AppLocale) => void;
}) {
  const [mode, setMode] = useState<AuthMode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [legacyClaimToken, setLegacyClaimToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const authMessages = messages.auth;
  const submitLabel =
    mode === "login" ? authMessages.loginButton : authMessages.signupButton;
  const submitIcon =
    mode === "login" ? <LogIn size={17} /> : <UserPlus size={17} />;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError(null);

    try {
      const response =
        mode === "login"
          ? await loginSession(email, password, locale)
          : await signUpSession(email, password, locale, legacyClaimToken);
      onAuthenticated(response);
    } catch {
      setError(
        mode === "login"
          ? authMessages.loginFailed
          : authMessages.signupFailed,
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-[#f7f8f5] px-4 py-8 text-zinc-900">
      <section className="w-full max-w-md rounded-lg border border-zinc-200 bg-white p-5 shadow-soft">
        <div className="flex flex-col gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-normal text-zinc-950">
              {authMessages.title}
            </h1>
            <p className="mt-1 text-sm text-zinc-600">
              {authMessages.subtitle}
            </p>
          </div>

          <LanguageToggle
            label={messages.app.languageLabel}
            locale={locale}
            onChange={onLocaleChange}
          />

          <div
            className="grid grid-cols-2 gap-1 rounded-md border border-zinc-200 bg-zinc-50 p-1"
            role="tablist"
          >
            <button
              className={authModeClassName(mode === "login")}
              type="button"
              aria-label={`${authMessages.loginTab} tab`}
              aria-pressed={mode === "login"}
              onClick={() => {
                setMode("login");
                setError(null);
                setLegacyClaimToken("");
              }}
            >
              {authMessages.loginTab}
            </button>
            <button
              className={authModeClassName(mode === "signup")}
              type="button"
              aria-label={`${authMessages.signupTab} tab`}
              aria-pressed={mode === "signup"}
              onClick={() => {
                setMode("signup");
                setError(null);
              }}
            >
              {authMessages.signupTab}
            </button>
          </div>

          <form className="space-y-3" onSubmit={handleSubmit}>
            <label className="block text-sm font-medium text-zinc-700">
              <span>{authMessages.emailLabel}</span>
              <input
                className="mt-1 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100"
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
            <label className="block text-sm font-medium text-zinc-700">
              <span>{authMessages.passwordLabel}</span>
              <input
                className="mt-1 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100"
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
            {mode === "signup" ? (
              <label className="block text-sm font-medium text-zinc-700">
                <span>{authMessages.legacyClaimTokenLabel}</span>
                <input
                  className="mt-1 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100"
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
              <p className="text-xs font-medium text-amber-700" role="status">
                {bootstrapError}
              </p>
            ) : null}
            {error ? (
              <p className="text-xs font-medium text-rose-700" role="alert">
                {error}
              </p>
            ) : null}

            <button
              className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-teal-700 px-3 text-sm font-semibold text-white transition hover:bg-teal-800 disabled:cursor-not-allowed disabled:opacity-60"
              type="submit"
              aria-label={submitLabel}
              disabled={busy}
            >
              {submitIcon}
              {busy ? authMessages.submitBusy : submitLabel}
            </button>
          </form>

          <p className="text-xs text-zinc-500">{authMessages.languageHint}</p>
        </div>
      </section>
    </main>
  );
}

function Dashboard({
  locale,
  messages,
  user,
  onLocaleChange,
  onLoggedOut,
}: {
  locale: AppLocale;
  messages: AppMessages;
  user: UserSession;
  onLocaleChange: (locale: AppLocale) => void;
  onLoggedOut: () => void;
}) {
  const monthController = useMonthController({ messages });

  async function handleLogout() {
    try {
      await logoutSession();
    } finally {
      onLoggedOut();
    }
  }

  return (
    <main className="min-h-screen bg-[#f7f8f5] text-zinc-900">
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 py-5 sm:px-6 lg:px-8">
        <header className="flex flex-col gap-4 border-b border-zinc-200 pb-5 xl:flex-row xl:items-center xl:justify-between">
          <div className="min-w-0">
            <h1 className="text-2xl font-semibold tracking-normal text-zinc-950">
              {messages.app.title}
            </h1>
            <p className="mt-1 flex flex-wrap items-center gap-2 text-sm text-zinc-600">
              <span>
                {messages.app.monthRecord(
                  formatMonth(monthController.month, locale),
                )}
              </span>
              <span
                aria-live="polite"
                className={statusClassName(monthController.loadStatus)}
                role="status"
              >
                {statusLabel(monthController.loadStatus, messages.status)}
              </span>
            </p>
            {monthController.loadError ? (
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <p className="text-xs font-medium text-amber-700" role="status">
                  {messages.app.fallbackNotice}
                </p>
                <button
                  className="inline-flex h-8 items-center gap-1.5 rounded-md border border-amber-300 bg-white px-2.5 text-xs font-semibold text-amber-800 shadow-sm transition hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-60"
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
            {monthController.saveError ? (
              <p className="mt-2 text-xs font-medium text-rose-700" role="alert">
                {monthController.saveError}
              </p>
            ) : null}
            {monthController.saveMessage ? (
              <p
                className="mt-2 text-xs font-medium text-teal-700"
                role="status"
              >
                {monthController.saveMessage}
              </p>
            ) : null}
          </div>

          <div className="flex flex-wrap items-center gap-2">
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
            <label className="flex h-10 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm font-medium shadow-sm">
              <CalendarDays size={17} className="text-teal-700" />
              <input
                className="w-[8.5rem] bg-transparent text-sm outline-none"
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
            <button
              className="icon-button"
              type="button"
              aria-label={messages.app.prepareMonth}
              title={messages.app.prepareMonth}
              disabled={
                !monthController.canSaveChanges ||
                monthController.isMutatingMonth
              }
              onClick={() => void monthController.prepareCurrentMonth()}
            >
              <CalendarPlus size={18} />
            </button>
            <LanguageToggle
              compact
              label={messages.app.languageLabel}
              locale={locale}
              onChange={onLocaleChange}
            />
            <p
              className="max-w-[13rem] truncate text-xs font-medium text-zinc-600"
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
          </div>
        </header>

        <MetricSummary
          activeGoalCount={monthController.activeMetricGoalCount}
          activeMetricLabel={monthController.activeMetricLabel}
          averageRate={monthController.averageRate}
          labels={messages.summary}
          totalCompleted={monthController.totalCompleted}
        />

        <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_20.5rem]">
          <DailyRecordTable
            canSaveChanges={monthController.canSaveChanges}
            chartData={monthController.chartData}
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

          <aside className="space-y-6">
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
              savingGoal={monthController.savingGoal}
              savingGoalTitle={monthController.savingGoalTitle}
              visibleGoals={monthController.visibleGoals}
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
            <ChartPanel
              chartData={monthController.chartData}
              goalCount={monthController.goals.length}
              labels={messages.chart}
              month={monthController.month}
            />
          </aside>
        </section>
      </div>
    </main>
  );
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
  return (
    <div
      className={`flex items-center gap-2 ${
        compact ? "h-10 rounded-md border border-zinc-300 bg-white px-2" : ""
      }`}
      aria-label={label}
    >
      <Languages size={17} className="text-teal-700" />
      <div className="inline-flex rounded-md border border-zinc-200 bg-zinc-50 p-0.5">
        <button
          className={languageButtonClassName(locale === "ko")}
          type="button"
          aria-label={`${label} 한국어`}
          aria-pressed={locale === "ko"}
          onClick={() => onChange("ko")}
        >
          한국어
        </button>
        <button
          className={languageButtonClassName(locale === "en")}
          type="button"
          aria-label={`${label} English`}
          aria-pressed={locale === "en"}
          onClick={() => onChange("en")}
        >
          English
        </button>
      </div>
    </div>
  );
}

function authModeClassName(active: boolean) {
  const base =
    "h-9 rounded-md px-3 text-sm font-semibold transition focus:outline-none focus:ring-2 focus:ring-teal-100";
  return active
    ? `${base} bg-white text-teal-800 shadow-sm`
    : `${base} text-zinc-600 hover:text-zinc-950`;
}

function languageButtonClassName(active: boolean) {
  const base =
    "h-8 min-w-[4rem] rounded px-2 text-xs font-semibold transition focus:outline-none focus:ring-2 focus:ring-teal-100";
  return active
    ? `${base} bg-white text-teal-800 shadow-sm`
    : `${base} text-zinc-600 hover:text-zinc-950`;
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
