import {
  Ban,
  Check,
  CheckCircle2,
  LoaderCircle,
  MoreVertical,
  Pencil,
  Plus,
  X,
} from "lucide-react";
import type { FormEvent } from "react";

import { formatGoalPeriod, shortDate } from "./appDisplay";
import {
  deactivationDateForGoal,
  monthEndDate,
  monthStartDate,
} from "./monthLogic";
import type { Goal } from "./types";

type GoalPanelProps = {
  canSaveChanges: boolean;
  deactivatingGoalIDs: number[];
  editingGoalID: number | null;
  editingGoalTitle: string;
  goalProgress: Record<number, GoalProgress>;
  goalFormOpen: boolean;
  goalListReferenceDate: string;
  isMutatingMonth: boolean;
  labels?: GoalPanelLabels;
  month: string;
  newGoalStartDate: string;
  newGoalTitle: string;
  saveError?: string | null;
  saveMessage?: string | null;
  savingGoal: boolean;
  savingGoalTitle: boolean;
  visibleGoals: Goal[];
  onCancelEditingGoal: () => void;
  onDeactivateGoal: (goal: Goal) => void;
  onEditingGoalTitleChange: (title: string) => void;
  onNewGoalStartDateChange: (date: string) => void;
  onNewGoalTitleChange: (title: string) => void;
  onStartEditingGoal: (goal: Goal) => void;
  onSubmitGoalTitle: (
    event: FormEvent<HTMLFormElement>,
    goalID: number,
  ) => void;
  onSubmitNewGoal: (event: FormEvent<HTMLFormElement>) => void;
  onToggleGoalForm: () => void;
};

type GoalProgress = {
  completed: number;
  percent: number;
  total: number;
};

