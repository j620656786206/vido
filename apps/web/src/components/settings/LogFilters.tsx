// Design ref: ux-design.pen Screen C12-D (K28SdR) · C12-M (dOEbF)；篩到沒結果見 C17-D (Gw61P) · C17-M (J186P)
import { useState } from 'react';
import { Search, X } from 'lucide-react';
import { cn } from '../../lib/utils';

const LOG_LEVELS = ['ERROR', 'WARN', 'INFO', 'DEBUG'] as const;

const LEVEL_CHIP_STYLES: Record<string, { active: string; inactive: string }> = {
  ERROR: {
    active: 'border-[var(--error-text)] bg-[var(--error-tint)] text-[var(--error-text)]',
    inactive:
      'border-[var(--border-subtle)] text-[var(--text-secondary)] hover:border-[var(--error)]/50 hover:text-[var(--error-text)]',
  },
  WARN: {
    active: 'border-[var(--warning-text)] bg-[var(--warning-tint)] text-[var(--warning-text)]',
    inactive:
      'border-[var(--border-subtle)] text-[var(--text-secondary)] hover:border-[var(--warning)]/50 hover:text-[var(--warning-text)]',
  },
  INFO: {
    active: 'border-[var(--info-text)] bg-[var(--info-tint)] text-[var(--info-text)]',
    inactive:
      'border-[var(--border-subtle)] text-[var(--text-secondary)] hover:border-[var(--info)]/50 hover:text-[var(--info-text)]',
  },
  DEBUG: {
    active: 'border-[var(--text-muted)] bg-[var(--bg-tertiary)] text-[var(--text-secondary)]',
    inactive:
      'border-[var(--border-subtle)] text-[var(--text-secondary)] hover:border-[var(--text-muted)]/50 hover:text-[var(--text-secondary)]',
  },
};

interface LogFiltersProps {
  level: string;
  keyword: string;
  onLevelChange: (level: string) => void;
  onKeywordChange: (keyword: string) => void;
}

export function LogFilters({ level, keyword, onLevelChange, onKeywordChange }: LogFiltersProps) {
  const [inputValue, setInputValue] = useState(keyword);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      onKeywordChange(inputValue);
    }
  };

  const handleClearKeyword = () => {
    setInputValue('');
    onKeywordChange('');
  };

  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center" data-testid="log-filters">
      {/* Level filter chips */}
      <div className="flex flex-wrap gap-2" data-testid="log-level-filters">
        <button
          onClick={() => onLevelChange('')}
          className={cn(
            'h-7 rounded-full border px-2 text-xs font-medium transition-colors sm:px-3',
            level === ''
              ? 'border-[var(--text-secondary)] bg-[var(--text-secondary)]/20 text-[var(--text-primary)]'
              : 'border-[var(--border-subtle)] text-[var(--text-secondary)] hover:border-[var(--text-muted)] hover:text-[var(--text-secondary)]'
          )}
          data-testid="log-filter-all"
        >
          全部
        </button>
        {LOG_LEVELS.map((lvl) => (
          <button
            key={lvl}
            onClick={() => onLevelChange(level === lvl ? '' : lvl)}
            className={cn(
              'h-7 rounded-full border px-2 text-xs font-medium transition-colors sm:px-3',
              level === lvl ? LEVEL_CHIP_STYLES[lvl].active : LEVEL_CHIP_STYLES[lvl].inactive
            )}
            data-testid={`log-filter-${lvl.toLowerCase()}`}
          >
            {lvl}
          </button>
        ))}
      </div>

      {/* Keyword search */}
      <div className="relative flex-1 sm:max-w-xs" data-testid="log-keyword-search">
        <Search
          className="absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-[var(--text-muted)]"
          aria-hidden="true"
        />
        <input
          type="text"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="搜尋關鍵字..."
          aria-label="搜尋關鍵字"
          className="h-11 w-full rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] pl-8 pr-8 font-mono text-xs text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-hover)] focus:outline-none sm:h-8"
          data-testid="log-keyword-input"
        />
        {inputValue && (
          <button
            onClick={handleClearKeyword}
            className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--text-secondary)]"
            aria-label="清除搜尋"
            data-testid="log-keyword-clear"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>
    </div>
  );
}
