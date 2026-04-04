interface StepProgressProps {
  steps: { id: string; title: string }[];
  currentStep: number;
}

export function StepProgress({ steps, currentStep }: StepProgressProps) {
  return (
    <div className="mb-8 flex items-center justify-center gap-2" data-testid="step-progress">
      {steps.map((step, index) => (
        <div key={step.id} className="flex items-center gap-2">
          <div
            className={`h-2.5 w-2.5 rounded-full transition-colors ${
              index <= currentStep ? 'bg-[var(--accent-primary)]' : 'bg-[var(--bg-tertiary)]'
            }`}
            aria-label={`${step.title}${index === currentStep ? ' (目前)' : ''}`}
            data-testid={`step-dot-${step.id}`}
          />
          {index < steps.length - 1 && (
            <div
              className={`h-0.5 w-6 transition-colors ${
                index < currentStep ? 'bg-[var(--accent-primary)]' : 'bg-[var(--bg-tertiary)]'
              }`}
            />
          )}
        </div>
      ))}
    </div>
  );
}
