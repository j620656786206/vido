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
}

const WIZARD_STEPS: WizardStep[] = [
  { id: 'welcome', title: '歡迎', component: WelcomeStep },
  { id: 'qbittorrent', title: 'qBittorrent', component: QBittorrentStep, optional: true },
  { id: 'media-folder', title: '媒體庫', component: MediaLibrarySetupStep },
  { id: 'api-keys', title: 'API 金鑰', component: ApiKeysStep, optional: true },
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
      if (step.id === 'api-keys') stepData.tmdbApiKey = formData.tmdbApiKey || '';

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
        aiProvider: formData.aiProvider,
        aiApiKey: formData.aiApiKey,
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
    <div
      className="w-full max-w-lg rounded-2xl border border-[var(--border-subtle)]/50 bg-[var(--bg-primary)] p-8 shadow-2xl"
      data-testid="setup-wizard"
    >
      <div className="mb-8 text-center">
        <h1 className="text-2xl font-bold text-[var(--text-primary)]">Vido 設定精靈</h1>
        <p className="mt-1 text-sm text-[var(--text-secondary)]">
          步驟 {currentStep + 1} / {WIZARD_STEPS.length}
        </p>
      </div>

      <StepProgress steps={WIZARD_STEPS} currentStep={currentStep} />

      {error && (
        <div
          className="mb-4 rounded-lg border border-red-500/30 bg-[var(--error)]/10 px-4 py-3 text-sm text-[var(--error)]"
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
