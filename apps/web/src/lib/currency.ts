/**
 * USD arithmetic and display for the subtitle-generation cost surfaces.
 *
 * Extracted by sub-4-3 from the two previously-duplicated copies in
 * GenerationBatchDialogV2 and GenerationWorkspaceV2 so the consent screens
 * (F14–F20) and the execution screen render every amount through ONE
 * formatter. Amounts render VERBATIM from the backend's estimated/spent values
 * — the 2026-08-11 §5-sexies ruling bans a "免費" rounding presentation.
 *
 * tech-money-decimal-arithmetic (Alexyu, 2026-09-04): money is NEVER added,
 * subtracted or compared with the `+ - <` operators on this project's screens.
 * A JS number is an IEEE-754 double, and `0.1 + 0.2 === 0.30000000000000004`
 * is not a curiosity — it is how a confirm screen ends up showing
 * 「$0.01 + $0.01 = $0.01」, or how a running total drifts away from the figure
 * the backend charged. Every operation here goes through decimal.js, which
 * holds the decimal value the provider actually bills.
 *
 * The backend does the same with shopspring/decimal, so both sides perform
 * IDENTICAL arithmetic — which is the property that keeps the quote on screen
 * and the amount on the invoice the same number, rather than merely close.
 *
 * ⚠️ The wire between them is still JSON, and `JSON.parse` yields doubles. That
 * is the one remaining lossy hop; see backlog-money-string-on-the-wire.
 */
import DecimalJs from 'decimal.js';

/**
 * CR L7: a CLONE, not `Decimal.set(...)`. Configuring the imported constructor
 * mutates decimal.js's process-wide default, so any future unrelated consumer
 * in this bundle would silently inherit this module's precision and rounding.
 * Money's settings belong to money.
 */
const Decimal = DecimalJs.clone({ precision: 28, rounding: DecimalJs.ROUND_HALF_UP });

/** The clone's instance type. Named apart from the `Decimal` const because a
 *  value and a type sharing a name is legal TS but trips eslint's no-redeclare. */
type DecimalValue = InstanceType<typeof Decimal>;

/** Parse a wire/`number` amount into an exact decimal. */
function dec(v: number | string | DecimalValue): DecimalValue {
  // Number → Decimal goes via the number's SHORTEST round-tripping decimal
  // form, which is what a `0.53` on the wire was meant to be — not the
  // 0.5300000000000000266453525910037569701671600341796875 the double holds.
  return v instanceof Decimal ? v : new Decimal(v);
}

/** Display form: `$4.50`. Rounds half-up to cents, the rule invoices state. */
export function usd(v: number | string | DecimalValue): string {
  return `$${dec(v).toFixed(2, Decimal.ROUND_HALF_UP)}`;
}

/**
 * Exact sum of USD amounts.
 *
 * Use this instead of `reduce((a, b) => a + b, 0)` anywhere a total is shown
 * next to the parts it is made of: a float sum of the parts can round to a
 * different cent than the parts themselves display, and a screen whose
 * breakdown does not add up to its own total is the failure this module
 * exists to prevent.
 */
export function sumUsd(values: Array<number | string | DecimalValue>): number {
  return values.reduce<DecimalValue>((acc, v) => acc.plus(dec(v)), new Decimal(0)).toNumber();
}

/** Exact `a + b`. */
export function addUsd(
  a: number | string | DecimalValue,
  b: number | string | DecimalValue
): number {
  return dec(a).plus(dec(b)).toNumber();
}

/** Exact `a - b`. */
export function subUsd(
  a: number | string | DecimalValue,
  b: number | string | DecimalValue
): number {
  return dec(a).minus(dec(b)).toNumber();
}

/** Exact `a > b`, for a soft-ceiling verdict that must not hinge on 1e-17. */
export function gtUsd(
  a: number | string | DecimalValue,
  b: number | string | DecimalValue
): boolean {
  return dec(a).greaterThan(dec(b));
}

/** Exact `a < b`. */
export function ltUsd(
  a: number | string | DecimalValue,
  b: number | string | DecimalValue
): boolean {
  return dec(a).lessThan(dec(b));
}

/**
 * `a / b` as a whole percent, rounded half-up. Returns undefined when `b` is
 * zero — "cheaper by ∞%" is not a thing to put on a screen.
 */
export function percentOfUsd(
  a: number | string | DecimalValue,
  b: number | string | DecimalValue
): number | undefined {
  const denom = dec(b);
  if (denom.isZero()) return undefined;
  return dec(a)
    .abs()
    .dividedBy(denom)
    .times(100)
    .toDecimalPlaces(0, Decimal.ROUND_HALF_UP)
    .toNumber();
}

/**
 * Fixed-decimal string at `places`, rounded half-up on the DECIMAL value.
 * `(1.005).toFixed(2)` is "1.00" because it rounds the double; this is "1.01",
 * which is what the amount was written as and what an invoice would say.
 */
export function fixedUsd(v: number | string | DecimalValue, places: number): string {
  return dec(v).toFixed(places, Decimal.ROUND_HALF_UP);
}

/** Exact `a / b` — for magnitude folding (dollars → thousands), not for money splits. */
export function divideUsd(
  a: number | string | DecimalValue,
  b: number | string | DecimalValue
): number {
  return dec(a).dividedBy(dec(b)).toNumber();
}

/** Round to whole cents, half-up — the presentation rule, applied exactly once. */
export function roundUsd(v: number | string | DecimalValue): number {
  return dec(v).toDecimalPlaces(2, Decimal.ROUND_HALF_UP).toNumber();
}
