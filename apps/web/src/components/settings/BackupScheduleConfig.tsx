import { useState, useEffect } from 'react';
import { Clock, Loader2 } from 'lucide-react';
import { useBackupSchedule, useUpdateSchedule } from '../../hooks/useBackups';

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
      setMessage(newEnabled ? '✅ 自動備份已啟用' : '自動備份已停用');
    } catch (err) {
      setEnabled(!newEnabled);
      setFrequency(prevFrequency);
      setMessage(err instanceof Error ? err.message : '更新失敗');
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
      setMessage('✅ 排程設定已儲存');
    } catch (err) {
      setMessage(err instanceof Error ? err.message : '更新失敗');
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 py-4" data-testid="schedule-loading">
        <Loader2 className="h-4 w-4 animate-spin text-slate-400" />
        <span className="text-sm text-slate-400">載入排程設定...</span>
      </div>
    );
  }

  return (
    <div
      className="rounded-lg border border-slate-700 bg-slate-800 p-4"
      data-testid="backup-schedule-config"
    >
      {/* Header with toggle */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Clock className="h-4 w-4 text-slate-400" />
          <span className="text-sm font-medium text-slate-200">自動備份</span>
        </div>
        <button
          onClick={handleToggle}
          disabled={updateSchedule.isPending}
          className={`relative h-6 w-11 rounded-full transition-colors ${enabled ? 'bg-blue-600' : 'bg-slate-600'}`}
          data-testid="schedule-toggle"
          role="switch"
          aria-checked={enabled}
        >
          <span
            className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform ${enabled ? 'translate-x-5' : 'translate-x-0'}`}
          />
        </button>
      </div>

      {/* Schedule options (visible when enabled) */}
      {enabled && (
        <div className="mt-4 space-y-3" data-testid="schedule-options">
          <p className="text-xs text-slate-500">
            {schedule?.nextBackupAt
              ? `下次備份：${new Date(schedule.nextBackupAt).toLocaleString('zh-TW')}`
              : '系統會在指定時間自動執行備份'}
          </p>

          <div className="flex items-center gap-3">
            {/* Frequency */}
            <div className="flex-1">
              <label className="mb-1 block text-xs text-slate-400">備份頻率</label>
              <select
                value={frequency}
                onChange={(e) => setFrequency(e.target.value as 'daily' | 'weekly')}
                className="w-full rounded-lg border border-slate-600 bg-slate-900 px-3 py-1.5 text-sm text-slate-200"
                data-testid="schedule-frequency"
              >
                <option value="daily">每日</option>
                <option value="weekly">每週</option>
              </select>
            </div>

            {/* Time */}
            <div className="w-24">
              <label className="mb-1 block text-xs text-slate-400">備份時間</label>
              <select
                value={hour}
                onChange={(e) => setHour(Number(e.target.value))}
                className="w-full rounded-lg border border-slate-600 bg-slate-900 px-3 py-1.5 text-sm text-slate-200"
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
              <div className="flex-1">
                <label className="mb-1 block text-xs text-slate-400">備份日</label>
                <select
                  value={dayOfWeek}
                  onChange={(e) => setDayOfWeek(Number(e.target.value))}
                  className="w-full rounded-lg border border-slate-600 bg-slate-900 px-3 py-1.5 text-sm text-slate-200"
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
          <p className="text-xs text-slate-500" data-testid="retention-info">
            保留策略：最近 7 個每日備份 + 最近 4 個每週備份
          </p>

          {/* Save button */}
          <button
            onClick={handleSave}
            disabled={updateSchedule.isPending}
            className="flex items-center gap-2 rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-blue-500 disabled:opacity-50"
            data-testid="schedule-save-btn"
          >
            {updateSchedule.isPending && <Loader2 className="h-3 w-3 animate-spin" />}
            儲存排程
          </button>
        </div>
      )}

      {/* Message */}
      {message && (
        <p className="mt-3 text-xs text-slate-400" data-testid="schedule-message">
          {message}
        </p>
      )}
    </div>
  );
}
