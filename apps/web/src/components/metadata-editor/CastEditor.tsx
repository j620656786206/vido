/**
 * CastEditor Component (Story 3.8 - AC1)
 * Add/remove cast members with input field
 */

import { useState, useCallback } from 'react';
import { X } from 'lucide-react';
import { cn } from '../../lib/utils';

export interface CastEditorProps {
  cast: string[];
  onAdd: (name: string) => void;
  onRemove: (name: string) => void;
  label?: string;
  placeholder?: string;
}

export function CastEditor({
  cast,
  onAdd,
  onRemove,
  label = '演員',
  placeholder = '輸入演員名稱後按 Enter',
}: CastEditorProps) {
  const [inputValue, setInputValue] = useState('');

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        const trimmed = inputValue.trim();
        if (trimmed && !cast.includes(trimmed)) {
          onAdd(trimmed);
          setInputValue('');
        }
      }
    },
    [inputValue, cast, onAdd]
  );

  return (
    <div>
      <label className="block text-sm font-medium text-slate-300 mb-1">{label}</label>
      <div className="flex flex-wrap gap-2 mb-2" data-testid="cast-list">
        {cast.map((actor) => (
          <span
            key={actor}
            className="inline-flex items-center gap-1 px-2 py-1 bg-slate-800 text-white rounded-lg text-sm"
          >
            {actor}
            <button
              type="button"
              onClick={() => onRemove(actor)}
              className="text-slate-400 hover:text-red-400 transition-colors"
              aria-label={`移除 ${actor}`}
            >
              <X className="h-3 w-3" />
            </button>
          </span>
        ))}
      </div>
      <input
        type="text"
        value={inputValue}
        onChange={(e) => setInputValue(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        className={cn(
          'w-full px-4 py-2',
          'bg-slate-800 border border-slate-700 rounded-lg',
          'text-white placeholder-slate-400',
          'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
          'transition-colors'
        )}
        data-testid="cast-input"
      />
    </div>
  );
}

export default CastEditor;
