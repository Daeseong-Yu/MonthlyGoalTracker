import {
  ArrowRight,
  BarChart3,
  CalendarCheck2,
  Check,
  Moon,
  NotebookPen,
  Sun,
  Target,
} from "lucide-react";

import type { AppMessages } from "./i18n";
import type { AppLocale, ResolvedTheme } from "./types";

type LandingPageProps = {
  healthLabel: string;
  healthTone: "healthy" | "checking" | "unhealthy";
  locale: AppLocale;
  messages: AppMessages;
  resolvedTheme: ResolvedTheme;
  onLocaleChange: (locale: AppLocale) => void;
  onOpenAuth: () => void;
  onStartPreview: () => void;
  onThemeToggle: () => void;
};

export default function LandingPage({
  healthLabel,
  healthTone,
  locale,
  messages,
  resolvedTheme,
  onLocaleChange,
  onOpenAuth,
  onStartPreview,
  onThemeToggle,
}: LandingPageProps) {
  const landing = messages.landing;
  const themeToggleLabel =
    resolvedTheme === "light"
      ? landing.switchToDark
      : landing.switchToLight;

  return (
    <main className="landing-root" id="landing-top">
      <div className="landing-glow landing-glow--one" aria-hidden="true" />
      <div className="landing-glow landing-glow--two" aria-hidden="true" />

      <nav className="landing-nav" aria-label={landing.navigationLabel}>
        <a className="landing-brand" href="#landing-top">
          <span className="landing-brand-mark" aria-hidden="true">
            <Target size={17} strokeWidth={2.4} />
          </span>
          <span>Monthly Goal Tracker</span>
        </a>

        <div className="landing-nav-links">
          <a href="#landing-features">{landing.navFeatures}</a>
          <a href="#product-preview">{landing.navPreview}</a>
          <button type="button" onClick={onOpenAuth}>
            {landing.navAccount}
          </button>
        </div>

        <div className="landing-nav-actions">
          <div
            className="landing-language"
            role="group"
            aria-label={messages.app.languageLabel}
          >
            {(["ko", "en"] as const).map((option) => (
              <button
                key={option}
                className={locale === option ? "is-active" : ""}
                type="button"
                aria-label={`${messages.app.languageLabel} ${option.toUpperCase()}`}
                aria-pressed={locale === option}
                onClick={() => onLocaleChange(option)}
              >
                {option.toUpperCase()}
              </button>
            ))}
          </div>
          <span className="landing-nav-divider" aria-hidden="true" />
          <button
            className="landing-icon-button"
            type="button"
            aria-label={themeToggleLabel}
            title={themeToggleLabel}
            onClick={onThemeToggle}
          >
            {resolvedTheme === "light" ? <Moon size={17} /> : <Sun size={17} />}
          </button>
          <button
            className="landing-login-button"
            type="button"
            aria-label={messages.app.login}
            onClick={onOpenAuth}
          >
            {messages.app.login}
          </button>
          <button
            className="landing-nav-cta"
            type="button"
            aria-label={landing.previewAction}
            onClick={onStartPreview}
          >
            {landing.navStart}
          </button>
        </div>
      </nav>

      <section className="landing-hero" aria-labelledby="landing-heading">
        <div className="landing-eyebrow">
          <span className={`landing-health-dot is-${healthTone}`} />
          <span>{healthLabel}</span>
          <span aria-hidden="true">·</span>
          <span>{landing.eyebrow}</span>
        </div>

        <h1 id="landing-heading" className="landing-heading">
          <span>{landing.headlineLead}</span>
          <span className="landing-heading-emphasis">
            {landing.headlineEmphasis}
          </span>
        </h1>
        <p className="landing-subtitle">{landing.subtitle}</p>

        <div className="landing-hero-actions">
          <button
            className="landing-primary-action"
            type="button"
            aria-label={landing.previewAction}
            onClick={onStartPreview}
          >
            <span>{landing.previewAction}</span>
            <ArrowRight size={17} />
          </button>
          <button
            className="landing-secondary-action"
            type="button"
            aria-label={landing.loginAction}
            onClick={onOpenAuth}
          >
            {landing.loginAction}
          </button>
        </div>

        <div className="landing-feature-strip" id="landing-features">
          <span>
            <CalendarCheck2 size={15} />
            {landing.featureMonthly}
          </span>
          <span>
            <NotebookPen size={15} />
            {landing.featureDaily}
          </span>
          <span>
            <BarChart3 size={15} />
            {landing.featureInsight}
          </span>
        </div>
      </section>

      <section
        className="landing-product-frame"
        id="product-preview"
        aria-label={landing.productPreviewLabel}
      >
        <div className="landing-window-bar">
          <div className="landing-window-brand">
            <span className="landing-window-logo" aria-hidden="true">
              <Target size={13} />
            </span>
            <span>Monthly Goal Tracker</span>
          </div>
          <div className="landing-window-dots" aria-hidden="true">
            <span />
            <span />
            <span />
          </div>
          <span className="landing-window-badge">{landing.previewBadge}</span>
        </div>

        <div className="landing-dashboard-preview">
          <aside className="landing-preview-sidebar" aria-hidden="true">
            <p className="landing-preview-overline">{landing.previewMonth}</p>
            <div className="landing-preview-nav-item is-active">
              <span />
              {messages.app.navDashboard}
            </div>
            <div className="landing-preview-nav-item">
              <span />
              {messages.app.navGoals}
            </div>
            <div className="landing-preview-nav-item">
              <span />
              {messages.app.navRecords}
            </div>
            <div className="landing-preview-sidebar-spacer" />
            <div className="landing-preview-mini-card">
              <span>{landing.previewFocus}</span>
              <strong>03</strong>
            </div>
          </aside>

          <div className="landing-preview-content">
            <div className="landing-preview-header">
              <div>
                <p>{landing.previewOverline}</p>
                <h2>{landing.previewHeading}</h2>
              </div>
              <span>{landing.previewToday}</span>
            </div>

            <div className="landing-preview-metrics">
              <PreviewMetric
                label={messages.summary.totalCompleted}
                value="18"
                change="+12%"
              />
              <PreviewMetric
                label={messages.summary.averageRate}
                value="76%"
                change="+8%"
              />
              <PreviewMetric
                label={messages.summary.todayActiveGoals}
                value="3"
                change={landing.previewOnTrack}
              />
            </div>

            <div className="landing-preview-grid">
              <div className="landing-preview-chart-card">
                <div className="landing-preview-card-heading">
                  <div>
                    <strong>{messages.chart.heading}</strong>
                    <span>{landing.previewTrend}</span>
                  </div>
                  <span className="landing-preview-chip">{landing.previewWeek}</span>
                </div>
                <div className="landing-chart" aria-hidden="true">
                  {Array.from({ length: 14 }, (_, index) => (
                    <span key={index} />
                  ))}
                </div>
              </div>

              <div className="landing-preview-goals-card">
                <div className="landing-preview-card-heading">
                  <strong>{messages.goalPanel.heading}</strong>
                  <span className="landing-preview-add">+</span>
                </div>
                {[82, 64, 38].map((progress, index) => (
                  <div className="landing-preview-goal" key={progress}>
                    <span className="landing-preview-check">
                      {index === 0 ? <Check size={11} /> : null}
                    </span>
                    <div>
                      <span>{landing.previewGoalLabels[index]}</span>
                      <i>
                        <b style={{ width: `${progress}%` }} />
                      </i>
                    </div>
                    <strong>{progress}%</strong>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </section>

      <p className="landing-footnote">{landing.footnote}</p>
    </main>
  );
}

function PreviewMetric({
  change,
  label,
  value,
}: {
  change: string;
  label: string;
  value: string;
}) {
  return (
    <div className="landing-preview-metric">
      <div>
        <span>{label}</span>
        <small>{change}</small>
      </div>
      <strong>{value}</strong>
    </div>
  );
}
