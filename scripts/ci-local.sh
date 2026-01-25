#!/bin/bash
# Local CI Mirror for Vido
# Usage: ./scripts/ci-local.sh
#
# Mirrors CI pipeline locally for debugging failed CI runs.
# Runs: lint → tests → burn-in (3 iterations)

set -e

echo "🔍 Vido Local CI Pipeline"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "This script mirrors the CI pipeline locally."
echo ""

START_TIME=$(date +%s)

# Stage 1: Lint
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📝 Stage 1: Lint & Format Check"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo "Running ESLint..."
npm run lint || {
  echo "❌ Lint failed"
  exit 1
}
echo "✅ Lint passed"

echo ""
echo "Checking formatting..."
npm run format:check || {
  echo "❌ Format check failed"
  echo "Run 'npm run format' to fix formatting issues"
  exit 1
}
echo "✅ Format check passed"

# Stage 2: E2E Tests
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 Stage 2: E2E Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo "Running E2E tests (chromium only for speed)..."
npx playwright test --project=chromium || {
  echo "❌ E2E tests failed"
  echo "Check playwright-report/ for details"
  exit 1
}
echo "✅ E2E tests passed"

# Stage 3: Burn-in (reduced iterations)
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔥 Stage 3: Burn-In (3 iterations)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

for i in {1..3}; do
  echo "Burn-in iteration $i/3..."
  npx playwright test --project=chromium || {
    echo "❌ Burn-in failed on iteration $i"
    exit 1
  }
  echo "✅ Iteration $i passed"
done

# Summary
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ LOCAL CI PIPELINE PASSED"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Total duration: ${DURATION}s"
echo ""
echo "Your code is ready for CI!"
