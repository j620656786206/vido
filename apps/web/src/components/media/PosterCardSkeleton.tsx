export function PosterCardSkeleton() {
  return (
    <div className="animate-pulse">
      <div className="aspect-[2/3] rounded-lg bg-gray-700" />
      <div className="mt-2 space-y-1">
        <div className="h-4 w-3/4 rounded bg-gray-700" />
        <div className="h-3 w-1/4 rounded bg-gray-700" />
      </div>
    </div>
  );
}
