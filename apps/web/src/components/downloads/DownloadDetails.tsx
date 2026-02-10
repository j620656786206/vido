import { useDownloadDetails } from '../../hooks/useDownloads';
import { formatSize, formatSpeed, formatDate } from './formatters';

interface DownloadDetailsProps {
  hash: string;
}

export function DownloadDetails({ hash }: DownloadDetailsProps) {
  const { data: details, isLoading, error } = useDownloadDetails(hash);

  if (isLoading) {
    return <div className="py-4 text-center text-sm text-slate-400">載入詳細資料中...</div>;
  }

  if (error) {
    return (
      <div className="py-4 text-center text-sm text-red-400">無法載入詳細資料：{error.message}</div>
    );
  }

  if (!details) return null;

  const fields = [
    { label: '總大小', value: formatSize(details.size) },
    { label: '已下載', value: formatSize(details.downloaded) },
    { label: '做種數', value: `${details.seeds}` },
    { label: '節點數', value: `${details.peers}` },
    { label: '儲存路徑', value: details.savePath },
    { label: '新增時間', value: formatDate(details.addedOn) },
    ...(details.completedOn ? [{ label: '完成時間', value: formatDate(details.completedOn) }] : []),
    { label: '平均下載速度', value: formatSpeed(details.avgDownSpeed) },
    { label: '平均上傳速度', value: formatSpeed(details.avgUpSpeed) },
    { label: '分塊大小', value: formatSize(details.pieceSize) },
    ...(details.timeElapsed > 0
      ? [
          {
            label: '已耗時間',
            value: `${Math.floor(details.timeElapsed / 3600)}h ${Math.floor((details.timeElapsed % 3600) / 60)}m`,
          },
        ]
      : []),
    ...(details.comment ? [{ label: '備註', value: details.comment }] : []),
  ];

  return (
    <div className="grid grid-cols-2 gap-x-8 gap-y-2 pt-3 text-sm" data-testid="download-details">
      {fields.map((field) => (
        <div key={field.label} className="flex justify-between">
          <span className="text-slate-400">{field.label}</span>
          <span className="text-slate-200 truncate ml-2 max-w-[60%] text-right">{field.value}</span>
        </div>
      ))}
    </div>
  );
}
