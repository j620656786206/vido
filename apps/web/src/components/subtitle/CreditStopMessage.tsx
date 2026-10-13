// Design ref: ux-design.pen Screen F9b-D-v2 (YHnnC)
/**
 * The F9b-D-v2 banner text: the batch stopped because the AI account has no
 * credit left (disc-2026-10-batch-stop-on-credit-exhausted-b). The batch
 * dialog and the generation workspace wrap it in their own ochre box.
 *
 * The billing link is underlined although F9b draws it in colour only: a link
 * told apart by colour alone fails WCAG 1.4.1 (Sally, 2026-10-13).
 */
import { creditProviderName, safeBillingUrl } from './creditStop';

const NUM = 'font-mono font-semibold tabular-nums text-[var(--text-primary)]';

export function CreditStopMessage({
  provider,
  done,
  paused,
  billingUrl,
}: {
  provider: string | null | undefined;
  done: number;
  paused: number;
  billingUrl: string | null | undefined;
}) {
  const name = creditProviderName(provider);
  const href = safeBillingUrl(billingUrl);
  return (
    <p className="flex flex-wrap items-center gap-x-1 gap-y-0.5 text-sm text-[var(--text-primary)]">
      {name} 帳號沒有額度了 — 已完成
      <span className={NUM}>{done}</span>
      部，剩餘
      <span className={NUM}>{paused}</span>
      部已暫停，儲值後按「下次繼續」
      {href && (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          data-testid="credit-stop-billing-link"
          className="ml-1 font-medium text-[var(--accent-text)] underline underline-offset-2 hover:decoration-2"
        >
          去 {name} 儲值 <span aria-hidden="true">↗</span>
          <span className="sr-only">（另開新分頁）</span>
        </a>
      )}
    </p>
  );
}
