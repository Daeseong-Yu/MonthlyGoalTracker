import type { LoadStatus } from "./appDisplay";
import type { AppLocale } from "./types";

export type AppMessages = {
  app: {
    title: string;
    monthRecord: (month: string) => string;
    fallbackNotice: string;
    retry: string;
    previousMonth: string;
    nextMonth: string;
    monthInput: string;
    prepareMonth: string;
    signedInAs: (email: string) => string;
    logout: string;
    languageLabel: string;
    bootstrapLoading: string;
    bootstrapError: string;
  };
  auth: {
    title: string;
    subtitle: string;
    loginTab: string;
    signupTab: string;
    emailLabel: string;
    emailPlaceholder: string;
    passwordLabel: string;
    passwordPlaceholder: string;
    legacyClaimTokenLabel: string;
    legacyClaimTokenPlaceholder: string;
    loginButton: string;
    signupButton: string;
    submitBusy: string;
    loginFailed: string;
    signupFailed: string;
    authRateLimited: string;
    signupWeakPassword: string;
    signupInvalidEmail: string;
    signupInvalidLocale: string;
    signupInvalidLegacyClaim: string;
    signupLegacyClaimRequired: string;
    languageHint: string;
  };
  status: Record<LoadStatus, string>;
  summary: {
    totalCompleted: string;
    averageRate: string;
    completedValue: (value: number) => string;
    goalValue: (value: number) => string;
    todayActiveGoals: string;
    dayActiveGoals: (day: number) => string;
  };
  goalPanel: {
    heading: string;
    add: string;
    newGoalTitleAria: string;
    newGoalTitlePlaceholder: string;
    newGoalStartDateAria: string;
    saveGoal: string;
    savingGoal: string;
    noActiveGoals: string;
    periodContinues: string;
    editTitleAria: (title: string) => string;
    savingTitleAria: (title: string) => string;
    saveTitleAria: (title: string) => string;
    saveTitle: string;
    savingTitle: string;
    cancelEditAria: (title: string) => string;
    cancelEditTitle: string;
    editGoalAria: (title: string) => string;
    editGoalTitle: string;
    deactivatingAria: (title: string) => string;
    alreadyDeactivatedAria: (title: string) => string;
    deactivateAria: (title: string) => string;
    deactivatingTitle: string;
    alreadyDeactivatedTitle: string;
    deactivateTitle: (date: string) => string;
  };
  dailyRecord: {
    heading: string;
    dateHeader: string;
    memoHeader: string;
    completedHeader: string;
    memoAria: (date: string) => string;
    completeAria: (date: string, title: string) => string;
  };
  chart: {
    heading: string;
    loading: string;
    failed: string;
    completed: string;
    goalCount: (value: number) => string;
    completedValue: (value: string | number) => string;
  };
  controller: {
    prepareUnavailable: string;
    prepareSuccess: string;
    prepareFailure: string;
    checkUnavailable: string;
    checkFailure: string;
    memoUnavailable: string;
    memoFailure: string;
    createFailure: string;
    createRefreshFailure: string;
    editUnavailable: string;
    editFailure: string;
    editRefreshFailure: string;
    deactivateUnavailable: string;
    alreadyEnded: string;
    deactivateFailure: string;
    deactivateRefreshFailure: string;
    deactivateSuccess: string;
  };
  validation: {
    addUnavailable: string;
    busy: string;
    titleRequired: string;
    startDateInMonth: string;
    activeGoalLimit: (limit: number) => string;
  };
};

