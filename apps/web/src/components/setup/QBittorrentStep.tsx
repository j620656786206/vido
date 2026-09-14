// Design ref: ux-design.pen Screen N2-D (CP7AX)
import type { StepProps } from './SetupWizard';
import { StepNav } from './StepNav';
import { WizardTextField } from './WizardTextField';

export function QBittorrentStep({ data, onUpdate, onNext, onBack, onSkip }: StepProps) {
  return (
    <div className="flex flex-col gap-4" data-testid="qbittorrent-step">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">qBittorrent 連線</h2>
      <p className="text-sm text-[var(--text-secondary)]">
        連接 qBittorrent 以監控下載進度。如果你尚未安裝，可以跳過此步驟。
      </p>

      {/* 主機位址 — the same word the settings page (QBittorrentForm) uses for
          the same field. The wizard used to say「WebUI 網址」and the design
          「伺服器位址」: three names for one setting. */}
      <WizardTextField
        id="qbt-url"
        label="主機位址"
        value={data.qbtUrl || ''}
        onChange={(qbtUrl) => onUpdate({ qbtUrl })}
        placeholder="http://localhost:8080"
        testId="qbt-url-input"
      />
      <WizardTextField
        id="qbt-username"
        label="使用者名稱"
        value={data.qbtUsername || ''}
        onChange={(qbtUsername) => onUpdate({ qbtUsername })}
        placeholder="admin"
        testId="qbt-username-input"
      />
      <WizardTextField
        id="qbt-password"
        label="密碼"
        type="password"
        value={data.qbtPassword || ''}
        onChange={(qbtPassword) => onUpdate({ qbtPassword })}
        testId="qbt-password-input"
      />

      <StepNav onNext={onNext} onBack={onBack} onSkip={onSkip} />
    </div>
  );
}
