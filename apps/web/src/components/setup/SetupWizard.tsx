// Design ref: ux-design.pen Screen N1-D (dzgq9) · N2-D (CP7AX) · N3-D (TyjL0) · N4-D (D990CP) · N5-D (CWh3E) · N3-M (YyaqL)
import { useState, useCallback } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useQueryClient } from '@tanstack/react-query';
import { setupKeys } from '../../hooks/useSetupStatus';
import { setupService, type SetupConfig } from '../../services/setupService';
import { StepProgress } from './StepProgress';
import { WelcomeStep } from './WelcomeStep';
import { QBittorrentStep } from './QBittorrentStep';
import { MediaLibrarySetupStep } from './MediaLibrarySetupStep';
import { ApiKeysStep } from './ApiKeysStep';
import { CompleteStep } from './CompleteStep';

export interface StepProps {
  data: Partial<SetupConfig>;
  onUpdate: (updates: Partial<SetupConfig>) => void;
  onNext: () => void;
  onBack: () => void;
  onSkip?: () => void;
  isFirst: boolean;
  isLast: boolean;
  isSubmitting?: boolean;
}

interface WizardStep {
  id: string;
  title: string;
  component: React.ComponentType<StepProps>;
  optional?: boolean;
  /** Fields 跳過 throws away — skipping a step means "don't set this up". */
  clearOnSkip?: (keyof SetupConfig)[];
}

const WIZARD_STEPS: WizardStep[] = [
  { id: 'welcome', title: '歡迎', component: WelcomeStep },
  {
    id: 'qbittorrent',
    title: 'qBittorrent',
    component: QBittorrentStep,
    optional: true,
    clearOnSkip: ['qbtUrl', 'qbtUsername', 'qbtPassword'],
  },
  { id: 'media-folder', title: '媒體庫', component: MediaLibrarySetupStep },
  {
    id: 'api-keys',
    title: 'API 金鑰',
    component: ApiKeysStep,
    optional: true,
    clearOnSkip: ['tmdbApiKey', 'claudeApiKey'],
  },
  { id: 'complete', title: '完成', component: CompleteStep },
];

export function SetupWizard() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [currentStep, setCurrentStep] = useState(0);
  const [formData, setFormData] = useState<Partial<SetupConfig>>({
    language: 'zh-TW',
  });
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleUpdate = useCallback((updates: Partial<SetupConfig>) => {
    setFormData((prev) => ({ ...prev, ...updates }));
  }, []);

  const handleNext = useCallback(async () => {
    setError(null);
    const step = WIZARD_STEPS[currentStep];

    // Validate current step
    try {
      const stepData: Record<string, unknown> = {};
      if (step.id === 'welcome') stepData.language = formData.language;
      if (step.id === 'qbittorrent') stepData.qbtUrl = formData.qbtUrl || '';
      if (step.id === 'media-folder') stepData.libraries = formData.libraries;
      if (step.id === 'api-keys') {
        stepData.tmdbApiKey = formData.tmdbApiKey || '';
        // Sent so the server can refuse a key it cannot store (no ENCRYPTION_KEY)
        // here, on this step, instead of after 完成設定.
        stepData.claudeApiKey = formData.claudeApiKey || '';
      }

      await setupService.validateStep(step.id, stepData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Validation failed');
      return;
    }

    if (currentStep < WIZARD_STEPS.length - 1) {
      setCurrentStep((prev) => prev + 1);
    }
  }, [currentStep, formData]);

  const handleBack = useCallback(() => {
    setError(null);
    if (currentStep > 0) {
      setCurrentStep((prev) => prev - 1);
    }
  }, [currentStep]);

  const handleSkip = useCallback(() => {
    setError(null);
    // Skipping never validates, so whatever was half-typed on this step must not
    // ride along to 完成設定 and be reported 已設定.
    const cleared = Object.fromEntries(
      (WIZARD_STEPS[currentStep].clearOnSkip ?? []).map((field) => [field, undefined])
    ) as Partial<SetupConfig>;
    setFormData((prev) => ({ ...prev, ...cleared }));
    if (currentStep < WIZARD_STEPS.length - 1) {
      setCurrentStep((prev) => prev + 1);
    }
  }, [currentStep]);

  const handleFinish = useCallback(async () => {
    setError(null);
    setIsSubmitting(true);
    try {
      await setupService.completeSetup({
        language: formData.language || 'zh-TW',
        qbtUrl: formData.qbtUrl,
        qbtUsername: formData.qbtUsername,
        qbtPassword: formData.qbtPassword,
        libraries: formData.libraries as SetupConfig['libraries'],
        tmdbApiKey: formData.tmdbApiKey,
        claudeApiKey: formData.claudeApiKey,
      });

      // Invalidate setup status query so root route knows setup is done
      await queryClient.invalidateQueries({ queryKey: setupKeys.status() });

      // Navigate to dashboard
      navigate({ to: '/' });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to complete setup');
    } finally {
      setIsSubmitting(false);
    }
  }, [formData, navigate, queryClient]);

  const step = WIZARD_STEPS[currentStep];
  const StepComponent = step.component;

  return (
    // No shadow: the card sits on an otherwise empty page, it does not float
    // over anything (DESIGN.md §Shadow Vocabulary, 2026-09-14 — the login card
    // lost its shadow for the same reason in dsr-12).
    <div
      className="flex w-full max-w-lg flex-col gap-6 rounded-[var(--radius-xl)] border border-[var(--border-subtle)] bg-[var(--bg-primary)] p-8"
      data-testid="setup-wizard"
    >
      {/* The product's one wordmark — the same「vido · NAS 媒體庫」the login
          gate and the sidebar carry. */}
      <div className="flex items-end justify-center gap-2">
        <h1 className="text-xl font-bold leading-none text-[var(--accent-text)] sm:text-2xl">
          vido
        </h1>
        <p className="text-xs leading-none text-[var(--text-muted)]">NAS 媒體庫</p>
      </div>

      <div>
        <StepProgress steps={WIZARD_STEPS} currentStep={currentStep} />
        {/* The dots carry progress for sighted users; this carries it in words. */}
        <p className="sr-only">
          步驟 {currentStep + 1} / {WIZARD_STEPS.length}
        </p>
      </div>

      {error && (
        <div
          className="rounded-lg border border-[var(--error)]/30 bg-[var(--error)]/10 px-4 py-3 text-sm text-[var(--error-text)]"
          role="alert"
          data-testid="setup-error"
        >
          {error}
        </div>
      )}

      <StepComponent
        data={formData}
        onUpdate={handleUpdate}
        onNext={step.id === 'complete' ? handleFinish : handleNext}
        onBack={handleBack}
        onSkip={step.optional ? handleSkip : undefined}
        isFirst={currentStep === 0}
        isLast={currentStep === WIZARD_STEPS.length - 1}
        isSubmitting={isSubmitting}
      />
    </div>
  );
}
