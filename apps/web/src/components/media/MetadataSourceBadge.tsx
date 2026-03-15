export type MetadataSource = 'tmdb' | 'douban' | 'wikipedia' | 'ai' | 'manual';

interface SourceConfig {
  icon: string;
  label: string;
  color: string;
  bgColor: string;
}

const SOURCE_CONFIG: Record<MetadataSource, SourceConfig> = {
  tmdb: { icon: '🎬', label: 'TMDb', color: '#0d253f', bgColor: 'bg-blue-900/50' },
  douban: { icon: '📗', label: '豆瓣', color: '#00b51d', bgColor: 'bg-green-900/50' },
  wikipedia: { icon: '📖', label: 'Wikipedia', color: '#636466', bgColor: 'bg-gray-700/50' },
  ai: { icon: '🤖', label: 'AI 解析', color: '#7c3aed', bgColor: 'bg-purple-900/50' },
  manual: { icon: '✏️', label: '手動輸入', color: '#f59e0b', bgColor: 'bg-amber-900/50' },
};

export interface MetadataSourceBadgeProps {
  source: string;
  fetchDate?: string;
}

export function MetadataSourceBadge({ source, fetchDate }: MetadataSourceBadgeProps) {
  const config = SOURCE_CONFIG[source as MetadataSource];
  if (!config) return null;

  const tooltipText = fetchDate
    ? `資料來源：${config.label}，於 ${new Date(fetchDate).toLocaleDateString('zh-TW')} 取得`
    : `資料來源：${config.label}`;

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium ${config.bgColor} text-gray-200`}
      title={tooltipText}
      data-testid="metadata-source-badge"
    >
      <span>{config.icon}</span>
      {config.label}
    </span>
  );
}
