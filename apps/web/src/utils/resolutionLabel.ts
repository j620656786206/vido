const RESOLUTION_MAP: Record<string, string> = {
  '3840x2160': '4K',
  '2560x1440': '2K',
  '1920x1080': '1080p',
  '1280x720': '720p',
};

export function resolutionLabel(resolution: string): string {
  return RESOLUTION_MAP[resolution] ?? resolution;
}

const CHANNEL_MAP: Record<number, string> = {
  2: 'Stereo',
  6: '5.1',
  8: '7.1',
};

export function audioChannelLabel(channels: number): string {
  return CHANNEL_MAP[channels] ?? String(channels);
}
