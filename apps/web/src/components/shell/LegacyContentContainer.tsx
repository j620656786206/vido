// Implements: <utility — no .pen counterpart>
/**
 * Legacy content container (UX Redesign Phase 2 — UX2-1, ADR "Legacy fidelity").
 *
 * During the strangler cascade the v2 shell hosts both migrated and untouched
 * routes. Untouched routes render here, reproducing the legacy `<main>` exactly:
 * the legacy shell's main was an unconstrained `flex-1` and every route self-owns
 * its width (`mx-auto max-w-7xl px-4 …`), so this container imposes NO width or
 * padding of its own — adding any would change those self-centering routes. It is
 * the explicit default wrapper; a migrated route opts OUT via `staticData.shell:
 * 'v2'` (see AppShellV2), which is the single source of truth for "is this screen
 * migrated yet".
 */
interface LegacyContentContainerProps {
  children: React.ReactNode;
}

export function LegacyContentContainer({ children }: LegacyContentContainerProps) {
  return (
    <div data-testid="legacy-content-container" className="min-w-0 flex-1">
      {children}
    </div>
  );
}
