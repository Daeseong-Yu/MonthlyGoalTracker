import { useEffect, useRef, useState } from "react";

import {
  bootstrapSession,
  clearAuthCSRFToken,
  logoutSession,
  updateUserLocale,
  verifyEmail,
} from "./api";
import { normalizeLocale } from "./i18n";
import type { AppLocale, AuthResponse, UserSession } from "./types";

const localeStorageKey = "monthlyGoalTracker.locale";

export type AccountLifecycleState = {
  authFlow: {
    bootstrapFailed: boolean;
    open: boolean;
    passwordResetToken: string | null;
    verificationFailed: boolean;
  };
  bootstrapped: boolean;
  locale: AppLocale;
  user: UserSession | null;
};

export function useAccountLifecycle() {
  const [locale, setLocale] = useState<AppLocale>(
    () => readStoredLocale() ?? "ko",
  );
  const [user, setUser] = useState<UserSession | null>(null);
  const [bootstrapped, setBootstrapped] = useState(false);
  const [bootstrapFailed, setBootstrapFailed] = useState(false);
  const [verificationFailed, setVerificationFailed] = useState(false);
  const [passwordResetToken, setPasswordResetToken] = useState<string | null>(
    () => readPasswordResetToken(),
  );
  const [showAuthScreen, setShowAuthScreen] = useState(false);
  const accountGenerationRef = useRef(0);
  const localeSaveSequenceRef = useRef(0);

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
        const verificationToken = nextUser
          ? null
          : readEmailVerificationToken();
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
            applyAuthentication(authResponse);
            return;
          } catch {
            if (cancelled) {
              return;
            }

            clearAuthCSRFToken();
            removeEmailVerificationToken();
            setUser(null);
            setPasswordResetToken(resetToken);
            setVerificationFailed(true);
            setBootstrapFailed(false);
            return;
          }
        }

        setUser(nextUser);
        setPasswordResetToken(resetToken);
        setVerificationFailed(false);
        setBootstrapFailed(false);
      } catch {
        if (cancelled) {
          return;
        }

        clearAuthCSRFToken();
        setUser(null);
        setPasswordResetToken(readPasswordResetToken());
        setVerificationFailed(false);
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

  async function changeLocale(nextLocale: AppLocale) {
    if (nextLocale === locale) {
      return;
    }

    setLocale(nextLocale);
    storeLocale(nextLocale);
    if (!user) {
      return;
    }

    const accountGeneration = accountGenerationRef.current;
    const localeSaveSequence = localeSaveSequenceRef.current + 1;
    localeSaveSequenceRef.current = localeSaveSequence;
    setUser({ ...user, locale: nextLocale });

    try {
      const updatedUser = await updateUserLocale(nextLocale);
      if (isCurrentLocaleSave(accountGeneration, localeSaveSequence)) {
        setUser(updatedUser);
      }
    } catch {
      if (isCurrentLocaleSave(accountGeneration, localeSaveSequence)) {
        setUser((currentUser) =>
          currentUser === null
            ? currentUser
            : { ...currentUser, locale: nextLocale },
        );
      }
    }
  }

  function applyAuthentication(response: AuthResponse) {
    accountGenerationRef.current += 1;
    const nextLocale = normalizeLocale(response.user.locale ?? response.locale);
    setLocale(nextLocale);
    storeLocale(nextLocale);
    setUser(response.user);
    setPasswordResetToken(null);
    setShowAuthScreen(false);
    removeEmailVerificationToken();
    removePasswordResetToken();
    setVerificationFailed(false);
    setBootstrapFailed(false);
  }

  async function logout() {
    try {
      await logoutSession();
    } finally {
      accountGenerationRef.current += 1;
      clearAuthCSRFToken();
      setUser(null);
      setPasswordResetToken(readPasswordResetToken());
      setShowAuthScreen(false);
      setVerificationFailed(false);
    }
  }

  function closeAuthFlow() {
    setShowAuthScreen(false);
    setVerificationFailed(false);
    if (passwordResetToken !== null) {
      consumePasswordResetToken();
    }
  }

  function consumePasswordResetToken() {
    setPasswordResetToken(null);
    removePasswordResetToken();
  }

  function isCurrentLocaleSave(
    accountGeneration: number,
    localeSaveSequence: number,
  ) {
    return (
      accountGenerationRef.current === accountGeneration &&
      localeSaveSequenceRef.current === localeSaveSequence
    );
  }

  const state: AccountLifecycleState = {
    authFlow: {
      bootstrapFailed,
      open:
        !user &&
        (showAuthScreen ||
          passwordResetToken !== null ||
          verificationFailed),
      passwordResetToken,
      verificationFailed,
    },
    bootstrapped,
    locale,
    user,
  };

  return {
    applyAuthentication,
    changeLocale,
    closeAuthFlow,
    consumePasswordResetToken,
    logout,
    openAuthFlow: () => setShowAuthScreen(true),
    state,
  };
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
    window.history.replaceState(
      {},
      "",
      `${url.pathname}${url.search}${url.hash}`,
    );
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
    window.history.replaceState(
      {},
      "",
      `${url.pathname}${url.search}${url.hash}`,
    );
  }
}
