// Implements: <screen-section — pending epic-19-8 mapping>
/**
 * 外觀 — the theme control (日巡 light-theme story).
 *
 * Two named themes, not a switch. 夜行/日巡 are the product's own words for the
 * two palettes, and a labelled pair states what a toggle only implies: this is
 * a choice between two designed worlds, not "dark mode on/off".
 *
 * ⚖️ A3 (Alexyu, 2026-08-26): with no stored choice the app FOLLOWS THE OS, and
 * keeps following it until the user picks a side. That is why this panel says
 * out loud which state it is in — a user whose Mac just went dark at sunset
 * should be able to see that Vido followed on purpose, and that picking a theme
 * here stops that from happening again.
 */
import { Moon, Sun } from 'lucide-react';
import { useTheme, type Theme } from '../../hooks/useTheme';
import { cn } from '../../lib/utils';

const OPTIONS: { value: Theme; label: string; caption: string; Icon: typeof Sun }[] = [
  { value: 'dark', label: '夜行', caption: '墨綠底 · 宣紙白字', Icon: Moon },
  { value: 'light', label: '日巡', caption: '宣紙底 · 松煙墨字', Icon: Sun },
];

export function AppearanceSettings() {
  const [theme, setTheme] = useTheme();
  // Reading storage directly rather than through the hook: the hook reports the
  // RESOLVED theme, and this panel needs to say whether that resolution came
  // from the user or from the OS.
  const hasChosen = readHasChosen();

  return (
    <section data-testid="appearance-settings" aria-labelledby="appearance-title">
      <h2 id="appearance-title" className="text-lg font-semibold text-[var(--text-primary)]">
        外觀
      </h2>
      <p className="mt-1 text-sm text-[var(--text-secondary)]">
        {hasChosen ? '已依你的選擇顯示。' : '目前跟隨系統設定；選了其中一個之後就不再跟隨。'}
      </p>

      <div
        role="radiogroup"
        aria-labelledby="appearance-title"
        className="mt-4 grid gap-3 sm:grid-cols-2"
      >
        {OPTIONS.map(({ value, label, caption, Icon }) => {
          const active = theme === value;
          return (
            <button
              key={value}
              type="button"
              role="radio"
              aria-checked={active}
              data-testid={`theme-option-${value}`}
              onClick={() => setTheme(value)}
              className={cn(
                'flex min-h-[44px] items-center gap-3 rounded-[var(--radius-lg)] border px-4 py-3 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
                active
                  ? 'border-[var(--accent-primary)] bg-[var(--accent-subtle)]'
                  : 'border-[var(--border-subtle)] hover:bg-[var(--bg-tertiary)]'
              )}
            >
              <Icon
                className={cn(
                  'h-5 w-5 shrink-0',
                  active ? 'text-[var(--accent-text)]' : 'text-[var(--text-muted)]'
                )}
                aria-hidden="true"
              />
              <span className="min-w-0">
                <span
                  className={cn(
                    'block text-sm font-semibold',
                    active ? 'text-[var(--accent-text)]' : 'text-[var(--text-primary)]'
                  )}
                >
                  {label}
                </span>
                <span className="block text-xs text-[var(--text-muted)]">{caption}</span>
              </span>
            </button>
          );
        })}
      </div>

      {/* The honest caveat: posters and key art are the product's主要 content and
          they were shot for a dark ground. Saying so is cheaper than a user
          discovering it and assuming the light theme is unfinished. */}
      <p className="mt-4 text-xs text-[var(--text-muted)]">
        海報與劇照在深色底上對比更好，因此夜行是預設。
      </p>
    </section>
  );
}

function readHasChosen(): boolean {
  try {
    const raw = localStorage.getItem('vido:theme');
    return raw === 'light' || raw === 'dark';
  } catch {
    return false;
  }
}
