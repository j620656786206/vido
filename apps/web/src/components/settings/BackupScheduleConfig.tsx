// Design ref: ux-design.pen Screen C5-D (uhAKd) · C5-M (gEQX4)
import { useState, useEffect } from 'react';
import { Clock, Loader2 } from 'lucide-react';
import { formatLocalDateTime } from '../../utils/formatLocalDateTime';
import { cn } from '../../lib/utils';
import { useBackupSchedule, useUpdateSchedule } from '../../hooks/useBackups';

const FREQUENCIES = [
  { value: 'daily', label: '每日' },
  { value: 'weekly', label: '每週' },
] as const;

const SELECT_CLASS =
  'h-10 w-full rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] px-3 text-sm text-[var(--text-primary)]';

const DAYS_OF_WEEK = [
  { value: 0, label: '週日' },
  { value: 1, label: '週一' },
  { value: 2, label: '週二' },
  { value: 3, label: '週三' },
  { value: 4, label: '週四' },
  { value: 5, label: '週五' },
  { value: 6, label: '週六' },
];

export function BackupScheduleConfig() {
  const { data: schedule, isLoading } = useBackupSchedule();
  const updateSchedule = useUpdateSchedule();
  const [enabled, setEnabled] = useState(false);
  const [frequency, setFrequency] = useState<'daily' | 'weekly' | 'disabled'>('disabled');
  const [hour, setHour] = useState(3);
  const [dayOfWeek, setDayOfWeek] = useState(0);
  const [message, setMessage] = useState<string | null>(null);

  useEffect(() => {
    if (schedule) {
      setEnabled(schedule.enabled);
      setFrequency(schedule.frequency);
      setHour(schedule.hour);
      setDayOfWeek(schedule.dayOfWeek);
    }
  }, [schedule]);

  const handleToggle = async () => {
    if (updateSchedule.isPending) return;
    const newEnabled = !enabled;
    const prevFrequency = frequency;
    const newFrequency = newEnabled ? (frequency === 'disabled' ? 'daily' : frequency) : 'disabled';
    setEnabled(newEnabled);
    setFrequency(newFrequency);
    setMessage(null);
    try {
      await updateSchedule.mutateAsync({
        enabled: newEnabled,
        frequency: newFrequency,
        hour,
        dayOfWeek,
      });
      setMessage(newEnabled ? '自動備份已啟用' : '自動備份已停用');
    } catch {
      setEnabled(!newEnabled);
      setFrequency(prevFrequency);
      setMessage('排程沒有更新成功，請稍後再試。');
    }
  };

  const handleSave = async () => {
    if (updateSchedule.isPending) return;
    setMessage(null);
    try {
      await updateSchedule.mutateAsync({
        enabled,
        frequency,
        hour,
        dayOfWeek,
      });
      setMessage('排程設定已儲存');
    } catch {
      setMessage('排程沒有儲存成功，請稍後再試。');
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 py-4" data-testid="schedule-loading">
        <Loader2 className="h-4 w-4 animate-spin text-[var(--text-secondary)]" />
        <span className="text-sm text-[var(--text-secondary)]">載入排程設定...</span>
      </div>
    );
  }

  // Two options, so a segmented radiogroup (C5 qxfV5) rather than a <select>:
  // roving tabindex + arrow keys, the same keys a native radio group answers.
  const onFrequencyKey = (e: React.KeyboardEvent) => {
    const toggle = ['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'].includes(e.key);
    if (!toggle && e.key !== 'Home' && e.key !== 'End') return;
    e.preventDefault();
    const next: 'daily' | 'weekly' =
      e.key === 'Home'
        ? 'daily'
        : e.key === 'End'
          ? 'weekly'
          : frequency === 'weekly'
            ? 'daily'
            : 'weekly';
    setFrequency(next);
    (
      e.currentTarget.parentElement?.querySelector(`[data-value="${next}"]`) as HTMLElement | null
    )?.focus();
  };

  return (
    <div
      className="rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 sm:p-6"
      data-testid="backup-schedule-config"
    >
      {/* Header with toggle */}
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <Clock className="h-4 w-4 text-[var(--text-secondary)]" aria-hidden="true" />
            <span className="text-sm font-semibold text-[var(--text-primary)]">自動備份</span>
          </div>
          {enabled && (
            <p className="mt-0.5 text-xs text-[var(--text-muted)]" data-testid="schedule-next">
              {schedule?.nextBackupAt
                ? `下次備份：${formatLocalDateTime(schedule.nextBackupAt)}`
                : '系統會在指定時間自動執行備份'}
            </p>
          )}
        </div>
        <button
          onClick={handleToggle}
          disabled={updateSchedule.isPending}
          className={`relative h-6 w-11 shrink-0 rounded-full transition-colors ${enabled ? 'bg-[var(--accent-primary)]' : 'bg-[var(--bg-tertiary)]'}`}
          data-testid="schedule-toggle"
          role="switch"
          aria-checked={enabled}
          aria-label="自動備份"
        >
          <span
            className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-[var(--text-on-accent)] transition-transform ${enabled ? 'translate-x-5' : 'translate-x-0'}`}
            data-testid="schedule-toggle-knob"
          />
        </button>
      </div>

      {/* Schedule options (visible when enabled) */}
      {enabled && (
        <div className="mt-4 space-y-4" data-testid="schedule-options">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:gap-4">
            {/* Frequency */}
            <div className="sm:flex-1">
              <span
                id="schedule-frequency-label"
                className="mb-2 block text-sm font-medium text-[var(--text-secondary)]"
              >
                備份頻率
              </span>
              <div
                role="radiogroup"
                aria-labelledby="schedule-frequency-label"
                className="flex gap-2"
                data-testid="schedule-frequency"
              >
                {FREQUENCIES.map((opt) => {
                  const checked = frequency === opt.value;
                  return (
                    <button
                      key={opt.value}
                      type="button"
                      role="radio"
                      aria-checked={checked}
                      tabIndex={
                        checked || (frequency === 'disabled' && opt.value === 'daily') ? 0 : -1
                      }
                      data-value={opt.value}
                      onClick={() => setFrequency(opt.value)}
                      onKeyDown={onFrequencyKey}
                      className={cn(
                        'h-10 rounded-[var(--radius-md)] border px-4 text-sm text-[var(--text-primary)] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
                        checked
                          ? 'border-[var(--accent-primary)] bg-[var(--accent-subtle)] font-semibold'
                          : 'border-transparent bg-[var(--bg-tertiary)]'
                      )}
                    >
                      {opt.label}
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Time */}
            <div className="sm:flex-1">
              <label
                htmlFor="schedule-hour"
                className="mb-2 block text-sm font-medium text-[var(--text-secondary)]"
              >
                備份時間
              </label>
              <select
                id="schedule-hour"
                value={hour}
                onChange={(e) => setHour(Number(e.target.value))}
                className={cn(SELECT_CLASS, 'font-mono')}
                data-testid="schedule-hour"
              >
                {Array.from({ length: 24 }, (_, i) => (
                  <option key={i} value={i}>
                    {String(i).padStart(2, '0')}:00
                  </option>
                ))}
              </select>
            </div>

            {/* Day of week (only for weekly) */}
            {frequency === 'weekly' && (
              <div className="sm:flex-1">
                <label
                  htmlFor="schedule-day"
                  className="mb-2 block text-sm font-medium text-[var(--text-secondary)]"
                >
                  備份日
                </label>
                <select
                  id="schedule-day"
                  value={dayOfWeek}
                  onChange={(e) => setDayOfWeek(Number(e.target.value))}
                  className={SELECT_CLASS}
                  data-testid="schedule-day"
                >
                  {DAYS_OF_WEEK.map((day) => (
                    <option key={day.value} value={day.value}>
                      {day.label}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>

          {/* Retention info */}
          <p className="text-xs text-[var(--text-muted)]" data-testid="retention-info">
            保留策略：最近 7 個每日備份 + 最近 4 個每週備份
          </p>

          {/* Save button */}
          <button
            onClick={handleSave}
            disabled={updateSchedule.isPending}
            className="flex h-11 w-full items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-xs font-semibold text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-hover)] disabled:opacity-50 sm:h-9 sm:w-auto"
            data-testid="schedule-save-btn"
          >
            {updateSchedule.isPending && <Loader2 className="h-3 w-3 animate-spin" />}
            儲存排程
          </button>
        </div>
      )}

      {/* Message */}
      {message && (
        <p className="mt-3 text-xs text-[var(--text-secondary)]" data-testid="schedule-message">
          {message}
        </p>
      )}
    </div>
  );
}
