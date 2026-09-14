// Design ref: ux-design.pen Screen N2-D (CP7AX) · N4-D (D990CP)
// Label · input · optional hint, 8px apart. The value is set in the mono face
// because everything typed into these fields is machine text — a URL, a user
// name, an API key — and the design sets all five of them in JetBrains Mono.
interface WizardTextFieldProps {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  testId: string;
  type?: 'text' | 'password';
  placeholder?: string;
  hint?: string;
}

export function WizardTextField({
  id,
  label,
  value,
  onChange,
  testId,
  type = 'text',
  placeholder,
  hint,
}: WizardTextFieldProps) {
  // Machine text, so no autocomplete / auto-capitalise / spell-check. A text
  // field followed by a password field otherwise reads to Chrome as a login
  // form: it would offer to save the Claude key as this site's password and
  // later autofill it into the real login gate (the settings page's key fields
  // opt out the same way).
  const hintId = hint ? `${id}-hint` : undefined;
  return (
    <div className="flex flex-col gap-2">
      <label htmlFor={id} className="text-sm font-medium text-[var(--text-secondary)]">
        {label}
      </label>
      <input
        id={id}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoComplete="off"
        autoCapitalize="off"
        spellCheck={false}
        aria-describedby={hintId}
        className="h-11 w-full rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 font-mono text-sm text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:border-[var(--focus-ring)] focus:outline-none focus:ring-1 focus:ring-[var(--focus-ring)]"
        data-testid={testId}
      />
      {hint && (
        <p id={hintId} className="text-xs text-[var(--text-muted)]">
          {hint}
        </p>
      )}
    </div>
  );
}