type GoalPanelLabels = {
  heading: string;
  add: string;
  newGoalTitleAria: string;
  newGoalTitlePlaceholder: string;
  newGoalStartDateAria: string;
  saveGoal: string;
  savingGoal: string;
  endedStatus: string;
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

const defaultLabels: GoalPanelLabels = {
  heading: "목표",
  add: "목표 추가",
  newGoalTitleAria: "새 목표 제목",
  newGoalTitlePlaceholder: "새 목표",
  newGoalStartDateAria: "새 목표 시작일",
  saveGoal: "목표 저장",
  savingGoal: "목표 저장 중",
  endedStatus: "종료됨",
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
};

export default function GoalPanel({
  canSaveChanges,
  deactivatingGoalIDs,
  editingGoalID,
  editingGoalTitle,
  goalProgress,
  goalFormOpen,
  goalListReferenceDate,
  isMutatingMonth,
  labels = defaultLabels,
  month,
  newGoalStartDate,
  newGoalTitle,
  saveError = null,
  saveMessage = null,
  savingGoal,
  savingGoalTitle,
  visibleGoals,
  onCancelEditingGoal,
  onDeactivateGoal,
  onEditingGoalTitleChange,
  onNewGoalStartDateChange,
  onNewGoalTitleChange,
  onStartEditingGoal,
  onSubmitGoalTitle,
  onSubmitNewGoal,
  onToggleGoalForm,
}: GoalPanelProps) {
  const feedback = saveError ? (
    <p className="text-xs font-semibold feedback-error" role="alert">
      {saveError}
    </p>
  ) : saveMessage ? (
    <p className="text-xs font-semibold feedback-success" role="status">
      {saveMessage}
    </p>
  ) : null;

  return (
    <section className="panel-card panel-card-padded">
      <div className="panel-header panel-header--flush">
        <div>
          <p className="panel-kicker">Goal stack</p>
          <h2 className="panel-heading">
            {labels.heading}
          </h2>
        </div>
        <button
          className="icon-button"
          type="button"
          aria-label={labels.add}
          title={labels.add}
          disabled={!canSaveChanges || isMutatingMonth}
          onClick={onToggleGoalForm}
        >
          <Plus size={18} />
        </button>
      </div>
      {goalFormOpen ? (
        <form
          className="surface-muted mb-4 space-y-2 p-3"
          onSubmit={onSubmitNewGoal}
        >
          <input
            className="field-control h-9 w-full rounded-md px-3 text-sm"
            aria-label={labels.newGoalTitleAria}
            value={newGoalTitle}
            disabled={isMutatingMonth}
            placeholder={labels.newGoalTitlePlaceholder}
            onChange={(event) => onNewGoalTitleChange(event.target.value)}
          />
          <div className="flex gap-2">
            <input
              className="field-control h-9 min-w-0 flex-1 rounded-md px-3 text-sm"
              type="date"
              aria-label={labels.newGoalStartDateAria}
              min={monthStartDate(month)}
              max={monthEndDate(month)}
              value={newGoalStartDate}
              disabled={isMutatingMonth}
              onChange={(event) => onNewGoalStartDateChange(event.target.value)}
            />
            <button
              className="primary-action h-9 min-w-[6.75rem] shrink-0 px-3 text-sm disabled:cursor-not-allowed disabled:opacity-60"
              type="submit"
              aria-label={savingGoal ? labels.savingGoal : labels.saveGoal}
              title={savingGoal ? labels.savingGoal : labels.saveGoal}
              disabled={isMutatingMonth}
            >
              {savingGoal ? (
                <LoaderCircle
                  aria-hidden="true"
                  className="animate-spin"
                  size={18}
                />
              ) : (
                <Plus aria-hidden="true" size={16} />
              )}
              <span>{savingGoal ? labels.savingGoal : labels.saveGoal}</span>
            </button>
          </div>
          {feedback}
        </form>
      ) : null}
      {!goalFormOpen && feedback ? <div className="mb-4">{feedback}</div> : null}
      <div className="space-y-2.5">
        {visibleGoals.length === 0 ? (
          <p className="surface-muted border-dashed px-3 py-4 text-sm font-semibold" style={{ color: "var(--text-muted)" }}>
            {labels.noActiveGoals}
          </p>
        ) : null}
        {visibleGoals.map((goal, goalIndex) => {
          const deactivationDate = deactivationDateForGoal(goal, month);
          const alreadyDeactivated =
            goal.endDate !== null && goal.endDate <= goalListReferenceDate;
          const deactivating = deactivatingGoalIDs.includes(goal.id);
          const progress = goalProgress[goal.id] ?? {
            completed: 0,
            percent: 0,
            total: 0,
          };

          return (
            <div
              key={goal.id}
              className={`surface-muted goal-card ${goalToneClass(goalIndex)} p-3`}
            >
              {editingGoalID === goal.id ? (
                <form
                  className="space-y-2"
                  onSubmit={(event) => onSubmitGoalTitle(event, goal.id)}
                >
                  <input
                    className="field-control h-9 w-full rounded-md px-3 text-sm"
                    aria-label={labels.editTitleAria(goal.title)}
                    value={editingGoalTitle}
                    disabled={isMutatingMonth}
                    onChange={(event) =>
                      onEditingGoalTitleChange(event.target.value)
                    }
                  />
                  <div className="flex items-center justify-between gap-2">
                    <p className="min-w-0 truncate text-xs" style={{ color: "var(--text-muted)" }}>
                      {formatGoalPeriod(goal, labels.periodContinues)}
                    </p>
                    <div className="flex shrink-0 gap-1">
                      <button
                        className="mini-icon-button"
                        type="submit"
                        aria-label={
                          savingGoalTitle
                            ? labels.savingTitleAria(goal.title)
                            : labels.saveTitleAria(goal.title)
                        }
                        title={savingGoalTitle ? labels.savingTitle : labels.saveTitle}
                        disabled={isMutatingMonth}
                      >
                        {savingGoalTitle ? (
                          <LoaderCircle
                            aria-hidden="true"
                            className="animate-spin"
                            size={15}
                          />
                        ) : (
                          <Check size={15} />
                        )}
                      </button>
                      <button
                        className="mini-icon-button"
                        type="button"
                        aria-label={labels.cancelEditAria(goal.title)}
                        title={labels.cancelEditTitle}
                        disabled={isMutatingMonth}
                        onClick={onCancelEditingGoal}
                      >
                        <X size={15} />
                      </button>
                    </div>
                  </div>
                </form>
              ) : (
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold" style={{ color: "var(--text-primary)" }}>
                      {goal.title}
                    </p>
                    <div className="mt-1 flex flex-wrap items-center gap-1.5">
                      <p className="text-xs" style={{ color: "var(--text-muted)" }}>
                        {formatGoalPeriod(goal, labels.periodContinues)}
                      </p>
                      {alreadyDeactivated ? (
                        <span className="rounded-md border px-1.5 py-0.5 text-[11px] font-semibold" style={{ borderColor: "var(--border-subtle)", background: "var(--control-bg)", color: "var(--text-secondary)" }}>
                          {labels.endedStatus}
                        </span>
                      ) : null}
                    </div>
                  </div>
                  <div className="goal-card-meta">
                    <p className="goal-card-count">
                      {progress.completed} / {Math.max(progress.total, 0)}
                    </p>
                    <p className="goal-card-percent">{progress.percent}%</p>
                    <div className="goal-card-actions">
                      <button
                        className="mini-icon-button"
                        type="button"
                        aria-label={labels.editGoalAria(goal.title)}
                        title={labels.editGoalTitle}
                        disabled={!canSaveChanges || isMutatingMonth}
                        onClick={() => onStartEditingGoal(goal)}
                      >
                        <Pencil size={15} />
                      </button>
                      <button
                        className="mini-icon-button"
                        type="button"
                        aria-label={
                          deactivating
                            ? labels.deactivatingAria(goal.title)
                            : alreadyDeactivated
                            ? labels.alreadyDeactivatedAria(goal.title)
                            : labels.deactivateAria(goal.title)
                        }
                        title={
                          deactivating
                            ? labels.deactivatingTitle
                            : alreadyDeactivated
                            ? labels.alreadyDeactivatedTitle
                            : labels.deactivateTitle(shortDate(deactivationDate))
                        }
                        disabled={
                          !canSaveChanges ||
                          isMutatingMonth ||
                          deactivating ||
                          alreadyDeactivated
                        }
                        onClick={() => onDeactivateGoal(goal)}
                      >
                        {deactivating ? (
                          <LoaderCircle
                            aria-hidden="true"
                            className="animate-spin"
                            size={15}
                          />
                        ) : alreadyDeactivated ? (
                          <CheckCircle2 size={15} />
                        ) : (
                          <Ban size={15} />
                        )}
                      </button>
                      <MoreVertical
                        aria-hidden="true"
                        size={16}
                        style={{ color: "var(--text-muted)" }}
                      />
                    </div>
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </section>
  );
}

function goalToneClass(index: number) {
  return ["goal-card--sage", "goal-card--denim", "goal-card--mauve", "goal-card--rose", "goal-card--slate"][
    index % 5
  ];
}
