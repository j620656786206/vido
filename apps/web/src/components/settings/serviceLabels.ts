// Design ref: ux-design.pen Screen C8-D (wqcqY) · C8-M (qx8Ma)
// 名稱與說明逐字抄自各卡的 col。
import type { ServiceConnectionStatus, ServiceStatus } from '../../services/serviceStatusService';

export interface ServiceFixHint {
  text: string;
  /** Optional in-app place to go fix it; `label` is the words inside `text` that become the link. */
  link?: { to: string; label: string };
}

export interface ServiceLabel {
  name: string;
  role: string;
  /**
   * Only advice that holds for EVERY reason the service could be unreachable.
   * The backend does not classify errors, so "金鑰錯誤" / "逾時" are guesses and
   * do not belong here. No reliable advice → no hint at all.
   */
  fixHint?: ServiceFixHint;
}

/**
 * The backend names its services in English (`Douban Scraper`, `Wikipedia API`,
 * `AI Parser` — models/degradation.go) and those strings are not for the page.
 * Keyed by `service.name` (the stable `ServiceName*` constants), NOT by
 * displayName: the display strings are free to change, the names are not.
 */
export const SERVICE_LABELS: Record<string, ServiceLabel> = {
  tmdb: {
    name: 'TMDB',
    role: '中繼資料與海報',
    fixHint: {
      text: '到「金鑰設定」確認 TMDB 金鑰。',
      link: { to: '/settings/keys', label: '金鑰設定' },
    },
  },
  douban: { name: '豆瓣', role: '評分（選配）' },
  wikipedia: { name: '維基百科', role: '中繼資料補充（選配）' },
  ai: { name: 'AI 解析', role: '檔名解析與字幕翻譯' },
  qbittorrent: {
    name: 'qBittorrent',
    role: '下載器',
    fixHint: {
      text: '請確認下載器已啟動，或到「連線設定」檢查位址。',
      link: { to: '/settings/connection', label: '連線設定' },
    },
  },
};

/** A service the table does not know yet still renders — by its own displayName, with nothing invented. */
export function getServiceLabel(
  service: Pick<ServiceStatus, 'name' | 'displayName'>
): ServiceLabel {
  return SERVICE_LABELS[service.name] ?? { name: service.displayName, role: '' };
}

/** The words on the status pill; the status-change notice uses the same ones. */
export const STATUS_LABELS: Record<ServiceConnectionStatus, string> = {
  connected: '已連線',
  rate_limited: '速率限制',
  error: '錯誤',
  disconnected: '已斷線',
  unconfigured: '未設定',
};

/** The two states the banner calls broken. rate_limited recovers by itself; unconfigured is a choice. */
export function isBroken(status: ServiceConnectionStatus): boolean {
  return status === 'error' || status === 'disconnected';
}
