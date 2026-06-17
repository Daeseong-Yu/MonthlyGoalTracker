import { Check, ChevronDown } from "lucide-react";

import {
  memoInputClassName,
  shortDate,
  weekday,
  weekdayClassName,
} from "./appDisplay";
import {
  activeGoalInSlot,
  checkKey,
  goalSlotTitle,
} from "./goalSlots";
import { isGoalActiveOnDate } from "./monthLogic";
import type {
  AppLocale,
  ChartPointWithLabel,
  DayEntry,
  Goal,
  GoalCheck,
} from "./types";

type DailyRecordTableProps = {
  canSaveChanges: boolean;
  chartData: ChartPointWithLabel[];
  checkableThroughDate: string;
  checks: GoalCheck[];
  dailyRecordGoalSlots: Goal[][];
  days: DayEntry[];
  isMutatingMonth: boolean;
  labels?: DailyRecordLabels;
  locale?: AppLocale;
  savingChecks: string[];
  savingMemos: string[];
  onMemoBlur: (date: string, memo: string) => void;
  onMemoChange: (date: string, memo: string) => void;
  onToggleCheck: (goalId: number, date: string) => void;
};

type DailyRecordLabels = {
  heading: string;
  dateHeader: string;
  memoHeader: string;
  completedHeader: string;
  memoAria: (date: string) => string;
  completeAria: (date: string, title: string) => string;
};

const defaultLabels: DailyRecordLabels = {
  heading: "날짜별 기록",
  dateHeader: "날짜",
  memoHeader: "메모",
  completedHeader: "완료",
  memoAria: (date) => `${date} 메모`,
  completeAria: (date, title) => `${date} ${title} 완료`,
};

export default function DailyRecordTable({
  canSaveChanges,
  chartData,
  checkableThroughDate,
  checks,
  dailyRecordGoalSlots,
  days,
  isMutatingMonth,
  labels = defaultLabels,
  locale = "ko",
  savingChecks,
  savingMemos,
  onMemoBlur,
  onMemoChange,
  onToggleCheck,
}: DailyRecordTableProps) {
  const visibleGoalSlots = dailyRecordGoalSlots.filter(
    (slotGoals) => slotGoals.length > 0,
  );
  const tableWidthClass =
    visibleGoalSlots.length > 0 ? "min-w-[48rem]" : "min-w-[34rem]";

  return (
    <section className="panel-card overflow-hidden">
      <div className="panel-header">
        <div>
          <p className="panel-kicker">Daily log</p>
          <h2 className="panel-heading">
            {labels.heading}
          </h2>
        </div>
        <span className="panel-count">{days.length}</span>
      </div>

      <div className="table-frame overflow-x-auto">
        <table className={`record-table w-full ${tableWidthClass} table-fixed border-collapse text-left text-sm`}>
          <thead className="text-xs uppercase" style={{ background: "var(--panel-muted)", color: "var(--text-muted)" }}>
            <tr>
              <th className="sticky left-0 z-10 w-20 px-3 py-2 font-semibold" style={{ background: "var(--panel-muted)" }}>
                {labels.dateHeader}
              </th>
              <th className="w-56 px-2 py-2 font-semibold normal-case">
                {labels.memoHeader}
              </th>
              {visibleGoalSlots.map((slotGoals, slotIndex) => (
                <th
                  key={`goal-slot-${slotIndex}`}
                  className={`w-20 py-2 font-semibold normal-case ${
                    slotIndex === 0 ? "pl-4 pr-2" : "px-2"
                  }`}
                  title={goalSlotTitle(slotGoals)}
                >
                  {slotGoals.length > 0 ? (
                    <span className="line-clamp-2">
                      {goalSlotTitle(slotGoals)}
                    </span>
                  ) : null}
                </th>
              ))}
              <th className="w-16 py-2 pl-2 pr-4 text-right font-semibold normal-case">
                {labels.completedHeader}
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[color:var(--border-subtle)]">
            {days.map((day) => {
              const point = chartData.find((item) => item.date === day.date);

              return (
                <tr key={day.date}>
                  <th className="record-date-cell sticky left-0 z-10 px-3 py-2 font-semibold">
                    <span className="block">{shortDate(day.date)}</span>
                    <span className={weekdayClassName(day.date)}>
                      {weekday(day.date, locale)}
                    </span>
                  </th>
                  <td className="px-2 py-2">
                    <input
                      className={memoInputClassName(
                        savingMemos.includes(day.date),
                      )}
                      aria-label={labels.memoAria(shortDate(day.date))}
                      value={day.memo}
                      disabled={
                        !canSaveChanges ||
                        isMutatingMonth ||
                        savingMemos.includes(day.date)
                      }
                      onChange={(event) =>
                        onMemoChange(day.date, event.target.value)
                      }
                      onBlur={(event) =>
                        onMemoBlur(day.date, event.target.value)
                      }
                    />
                  </td>
                  {visibleGoalSlots.map((slotGoals, slotIndex) => {
                    const goal = activeGoalInSlot(slotGoals, day.date);

                    if (!goal) {
                      return (
                        <td
                          key={`goal-slot-${slotIndex}`}
                          className={`py-2 ${
                            slotIndex === 0 ? "pl-4 pr-2" : "px-2"
                          }`}
                        />
                      );
                    }

                    const active = isGoalActiveOnDate(goal, day.date);
                    const checkable =
                      active && day.date <= checkableThroughDate;
                    const checked =
                      active &&
                      checks.some(
                        (check) =>
                          check.goalId === goal.id && check.date === day.date,
                      );
                    const saving = savingChecks.includes(
                      checkKey(goal.id, day.date),
                    );

                    return (
                      <td
                        key={`goal-slot-${slotIndex}`}
                        className={`py-2 ${
                          slotIndex === 0 ? "pl-4 pr-2" : "px-2"
                        }`}
                      >
                        <button
                          className={
                            checkable
                              ? checked
                                ? "check-button checked"
                                : "check-button"
                              : "check-button inactive"
                          }
                          type="button"
                          aria-label={labels.completeAria(
                            shortDate(day.date),
                            goal.title,
                          )}
                          aria-pressed={checked}
                          disabled={
                            !checkable ||
                            !canSaveChanges ||
                            saving ||
                            isMutatingMonth
                          }
                          title={goal.title}
                          onClick={() => onToggleCheck(goal.id, day.date)}
                        >
                          {checked ? <Check size={16} /> : null}
                        </button>
                      </td>
                    );
                  })}
                  <td className="py-2 pl-2 pr-4 text-right text-xs font-semibold" style={{ color: "var(--text-primary)" }}>
                    {point?.completedCount ?? 0}/{point?.activeGoalCount ?? 0}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <div className="record-more-footer">
        <button
          className="record-more-button"
          type="button"
          onClick={(event) => {
            event.currentTarget
              .closest("section")
              ?.querySelector(".table-frame")
              ?.scrollBy({ behavior: "smooth", top: 360 });
          }}
        >
          {locale === "ko" ? "더 보기" : "Show more"}
          <ChevronDown size={15} />
        </button>
      </div>
    </section>
  );
}
