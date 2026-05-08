import { Check } from "lucide-react";

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
  return (
    <section className="rounded-lg border border-zinc-200 bg-white shadow-soft">
      <div className="border-b border-zinc-200 px-4 py-3">
        <h2 className="text-base font-semibold text-zinc-950">
          {labels.heading}
        </h2>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full min-w-[48rem] table-fixed border-collapse text-left text-sm">
          <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
            <tr>
              <th className="sticky left-0 z-10 w-20 bg-zinc-50 px-3 py-2 font-semibold">
                {labels.dateHeader}
              </th>
              <th className="w-56 px-2 py-2 font-semibold normal-case">
                {labels.memoHeader}
              </th>
              {dailyRecordGoalSlots.map((slotGoals, slotIndex) => (
                <th
                  key={`goal-slot-${slotIndex}`}
                  className={`w-20 py-2 font-semibold normal-case text-zinc-600 ${
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
          <tbody className="divide-y divide-zinc-100">
            {days.map((day) => {
              const point = chartData.find((item) => item.date === day.date);

              return (
                <tr key={day.date} className="hover:bg-[#f6fbf8]">
                  <th className="sticky left-0 z-10 bg-white px-3 py-2 font-medium text-zinc-800">
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
                  {dailyRecordGoalSlots.map((slotGoals, slotIndex) => {
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
                            active
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
                            !active || !canSaveChanges || saving || isMutatingMonth
                          }
                          title={goal.title}
                          onClick={() => onToggleCheck(goal.id, day.date)}
                        >
                          {checked ? <Check size={16} /> : null}
                        </button>
                      </td>
                    );
                  })}
                  <td className="py-2 pl-2 pr-4 text-right text-xs font-semibold text-zinc-800">
                    {point?.completedCount ?? 0}/{point?.activeGoalCount ?? 0}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}
