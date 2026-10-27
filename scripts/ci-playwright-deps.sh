#!/usr/bin/env bash
# Install Playwright's browsers/system packages on a CI runner without letting
# one slow apt mirror cancel the job (disc-2026-10-ci-playwright-install-deps-hang).
#
#   scripts/ci-playwright-deps.sh install --with-deps chromium webkit
#   scripts/ci-playwright-deps.sh install-deps chromium webkit
#
# Seen 2026-10-27: on one runner the azure mirror served ~70 KB/s while the
# other three shards fetched the same 119 MB in 30 s. Each attempt gets 6
# minutes; a second attempt goes to Canonical's mirror instead.
set -uo pipefail

for attempt in 1 2; do
  timeout 6m npx playwright "$@" && exit 0
  echo "::warning::playwright $* attempt $attempt did not finish in 6 minutes"
  # timeout stops npx, not the apt-get it started through sudo — that one
  # still holds the dpkg lock and the next attempt could not even start.
  sudo pkill -x apt-get || true
  sleep 3
  sudo dpkg --configure -a || true
  echo "http://archive.ubuntu.com/ubuntu/" | sudo tee /etc/apt/apt-mirrors.txt > /dev/null
done
exit 1