export const messages: Record<AppLocale, AppMessages> = {
  ko: {
    app: {
      title: "월간 목표 트래커",
      monthRecord: (month) => `${month} 기록`,
      fallbackNotice: "API 응답을 받지 못해 샘플 데이터를 표시합니다.",
      retry: "다시 시도",
      previousMonth: "이전 달",
      nextMonth: "다음 달",
      monthInput: "기록할 월",
      prepareMonth: "목표 이월",
      signedInAs: (email) => `${email}로 로그인됨`,
      logout: "로그아웃",
      languageLabel: "언어",
      bootstrapLoading: "로그인 상태를 확인하는 중",
      bootstrapError: "로그인 상태를 확인하지 못했습니다.",
    },
    auth: {
      title: "월간 목표 트래커",
      subtitle: "개인 목표를 계정별로 저장합니다.",
      loginTab: "로그인",
      signupTab: "회원가입",
      emailLabel: "이메일",
      emailPlaceholder: "you@example.com",
      passwordLabel: "비밀번호",
      passwordPlaceholder: "8자 이상",
      legacyClaimTokenLabel: "기존 데이터 이전 토큰",
      legacyClaimTokenPlaceholder: "소유자에게만 제공된 토큰(선택)",
      loginButton: "로그인",
      signupButton: "회원가입",
      submitBusy: "처리 중",
      loginFailed: "이메일 또는 비밀번호를 확인해 주세요.",
      signupFailed: "회원가입에 실패했습니다. 이메일 또는 비밀번호를 확인해 주세요.",
      authRateLimited: "요청이 너무 많습니다. 잠시 후 다시 시도해 주세요.",
      signupWeakPassword: "비밀번호는 8자 이상이어야 합니다.",
      signupInvalidEmail: "올바른 이메일 주소를 입력해 주세요.",
      signupInvalidLocale: "지원하지 않는 언어입니다.",
      signupInvalidLegacyClaim: "기존 데이터 이전 토큰을 확인해 주세요.",
      signupLegacyClaimRequired: "기존 데이터가 있어 이전 토큰이 필요합니다.",
      languageHint: "접속 지역을 기준으로 언어를 먼저 제안합니다.",
    },
    status: {
      loading: "불러오는 중",
      api: "API 데이터",
      fallback: "샘플 데이터",
    },
    summary: {
      totalCompleted: "이번 달 완료",
      averageRate: "평균 달성률",
      completedValue: (value) => `${value}개`,
      goalValue: (value) => `${value}개`,
      todayActiveGoals: "오늘 활성 목표",
      dayActiveGoals: (day) => `${day}일 활성 목표`,
    },
    goalPanel: {
      heading: "목표",
      add: "목표 추가",
      newGoalTitleAria: "새 목표 제목",
      newGoalTitlePlaceholder: "새 목표",
      newGoalStartDateAria: "새 목표 시작일",
      saveGoal: "목표 저장",
      savingGoal: "목표 저장 중",
      noActiveGoals: "진행 중인 목표가 없습니다.",
      periodContinues: "계속",
      editTitleAria: (title) => `${title} 제목 수정`,
      savingTitleAria: (title) => `${title} 저장 중`,
      saveTitleAria: (title) => `${title} 저장`,
      saveTitle: "목표 저장",
      savingTitle: "목표 저장 중",
      cancelEditAria: (title) => `${title} 수정 취소`,
      cancelEditTitle: "수정 취소",
      editGoalAria: (title) => `${title} 수정`,
      editGoalTitle: "목표 수정",
      deactivatingAria: (title) => `${title} 종료 중`,
      alreadyDeactivatedAria: (title) => `${title} 이미 종료됨`,
      deactivateAria: (title) => `${title} 종료`,
      deactivatingTitle: "목표 종료 중",
      alreadyDeactivatedTitle: "이미 종료됨",
      deactivateTitle: (date) => `목표 종료 (${date}까지 활성)`,
    },
    dailyRecord: {
      heading: "날짜별 기록",
      dateHeader: "날짜",
      memoHeader: "메모",
      completedHeader: "완료",
      memoAria: (date) => `${date} 메모`,
      completeAria: (date, title) => `${date} ${title} 완료`,
    },
    chart: {
      heading: "일별 완료 개수",
      loading: "차트 불러오는 중",
      failed: "차트를 불러오지 못했습니다.",
      completed: "완료",
      goalCount: (value) => `${value}개 목표`,
      completedValue: (value) => `${value}개`,
    },
    controller: {
      prepareUnavailable: "API 데이터에서만 목표를 이월할 수 있습니다.",
      prepareSuccess: "목표를 이월했습니다.",
      prepareFailure: "목표 이월에 실패했습니다.",
      checkUnavailable: "API 데이터에서만 체크를 저장할 수 있습니다.",
      checkFailure: "체크 저장에 실패했습니다.",
      memoUnavailable: "API 데이터에서만 메모를 저장할 수 있습니다.",
      memoFailure: "메모 저장에 실패했습니다.",
      createFailure: "목표 추가에 실패했습니다.",
      createRefreshFailure: "목표를 추가했지만 화면 갱신에 실패했습니다.",
      editUnavailable: "API 데이터에서만 목표를 수정할 수 있습니다.",
      editFailure: "목표 수정에 실패했습니다.",
      editRefreshFailure: "목표를 수정했지만 화면 갱신에 실패했습니다.",
      deactivateUnavailable: "API 데이터에서만 목표를 종료할 수 있습니다.",
      alreadyEnded: "이미 종료된 목표입니다.",
      deactivateFailure: "목표 종료에 실패했습니다.",
      deactivateRefreshFailure: "목표를 종료했지만 화면 갱신에 실패했습니다.",
      deactivateSuccess: "목표를 종료했습니다.",
    },
    validation: {
      addUnavailable: "API 데이터에서만 목표를 추가할 수 있습니다.",
      busy: "다른 저장 작업이 끝난 뒤 다시 시도해 주세요.",
      titleRequired: "목표 제목을 입력해 주세요.",
      startDateInMonth: "시작일은 선택한 월 안에서 골라 주세요.",
      activeGoalLimit: (limit) =>
        `할일은 날짜별로 최대 ${limit}개까지 등록할 수 있습니다.`,
    },
  },
  en: {
    app: {
      title: "Monthly Goal Tracker",
      monthRecord: (month) => `${month} records`,
      fallbackNotice: "Showing sample data because the API did not respond.",
      retry: "Retry",
      previousMonth: "Previous month",
      nextMonth: "Next month",
      monthInput: "Month to track",
      prepareMonth: "Carry goals forward",
      signedInAs: (email) => `Signed in as ${email}`,
      logout: "Log out",
      languageLabel: "Language",
      bootstrapLoading: "Checking your session",
      bootstrapError: "Could not check your session.",
    },
    auth: {
      title: "Monthly Goal Tracker",
      subtitle: "Save personal goals under your own account.",
      loginTab: "Log in",
      signupTab: "Sign up",
      emailLabel: "Email",
      emailPlaceholder: "you@example.com",
      passwordLabel: "Password",
      passwordPlaceholder: "At least 8 characters",
      legacyClaimTokenLabel: "Existing data claim token",
      legacyClaimTokenPlaceholder: "Owner token (optional)",
      loginButton: "Log in",
      signupButton: "Sign up",
      submitBusy: "Working",
      loginFailed: "Check your email or password.",
      signupFailed: "Sign-up failed. Check your email or password.",
      authRateLimited: "Too many attempts. Try again shortly.",
      signupWeakPassword: "Password must be at least 8 characters.",
      signupInvalidEmail: "Enter a valid email address.",
      signupInvalidLocale: "This language is not supported.",
      signupInvalidLegacyClaim: "Check the existing data claim token.",
      signupLegacyClaimRequired: "Existing data needs a claim token.",
      languageHint: "Language is suggested from your connection region first.",
    },
    status: {
      loading: "Loading",
      api: "API data",
      fallback: "Sample data",
    },
    summary: {
      totalCompleted: "Completed this month",
      averageRate: "Average completion",
      completedValue: (value) => String(value),
      goalValue: (value) => String(value),
      todayActiveGoals: "Active goals today",
      dayActiveGoals: (day) => `Active goals on day ${day}`,
    },
    goalPanel: {
      heading: "Goals",
      add: "Add goal",
      newGoalTitleAria: "New goal title",
      newGoalTitlePlaceholder: "New goal",
      newGoalStartDateAria: "New goal start date",
      saveGoal: "Save goal",
      savingGoal: "Saving goal",
      noActiveGoals: "No active goals.",
      periodContinues: "Ongoing",
      editTitleAria: (title) => `Edit ${title} title`,
      savingTitleAria: (title) => `Saving ${title}`,
      saveTitleAria: (title) => `Save ${title}`,
      saveTitle: "Save goal",
      savingTitle: "Saving goal",
      cancelEditAria: (title) => `Cancel editing ${title}`,
      cancelEditTitle: "Cancel edit",
      editGoalAria: (title) => `Edit ${title}`,
      editGoalTitle: "Edit goal",
      deactivatingAria: (title) => `Ending ${title}`,
      alreadyDeactivatedAria: (title) => `${title} already ended`,
      deactivateAria: (title) => `End ${title}`,
      deactivatingTitle: "Ending goal",
      alreadyDeactivatedTitle: "Already ended",
      deactivateTitle: (date) => `End goal (active through ${date})`,
    },
    dailyRecord: {
      heading: "Daily records",
      dateHeader: "Date",
      memoHeader: "Memo",
      completedHeader: "Done",
      memoAria: (date) => `${date} memo`,
      completeAria: (date, title) => `${date} ${title} completed`,
    },
    chart: {
      heading: "Daily completions",
      loading: "Loading chart",
      failed: "Could not load the chart.",
      completed: "Completed",
      goalCount: (value) => `${value} ${value === 1 ? "goal" : "goals"}`,
      completedValue: (value) => String(value),
    },
    controller: {
      prepareUnavailable: "Goals can only be carried forward from API data.",
      prepareSuccess: "Goals were carried forward.",
      prepareFailure: "Could not carry goals forward.",
      checkUnavailable: "Checks can only be saved to API data.",
      checkFailure: "Could not save the check.",
      memoUnavailable: "Memos can only be saved to API data.",
      memoFailure: "Could not save the memo.",
      createFailure: "Could not add the goal.",
      createRefreshFailure: "The goal was added, but the screen could not refresh.",
      editUnavailable: "Goals can only be edited in API data.",
      editFailure: "Could not update the goal.",
      editRefreshFailure: "The goal was updated, but the screen could not refresh.",
      deactivateUnavailable: "Goals can only be ended in API data.",
      alreadyEnded: "This goal has already ended.",
      deactivateFailure: "Could not end the goal.",
      deactivateRefreshFailure: "The goal was ended, but the screen could not refresh.",
      deactivateSuccess: "Goal ended.",
    },
    validation: {
      addUnavailable: "Goals can only be added to API data.",
      busy: "Try again after the current save finishes.",
      titleRequired: "Enter a goal title.",
      startDateInMonth: "Choose a start date within the selected month.",
      activeGoalLimit: (limit) =>
        `You can register up to ${limit} tasks per date.`,
    },
  },
};

export function messagesForLocale(locale: AppLocale) {
  return messages[locale];
}

export function normalizeLocale(value: string | null | undefined): AppLocale {
  return value === "en" ? "en" : "ko";
}
