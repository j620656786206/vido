import { useState } from 'react';
import { Search, X } from 'lucide-react';
import { cn } from '../../lib/utils';

const LOG_LEVELS = ['ERROR', 'WARN', 'INFO', 'DEBUG'] as const;

const LEVEL_CHIP_STYLES: Record<string, { active: string; inactive: string }> = {
  ERROR: {
    active: 'border-red-400 bg-red-400/20 text-red-300',
    inactive: 'border-slate-600 text-slate-400 hover:border-red-400/50 hover:text-red-300',
  },
  WARN: {
    active: 'border-yellow-400 bg-yellow-400/20 text-yellow-300',
    inactive: 'border-slate-600 text-slate-400 hover:border-yellow-400/50 hover:text-yellow-300',
  },
  INFO: {
    active: 'border-blue-400 bg-blue-400/20 text-blue-300',
    inactive: 'border-slate-600 text-slate-400 hover:border-blue-400/50 hover:text-blue-300',
  },
  DEBUG: {
    active: 'border-gray-400 bg-gray-400/20 text-gray-300',
    inactive: 'border-slate-600 text-slate-400 hover:border-gray-400/50 hover:text-gray-300',
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
      <div className="flex flex-wrap gap-1.5" data-testid="log-level-filters">
        <button
          onClick={() => onLevelChange('')}
          className={cn(
            'rounded-full border px-3 py-1 text-xs font-medium transition-colors',
            level === ''
              ? 'border-slate-300 bg-slate-300/20 text-slate-200'
              : 'border-slate-600 text-slate-400 hover:border-slate-400 hover:text-slate-300'
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
              'rounded-full border px-3 py-1 text-xs font-medium transition-colors',
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
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
        <input
          type="text"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="搜尋關鍵字..."
          className="w-full rounded-lg border border-slate-600 bg-slate-800 py-1.5 pl-9 pr-8 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-400 focus:outline-none"
          data-testid="log-keyword-input"
        />
        {inputValue && (
          <button
            onClick={handleClearKeyword}
            className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300"
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
