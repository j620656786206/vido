// Design ref: ux-design.pen Screen C4-D (6UCtX) · C4-M (2H4OM)
import { CheckCircle, XCircle } from 'lucide-react';
import { cn } from '../../lib/utils';

export interface ConnectionTestResultProps {
  success: boolean;
  message: string;
  version?: string;
  apiVersion?: string;
}

export function ConnectionTestResult({
  success,
  message,
  version,
  apiVersion,
}: ConnectionTestResultProps) {
  return (
    <div
      data-testid="connection-test-result"
      className={cn(
        'mt-4 flex items-start gap-3 rounded-lg p-4',
        success
          ? 'bg-[var(--success-tint)] text-[var(--success-text)]'
          : 'bg-[var(--error-tint)] text-[var(--error-text)]'
      )}
    >
      {success ? (
        <CheckCircle className="mt-0.5 h-5 w-5 flex-shrink-0" />
      ) : (
        <XCircle className="mt-0.5 h-5 w-5 flex-shrink-0" />
      )}
      <div>
        <p className="font-medium">{message}</p>
        {success && version && (
          <p className="mt-1 text-sm opacity-80">
            qBittorrent {version}
            {apiVersion && ` (API ${apiVersion})`}
          </p>
        )}
      </div>
    </div>
  );
}
