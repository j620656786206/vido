import { createFileRoute, notFound, useNavigate } from '@tanstack/react-router';
import { LocalDetailV2 } from '../../components/media/LocalDetailV2';
import { TMDbDetailV2 } from '../../components/media/TMDbDetailV2';
import { DetailNotFoundV2 } from '../../components/media/DetailStatesV2';

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

// dsr-2 AC #8: the route-level 404 (bad type / empty id) and the component-level
// not-found (missing row) used to say two different things for the same event —
// 「404 · 找不到該媒體內容」 here, 「找不到這部影片」 there. One sentence now.
function NotFoundComponent() {
  const navigate = useNavigate();
  // inLibrary={false}: a malformed type/id never pointed at a library item.
  return <DetailNotFoundV2 onBack={() => navigate({ to: '/library' })} inLibrary={false} />;
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
