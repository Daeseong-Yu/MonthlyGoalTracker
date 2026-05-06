import { Ban, Check, Pencil, Plus, X } from "lucide-react";
import type { FormEvent } from "react";

import { formatGoalPeriod, shortDate } from "./appDisplay";
import { deactivationDateForGoal, monthEndDate, monthStartDate } from "./monthLogic";
import type { Goal } from "./types";

type GoalPanelProps = {
  canSaveChanges: boolean;
  deactivatingGoalIDs: number[];
  editingGoalID: number | null;
  editingGoalTitle: string;
  goalFormOpen: boolean;
  goalListReferenceDate: string;
  isMutatingMonth: boolean;
  month: string;
  newGoalStartDate: string;
  newGoalTitle: string;
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

export default function GoalPanel({
  canSaveChanges,
  deactivatingGoalIDs,
  editingGoalID,
  editingGoalTitle,
  goalFormOpen,
  goalListReferenceDate,
  isMutatingMonth,
  month,
  newGoalStartDate,
  newGoalTitle,
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
  return (
    <section className="rounded-lg border border-zinc-200 bg-white p-4 shadow-soft">
      <div className="mb-4 flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold text-zinc-950">목표</h2>
        <button
          className="icon-button"
          type="button"
          aria-label="목표 추가"
          title="목표 추가"
          disabled={!canSaveChanges || isMutatingMonth}
          onClick={onToggleGoalForm}
        >
          <Plus size={18} />
        </button>
      </div>
      {goalFormOpen ? (
        <form
          className="mb-4 space-y-2 rounded-md border border-zinc-200 bg-zinc-50 p-3"
          onSubmit={onSubmitNewGoal}
        >
          <input
            className="h-9 w-full rounded-md border border-zinc-200 bg-white px-3 text-sm outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100"
            aria-label="새 목표 제목"
            value={newGoalTitle}
            disabled={isMutatingMonth}
            placeholder="새 목표"
            onChange={(event) => onNewGoalTitleChange(event.target.value)}
          />
          <div className="flex gap-2">
            <input
              className="h-9 min-w-0 flex-1 rounded-md border border-zinc-200 bg-white px-3 text-sm outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100"
              type="date"
              aria-label="새 목표 시작일"
              min={monthStartDate(month)}
              max={monthEndDate(month)}
              value={newGoalStartDate}
              disabled={isMutatingMonth}
              onChange={(event) => onNewGoalStartDateChange(event.target.value)}
            />
            <button
              className="icon-button"
              type="submit"
              aria-label="목표 저장"
              title="목표 저장"
              disabled={isMutatingMonth}
            >
              <Plus size={18} />
            </button>
          </div>
        </form>
      ) : null}
      <div className="space-y-3">
        {visibleGoals.length === 0 ? (
          <p className="rounded-md border border-dashed border-zinc-200 bg-zinc-50 px-3 py-4 text-sm font-medium text-zinc-500">
            진행 중인 목표가 없습니다.
          </p>
        ) : null}
        {visibleGoals.map((goal) => {
          const deactivationDate = deactivationDateForGoal(goal, month);
          const alreadyDeactivated =
            goal.endDate !== null && goal.endDate <= goalListReferenceDate;
          const deactivating = deactivatingGoalIDs.includes(goal.id);

          return (
            <div
              key={goal.id}
              className="rounded-md border border-zinc-200 bg-zinc-50 p-3"
            >
              {editingGoalID === goal.id ? (
                <form
                  className="space-y-2"
                  onSubmit={(event) => onSubmitGoalTitle(event, goal.id)}
                >
                  <input
                    className="h-9 w-full rounded-md border border-zinc-200 bg-white px-3 text-sm outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-100"
                    aria-label={`${goal.title} 제목 수정`}
                    value={editingGoalTitle}
                    disabled={isMutatingMonth}
                    onChange={(event) =>
                      onEditingGoalTitleChange(event.target.value)
                    }
                  />
                  <div className="flex items-center justify-between gap-2">
                    <p className="min-w-0 truncate text-xs text-zinc-500">
                      {formatGoalPeriod(goal)}
                    </p>
                    <div className="flex shrink-0 gap-1">
                      <button
                        className="mini-icon-button"
                        type="submit"
                        aria-label={`${goal.title} 저장`}
                        title="목표 저장"
                        disabled={isMutatingMonth}
                      >
                        <Check size={15} />
                      </button>
                      <button
                        className="mini-icon-button"
                        type="button"
                        aria-label={`${goal.title} 수정 취소`}
                        title="수정 취소"
                        disabled={isMutatingMonth}
                        onClick={onCancelEditingGoal}
                      >
                        <X size={15} />
                      </button>
                    </div>
                  </div>
                </form>
              ) : (
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold text-zinc-950">
                      {goal.title}
                    </p>
                    <p className="mt-1 text-xs text-zinc-500">
                      {formatGoalPeriod(goal)}
                    </p>
                  </div>
                  <div className="flex shrink-0 gap-1">
                    <button
                      className="mini-icon-button"
                      type="button"
                      aria-label={`${goal.title} 수정`}
                      title="목표 수정"
                      disabled={!canSaveChanges || isMutatingMonth}
                      onClick={() => onStartEditingGoal(goal)}
                    >
                      <Pencil size={15} />
                    </button>
                    <button
                      className="mini-icon-button"
                      type="button"
                      aria-label={
                        alreadyDeactivated
                          ? `${goal.title} 이미 종료됨`
                          : `${goal.title} 종료`
                      }
                      title={
                        alreadyDeactivated
                          ? "이미 종료됨"
                          : `목표 종료 (${shortDate(deactivationDate)}까지 활성)`
                      }
                      disabled={!canSaveChanges || isMutatingMonth || deactivating}
                      onClick={() => onDeactivateGoal(goal)}
                    >
                      <Ban size={15} />
                    </button>
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
