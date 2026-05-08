import {
  CalendarDays,
  CalendarPlus,
  ChevronLeft,
  ChevronRight,
  RefreshCw,
} from "lucide-react";

import {
  formatMonth,
  statusClassName,
  statusLabel,
} from "./appDisplay";
import ChartPanel from "./ChartPanel";
import DailyRecordTable from "./DailyRecordTable";
import GoalPanel from "./GoalPanel";
import MetricSummary from "./MetricSummary";
import { useMonthController } from "./useMonthController";

export default function App() {
  const monthController = useMonthController();

  return (
    <main className="min-h-screen bg-[#f7f8f5] text-zinc-900">
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 py-5 sm:px-6 lg:px-8">
        <header className="flex flex-col gap-4 border-b border-zinc-200 pb-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 className="text-2xl font-semibold tracking-normal text-zinc-950">
              월간 목표 트래커
            </h1>
            <p className="mt-1 flex flex-wrap items-center gap-2 text-sm text-zinc-600">
              <span>{formatMonth(monthController.month)} 기록</span>
              <span
                aria-live="polite"
                className={statusClassName(monthController.loadStatus)}
                role="status"
              >
                {statusLabel(monthController.loadStatus)}
              </span>
            </p>
            {monthController.loadError ? (
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <p className="text-xs font-medium text-amber-700" role="status">
                  API 응답을 받지 못해 샘플 데이터를 표시합니다.
                </p>
                <button
                  className="inline-flex h-8 items-center gap-1.5 rounded-md border border-amber-300 bg-white px-2.5 text-xs font-semibold text-amber-800 shadow-sm transition hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-60"
                  type="button"
                  aria-label="다시 시도"
                  disabled={monthController.isLoading}
                  onClick={() =>
                    void monthController.loadMonth(monthController.month)
                  }
                >
                  <RefreshCw size={14} />
                  다시 시도
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
              aria-label="이전 달"
              title="이전 달"
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
                aria-label="기록할 월"
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
              aria-label="다음 달"
              title="다음 달"
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
              aria-label="목표 이월"
              title="목표 이월"
              disabled={
                !monthController.canSaveChanges ||
                monthController.isMutatingMonth
              }
              onClick={() => void monthController.prepareCurrentMonth()}
            >
              <CalendarPlus size={18} />
            </button>
          </div>
        </header>

        <MetricSummary
          activeGoalCount={monthController.activeMetricGoalCount}
          activeMetricLabel={monthController.activeMetricLabel}
          averageRate={monthController.averageRate}
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
            <ChartPanel
              chartData={monthController.chartData}
              goalCount={monthController.goals.length}
              month={monthController.month}
            />
            <GoalPanel
              canSaveChanges={monthController.canSaveChanges}
              deactivatingGoalIDs={monthController.deactivatingGoalIDs}
              editingGoalID={monthController.editingGoalID}
              editingGoalTitle={monthController.editingGoalTitle}
              goalFormOpen={monthController.goalFormOpen}
              goalListReferenceDate={monthController.goalListReferenceDate}
              isMutatingMonth={monthController.isMutatingMonth}
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
          </aside>
        </section>
      </div>
    </main>
  );
}
