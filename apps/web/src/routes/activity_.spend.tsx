import { createFileRoute } from '@tanstack/react-router';
import { SpendHistoryPage } from '../components/activity/SpendHistoryPage';
import { isMonth } from '../components/activity/spendHistoryCopy';
import type { ChartCategory } from '../services/subtitleRunHistoryService';

/**
 * disc-2026-11-spend-history-d — AI 花費明細 at `/activity/spend`.
 *
 * `activity_` (trailing underscore) un-nests the route: the URL sits under
 * /activity, but the page is NOT rendered inside ActivityHub (which has no
 * <Outlet/> and should not grow one for a whole separate page).
 */
interface SpendSearch {
  /** `YYYY-MM`; absent = this month. */
  month?: string;
  category?: ChartCategory;
}

const CATEGORIES: readonly string[] = ['chinese_track', 'translate_english', 'asr', 'legacy'];

/** This month, `YYYY-MM`, by the viewer's clock — only bounds the switcher. */
function thisMonth(now = new Date()): string {
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
}

export const Route = createFileRoute('/activity_/spend')({
  // Rule 26: `2026-10` is not valid JSON, so the default parser hands it over as
  // a string; the enum guard is safe because no category is all digits. A month
  // in the future (a hand-edited URL) is dropped back to this month.
  validateSearch: (search: Record<string, unknown>): SpendSearch => ({
    month: isMonth(search.month) && search.month < thisMonth() ? search.month : undefined,
    category: CATEGORIES.includes(search.category as string)
      ? (search.category as ChartCategory)
      : undefined,
  }),
  component: SpendHistoryRoute,
});

function SpendHistoryRoute() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const currentMonth = thisMonth();
  return (
    <SpendHistoryPage
      month={search.month}
      currentMonth={currentMonth}
      // This month drops the param, so the URL stays `/activity/spend`.
      onMonthChange={(month) =>
        void navigate({
          search: (prev) => ({ ...prev, month: month >= currentMonth ? undefined : month }),
        })
      }
      category={search.category}
      onCategoryChange={(category) =>
        void navigate({ search: (prev) => ({ ...prev, category }), replace: true })
      }
    />
  );
}
