import { useExploreBlocks } from '../../hooks/useExploreBlocks';
import { ExploreBlock } from './ExploreBlock';

/**
 * ExploreBlocksList — renders every configured explore block in sort_order.
 * Hides itself while loading to avoid shifting layout beneath HeroBanner,
 * and on error (same AC #5 spirit as HeroBanner).
 *
 * Story 10.3 AC #1, #5.
 */
export function ExploreBlocksList() {
  const { data, isLoading, isError } = useExploreBlocks();

  if (isLoading || isError) return null;

  const blocks = data?.blocks ?? [];
  if (blocks.length === 0) return null;

  return (
    <div data-testid="explore-blocks-list" className="bg-[var(--bg-primary)]">
      {blocks.map((block) => (
        <ExploreBlock key={block.id} block={block} />
      ))}
    </div>
  );
}
