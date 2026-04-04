export function PosterCardSkeleton() {
  return (
    <div className="animate-pulse">
      <div className="aspect-[2/3] rounded-lg bg-[var(--bg-tertiary)]" />
      <div className="mt-2 space-y-1">
        <div className="h-4 w-3/4 rounded bg-[var(--bg-tertiary)]" />
        <div className="h-3 w-1/4 rounded bg-[var(--bg-tertiary)]" />
      </div>
    </div>
  );
}
