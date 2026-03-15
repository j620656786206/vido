import { useState } from 'react';

const YOUTUBE_EMBED_BASE = 'https://www.youtube-nocookie.com/embed/';

export interface TrailerEmbedProps {
  videoKey: string;
  title: string;
}

export function TrailerEmbed({ videoKey, title }: TrailerEmbedProps) {
  const [showPlayer, setShowPlayer] = useState(false);

  if (!showPlayer) {
    return (
      <button
        onClick={() => setShowPlayer(true)}
        className="flex w-full items-center justify-center gap-2 rounded-lg bg-slate-800 px-4 py-3 text-sm text-white transition-colors hover:bg-slate-700"
        data-testid="trailer-button"
      >
        <span className="text-lg">▶</span>
        觀看預告片
      </button>
    );
  }

  return (
    <div className="aspect-video w-full" data-testid="trailer-player">
      <iframe
        src={`${YOUTUBE_EMBED_BASE}${videoKey}`}
        title={`${title} 預告片`}
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope"
        allowFullScreen
        className="h-full w-full rounded-lg"
      />
    </div>
  );
}
