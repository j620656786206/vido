#!/usr/bin/env bash
set -euo pipefail

image="${1:-vido:local}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if ! command -v syft >/dev/null 2>&1; then
  echo "syft is required: install Anchore Syft, then rerun: $0 $image" >&2
  exit 2
fi

mkdir -p artifacts/sbom
syft "$image" -o spdx-json="artifacts/sbom/vido.spdx.json"
jq -r '.packages[] | select(.licenseDeclared|test("GPL|LGPL";"i")) | [.name,.versionInfo,.licenseDeclared] | @tsv' \
  artifacts/sbom/vido.spdx.json | sort -u > artifacts/sbom/copyleft-packages.tsv
if (cd apps/api && go list -deps ./cmd/api) | grep -Eq 'github.com/(longbridgeapp/opencc|liuzl/(da|cedar-go))'; then
  echo "GPL-related OpenCC Go dependency remains in production graph" >&2
  exit 1
fi
grep -q 'VIDO_OPENCC_CONFIG=/usr/share/opencc/s2twp.json' Dockerfile
grep -q 'OPENCC_VERSION=ver.1.4.2' Dockerfile
grep -q 's2twp.json' Dockerfile
docker run --rm --entrypoint /usr/local/bin/opencc "$image" --version | grep -q 'Version: 1.4.2'
jq -n '{spdxVersion:"SPDX-2.3",dataLicense:"CC0-1.0",SPDXID:"SPDXRef-DOCUMENT",name:"vido-opencc-supplement",packages:[{name:"OpenCC",SPDXID:"SPDXRef-Package-OpenCC",versionInfo:"1.4.2",downloadLocation:"https://github.com/BYVoid/OpenCC/tree/ver.1.4.2",licenseConcluded:"Apache-2.0",licenseDeclared:"Apache-2.0",filesAnalyzed:false,comment:"Official C++ helper and data/*.json dictionaries bundled in image"}]}' > artifacts/sbom/opencc-supplemental.spdx.json
echo "SBOM and OpenCC license checks passed: $image"
echo "注意：image 的 Alpine/FFmpeg runtime 仍含 GPL/LGPL 套件；詳見 artifacts/sbom/copyleft-packages.tsv"
