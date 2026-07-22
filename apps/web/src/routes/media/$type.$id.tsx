import { createFileRoute, notFound, useNavigate } from '@tanstack/react-router';
import { LocalDetailV2 } from '../../components/media/LocalDetailV2';
import { TMDbDetailV2 } from '../../components/media/TMDbDetailV2';

const validMediaTypes = ['movie', 'tv'] as const;
type ValidMediaType = (typeof validMediaTypes)[number];

function isValidMediaType(type: string): type is ValidMediaType {
  return validMediaTypes.includes(type as ValidMediaType);
}

export type IdKind = 'local-uuid' | 'tmdb-numeric';

// bugfix-10-1 [@contract-v1] AC #2 — A pure positive-integer string is a TMDb
// numeric ID; everything else (UUIDs, mixed strings) routes through the local
// DB path. Widens bugfix-1 [@contract-v0] (UUID-only) to cover homepage TMDb
// items surfaced by Story 10-3 ExploreBlock.
export function classifyId(id: string): IdKind {
  if (/^\d+$/.test(id) && parseInt(id, 10) > 0) return 'tmdb-numeric';
  return 'local-uuid';
}

export const Route = createFileRoute('/media/$type/$id')({
  loader: async ({ params }) => {
    const { type, id } = params;

    if (!isValidMediaType(type)) {
      throw notFound();
    }

    if (!id || id.trim() === '') {
      throw notFound();
    }

    return {
      type: type as ValidMediaType,
      id,
      idKind: classifyId(id),
    };
  },
  notFoundComponent: NotFoundComponent,
  component: MediaDetailRoute,
});

function NotFoundComponent() {
  const navigate = useNavigate();

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <h1 className="mb-4 text-4xl font-bold text-white">404</h1>
        <p className="mb-6 text-[var(--text-secondary)]">找不到該媒體內容</p>
        <button
          onClick={() => navigate({ to: '/library' })}
          className="rounded-lg bg-[var(--accent-primary)] px-4 py-2 text-white hover:bg-[var(--accent-pressed)]"
        >
          返回媒體庫
        </button>
      </div>
    </div>
  );
}

// ux3-cutover-3: legacy LocalDetailView/TMDbDetailView removed — the v2 detail
// components are the only render for both id kinds.
function MediaDetailRoute() {
  const { type, id, idKind } = Route.useLoaderData();

  // bugfix-10-1 — Homepage / search PosterCards emit raw TMDb numeric IDs
  // (Story 10-3 ExploreBlock + Story 2-3 search MediaGrid). Those never resolve
  // against /api/v1/movies/:id (UUID-keyed). Branch off to the TMDb-backed
  // detail render and skip the local-DB hooks entirely.
  if (idKind === 'tmdb-numeric') {
    return <TMDbDetailV2 type={type} tmdbId={parseInt(id, 10)} />;
  }

  return <LocalDetailV2 type={type} id={id} />;
}
