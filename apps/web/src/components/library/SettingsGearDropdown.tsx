import { useState, useRef, useEffect, useCallback } from 'react';
import { Settings } from 'lucide-react';
import { cn } from '../../lib/utils';

export type PosterDensity = 'small' | 'medium' | 'large';
export type TitleLanguage = 'zh-tw' | 'original';

interface LibraryPreferences {
  density: PosterDensity;
  defaultSort: string;
  titleLanguage: TitleLanguage;
}

interface SettingsGearDropdownProps {
  preferences: LibraryPreferences;
  onPreferencesChange: (prefs: LibraryPreferences) => void;
}

const STORAGE_KEY = 'vido-library-preferences';

export function getStoredPreferences(): LibraryPreferences {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) return JSON.parse(stored);
  } catch {
    // ignore
  }
  return { density: 'medium', defaultSort: 'created_at', titleLanguage: 'zh-tw' };
}

export function savePreferences(prefs: LibraryPreferences) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
}

export function SettingsGearDropdown({
  preferences,
  onPreferencesChange,
}: SettingsGearDropdownProps) {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const handleClickOutside = useCallback((e: MouseEvent) => {
    if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
      setIsOpen(false);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen, handleClickOutside]);

  const updatePref = (updates: Partial<LibraryPreferences>) => {
    const newPrefs = { ...preferences, ...updates };
    onPreferencesChange(newPrefs);
    savePreferences(newPrefs);
  };

  return (
    <div ref={dropdownRef} className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-800 hover:text-white"
        aria-label="媒體庫設定"
        data-testid="settings-gear-button"
      >
        <Settings className="h-5 w-5" />
      </button>

      {isOpen && (
        <div
          data-testid="settings-dropdown"
          className="absolute right-0 top-full z-30 mt-2 w-64 rounded-lg bg-slate-800 p-4 shadow-xl ring-1 ring-slate-700"
        >
          {/* Poster Density */}
          <div className="mb-4">
            <label className="mb-2 block text-xs font-medium text-slate-400">海報大小</label>
            <div className="flex gap-2">
              {(['small', 'medium', 'large'] as const).map((size) => (
                <button
                  key={size}
                  onClick={() => updatePref({ density: size })}
                  className={cn(
                    'flex-1 rounded-md px-3 py-1.5 text-sm transition-colors',
                    preferences.density === size
                      ? 'bg-blue-600 text-white'
                      : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
                  )}
                >
                  {size === 'small' ? '小' : size === 'medium' ? '中' : '大'}
                </button>
              ))}
            </div>
          </div>

          {/* Default Sort */}
          <div className="mb-4">
            <label className="mb-2 block text-xs font-medium text-slate-400">預設排序</label>
            <select
              value={preferences.defaultSort}
              onChange={(e) => updatePref({ defaultSort: e.target.value })}
              className="w-full rounded-md bg-slate-700 px-3 py-2 text-sm text-white outline-none"
            >
              <option value="created_at">加入日期</option>
              <option value="title">標題</option>
              <option value="release_date">上映日期</option>
              <option value="rating">評分</option>
            </select>
          </div>

          {/* Title Language */}
          <div>
            <label className="mb-2 block text-xs font-medium text-slate-400">標題語言</label>
            <div className="flex gap-2">
              {(['zh-tw', 'original'] as const).map((lang) => (
                <button
                  key={lang}
                  onClick={() => updatePref({ titleLanguage: lang })}
                  className={cn(
                    'flex-1 rounded-md px-3 py-1.5 text-sm transition-colors',
                    preferences.titleLanguage === lang
                      ? 'bg-blue-600 text-white'
                      : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
                  )}
                >
                  {lang === 'zh-tw' ? '中文優先' : '原始語言'}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
