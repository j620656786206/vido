import type { TVShowDetails } from '../../types/tmdb';
import { cn } from '../../lib/utils';

export interface TVShowInfoProps {
  show: TVShowDetails;
  className?: string;
}

// Task 6.4: Status translations for Traditional Chinese
const TV_STATUS_MAP: Record<string, string> = {
  'Returning Series': '回歸中',
  'Ended': '已完結',
  'Canceled': '已取消',
  'In Production': '製作中',
  'Planned': '計畫中',
  'Pilot': '試播中',
};

/**
 * TVShowInfo - TV show specific information section
 * AC #2: TV show details display
 */
export function TVShowInfo({ show, className }: TVShowInfoProps) {
  // Task 6.4: Translate status
  const statusText = TV_STATUS_MAP[show.status] || show.status;

  // Format date for Traditional Chinese
  const formatDate = (dateStr: string | null | undefined): string => {
    if (!dateStr) return '未知';
    try {
      return new Date(dateStr).toLocaleDateString('zh-TW', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <div className={cn('mt-6', className)} data-testid="tv-show-info">
      <h3 className="mb-3 text-sm font-semibold text-gray-400">影集資訊</h3>

      <div className="space-y-2 text-sm">
        {/* Task 6.2: Number of seasons and episodes */}
        <InfoRow
          label="季數"
          value={`${show.number_of_seasons} 季`}
          testId="seasons-count"
        />
        <InfoRow
          label="集數"
          value={`${show.number_of_episodes} 集`}
          testId="episodes-count"
        />

        {/* Task 6.4: Status */}
        <InfoRow
          label="狀態"
          value={statusText}
          testId="show-status"
        />

        {/* Task 6.3: First air date and last air date */}
        <InfoRow
          label="首播日期"
          value={formatDate(show.first_air_date)}
          testId="first-air-date"
        />
        {show.last_air_date && (
          <InfoRow
            label="最新集數"
            value={formatDate(show.last_air_date)}
            testId="last-air-date"
          />
        )}

        {/* Task 6.5: Networks/streaming platforms */}
        {show.networks && show.networks.length > 0 && (
          <InfoRow
            label="播出平台"
            value={show.networks.map((n) => n.name).join(', ')}
            testId="networks"
          />
        )}

        {/* Show type (Scripted, Documentary, etc.) */}
        {show.type && (
          <InfoRow
            label="類型"
            value={show.type}
            testId="show-type"
          />
        )}
      </div>
    </div>
  );
}

interface InfoRowProps {
  label: string;
  value: string;
  testId?: string;
}

function InfoRow({ label, value, testId }: InfoRowProps) {
  return (
    <div className="flex justify-between" data-testid={testId}>
      <span className="text-gray-400">{label}</span>
      <span className="text-white">{value}</span>
    </div>
  );
}

export default TVShowInfo;
