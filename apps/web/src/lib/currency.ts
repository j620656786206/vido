/**
 * USD display formatting for the subtitle-generation cost surfaces.
 *
 * Extracted by sub-4-3 from the two previously-duplicated copies in
 * GenerationBatchDialogV2 and GenerationWorkspaceV2 so the consent screens
 * (F14–F20) and the execution screen render every amount through ONE formatter.
 * Amounts render VERBATIM from the backend's estimated/spent values — the
 * 2026-08-11 §5-sexies ruling bans a "免費" rounding presentation.
 */
export function usd(v: number): string {
  return `$${v.toFixed(2)}`;
}
