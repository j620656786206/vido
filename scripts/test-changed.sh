#!/bin/bash
# Selective Test Runner for Vido
# Usage: ./scripts/test-changed.sh [base-branch]
#
# Intelligently runs tests based on changed files.
# - Changed test files: Run directly
# - Critical config changes: Run ALL tests
# - Component changes: Run related tests
# - Documentation only: Skip tests

set -e

BASE_BRANCH=${1:-main}
TEST_ENV=${TEST_ENV:-local}

echo "🎯 Selective Test Runner"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Base branch: $BASE_BRANCH"
echo "Environment: $TEST_ENV"
echo ""

# Detect changed files
CHANGED_FILES=$(git diff --name-only $BASE_BRANCH...HEAD 2>/dev/null || git diff --name-only HEAD~1)

if [ -z "$CHANGED_FILES" ]; then
  echo "✅ No files changed. Skipping tests."
  exit 0
fi

echo "Changed files:"
echo "$CHANGED_FILES" | sed 's/^/  - /'
echo ""

# Determine test strategy
RUN_ALL_TESTS=false
RUN_SMOKE_ONLY=false
DIRECT_TEST_FILES=""
RELATED_TEST_FILES=""

# Process each changed file
while IFS= read -r file; do
  case "$file" in
    # Changed test files: run them directly
    *.spec.ts|*.spec.js|*.test.ts|*.test.js)
      DIRECT_TEST_FILES="$DIRECT_TEST_FILES $file"
      ;;

    # Critical config changes: run ALL tests
    package.json|package-lock.json|playwright.config.ts|tsconfig*.json|.github/workflows/*)
      echo "⚠️  Critical file changed: $file"
      RUN_ALL_TESTS=true
      ;;

    # Frontend component changes
    apps/web/src/components/*|apps/web/src/routes/*)
      echo "🎨 Frontend file changed: $file"
      RELATED_TEST_FILES="$RELATED_TEST_FILES tests/e2e/"
      ;;

    # Backend API changes
    apps/api/*.go|apps/api/**/*.go)
      echo "🔌 Backend file changed: $file"
      # Run API tests
      RELATED_TEST_FILES="$RELATED_TEST_FILES tests/e2e/*.api.spec.ts"
      ;;

    # Documentation only: skip tests
    *.md|docs/*|README*)
      echo "📄 Documentation changed: $file (no tests needed)"
      ;;

    *)
      echo "❓ Other change: $file"
      RUN_SMOKE_ONLY=true
      ;;
  esac
done <<< "$CHANGED_FILES"

# Execute tests based on analysis
if [ "$RUN_ALL_TESTS" = true ]; then
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "🚨 Running FULL test suite (critical changes detected)"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  npm run test:e2e
  exit $?
fi

# Combine test files
ALL_TEST_FILES="$DIRECT_TEST_FILES $RELATED_TEST_FILES"
# Remove duplicates and empty entries
UNIQUE_TEST_FILES=$(echo "$ALL_TEST_FILES" | tr ' ' '\n' | sort -u | grep -v '^$' | tr '\n' ' ')

if [ -z "$UNIQUE_TEST_FILES" ] || [ "$RUN_SMOKE_ONLY" = true ]; then
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "🔍 Running smoke tests only"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  npm run test:smoke
  exit $?
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎯 Running selective tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Tests to run: $UNIQUE_TEST_FILES"
echo ""

npx playwright test $UNIQUE_TEST_FILES --project=chromium
