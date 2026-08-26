#!/usr/bin/env python3
"""Export screenshots from ux-design.pen organized by user flow.

Usage:
    python3 scripts/export-pen-screenshots.py

Requires the Pencil app to be running — installed as `Pen.app` since 1.2.2, and as
`Pencil.app` before that; both paths are probed (see MCP_BIN_CANDIDATES). Spawns its
MCP server in stdio mode (1.1.61 removed the old `--http` transport), captures each
design screen, and saves PNGs to _bmad-output/screenshots/.

Note: Pen 1.2.5 removed the `get_screenshot` MCP tool (surface is now browser /
execute / get_app_state / get_guidelines). Screens are exported via `execute` +
`Export(...)` at scale 1 into a temp dir, then renamed into flow folders and
downscaled with `sips -Z 400` to keep the historical 400px-long-edge thumbnail
convention. A text-dense spec screen (flow-j-specs) is unreadable at that size —
see backlog-pen-spec-screen-readable-export.

Layout convention (2026-06-05 A–J merged-block rework):
  Screens are named with flow codes `{Flow}{seq}-{D|M}` (desktop/mobile) on the
  canvas. Screenshots mirror that: one folder per flow, filename == lowercased code.
  See .claude/memory/project_pen_flow_layout_convention.md.
"""

import json
import os
import shutil
import subprocess
import sys
import tempfile

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PEN_FILE = os.path.join(PROJECT_ROOT, "ux-design.pen")
OUT_DIR = os.path.join(PROJECT_ROOT, "_bmad-output", "screenshots")
# The app shipped as Pencil.app and was renamed to Pen.app (1.2.2, found at story
# sub-1-7a). Probe both rather than hardcoding either — a hardcoded path turns the
# next rename into "the export script is broken" instead of "the app moved".
MCP_BIN_CANDIDATES = (
    "/Applications/Pen.app/Contents/Resources/app.asar.unpacked/out/mcp-server-darwin-arm64",
    "/Applications/Pencil.app/Contents/Resources/app.asar.unpacked/out/mcp-server-darwin-arm64",
)


def resolve_mcp_bin():
    """First existing candidate, or None."""
    for candidate in MCP_BIN_CANDIDATES:
        if os.path.exists(candidate):
            return candidate
    return None


MCP_BIN = resolve_mcp_bin()

# Screen node ID -> (flow_folder, filename). Filename == canvas frame code (lowercased).
SCREENS = {
    # Flow A — 瀏覽主流程 (browse: empty / loading / grid / list / sort / filter)
    "4VILE": ("flow-a-browse", "a1-d"),
    "OYqNo": ("flow-a-browse", "a1-m"),
    "IpZhv": ("flow-a-browse", "a2-d"),
    "RxdY5": ("flow-a-browse", "a2-m"),
    "KNI8F": ("flow-a-browse", "a3-d"),
    "GOL63": ("flow-a-browse", "a3-m"),
    "LZ8Ds": ("flow-a-browse", "a4-d"),
    "3aSCw": ("flow-a-browse", "a5-m"),
    "oypj1": ("flow-a-browse", "a6-m"),
    # Flow B — 詳情與互動 (hover / context menus / detail panels / fallbacks / tech badges / image-load spec)
    "Qm662": ("flow-b-detail-interaction", "b1-d"),
    "auArc": ("flow-b-detail-interaction", "b2-d"),
    "1UHzI": ("flow-b-detail-interaction", "b2-m"),
    "RgSxQ": ("flow-b-detail-interaction", "b3-d"),
    "kcn1v": ("flow-b-detail-interaction", "b3-m"),
    "407vK": ("flow-b-detail-interaction", "b4-d"),
    "7mdTJ": ("flow-b-detail-interaction", "b5-d"),
    "APfjC": ("flow-b-detail-interaction", "b5-m"),
    "2ltBl": ("flow-b-detail-interaction", "b6-d"),
    "2m1Pv": ("flow-b-detail-interaction", "b6-m"),
    "wQOkg": ("flow-b-detail-interaction", "b7-d"),
    "7UnDy": ("flow-b-detail-interaction", "b7-m"),
    "vlL6O": ("flow-b-detail-interaction", "b8-d"),
    "6OR3z": ("flow-b-detail-interaction", "b8-m"),
    # B9 = disc-flaky-visual-media-detail-panel case (B) image-load fallback spec
    "Tn4Gz": ("flow-b-detail-interaction", "b9-d"),
    "jH6rM": ("flow-b-detail-interaction", "b9-m"),
    # Flow C — 搜尋 / 篩選 / 設定 (search-filter / batch ops / settings / backup)
    "rsAxf": ("flow-c-search-settings", "c1-d"),
    "dcf67": ("flow-c-search-settings", "c2-d"),
    "0KOE7": ("flow-c-search-settings", "c2-m"),
    "7fE0b": ("flow-c-search-settings", "c3-d"),
    "IfrPQ": ("flow-c-search-settings", "c3-m"),
    "6UCtX": ("flow-c-search-settings", "c4-d"),
    "2H4OM": ("flow-c-search-settings", "c4-m"),
    "uhAKd": ("flow-c-search-settings", "c5-d"),
    # Flow D — 下載管理 (downloads)
    "rWvuG": ("flow-d-downloads", "d1-d"),
    "cZd7j": ("flow-d-downloads", "d1-m"),
    "3ULXd": ("flow-d-downloads", "d2-d"),
    "tqHK9": ("flow-d-downloads", "d3-m"),
    # Flow E — 媒體庫掃描 (scanner settings / progress / complete toast / filtered-unmatched)
    "KvZSc": ("flow-e-scanner", "e1-d"),
    "uABWl": ("flow-e-scanner", "e1-m"),
    "wyuhF": ("flow-e-scanner", "e2-d"),
    "yezIo": ("flow-e-scanner", "e2-m"),
    "szzaW": ("flow-e-scanner", "e3-d"),
    "ZjoEI": ("flow-e-scanner", "e3-m"),
    "QTqcC": ("flow-e-scanner", "e4-d"),
    "n7jVF": ("flow-e-scanner", "e4-m"),
    # Media library edit modal (9R-10b) — auto-generation toggle, free-only copy
    "hUVYm": ("flow-e-scanner", "e5-d"),
    "P0P82x": ("flow-e-scanner", "e5-m"),
    # Flow F — 字幕搜尋 / 批次 (subtitle search dialog / preview-download / batch progress)
    "cOrOR": ("flow-f-subtitle", "f1-d"),
    "GZ294": ("flow-f-subtitle", "f1-m"),
    "wy5Nx": ("flow-f-subtitle", "f2-d"),
    "ogQ6Y": ("flow-f-subtitle", "f2-m"),
    "NXijD": ("flow-f-subtitle", "f3-d"),
    "fUtqO": ("flow-f-subtitle", "f3-m"),
    # Flow G — AI 字幕增強 (correction / transcription progress / translation confirm)
    "TIIRl": ("flow-g-ai-subtitle", "g1-d"),
    "mgRJA": ("flow-g-ai-subtitle", "g1-m"),
    "kzhNP": ("flow-g-ai-subtitle", "g2-d"),
    "yNAHK": ("flow-g-ai-subtitle", "g2-m"),
    "22bcv": ("flow-g-ai-subtitle", "g3-d"),
    "8Wsez": ("flow-g-ai-subtitle", "g3-m"),
    # Flow H — 首頁 TV Wall (homepage / loading skeleton / block CRUD modal / exploreblock spec)
    "sAaCR": ("flow-h-homepage", "h1-d"),
    "g5LFD": ("flow-h-homepage", "h2-m"),
    "Paqlk": ("flow-h-homepage", "h3"),
    "g6p38": ("flow-h-homepage", "h4-d"),
    "Y5XvRv": ("flow-h-homepage", "h5-d"),
    # Flow I — 進階搜尋 / 篩選 (filter chips / suggestions dropdown / save preset / filter sheet)
    "NWxok": ("flow-i-advanced-search", "i1-d"),
    "TMaw5": ("flow-i-advanced-search", "i2"),
    "i74p2": ("flow-i-advanced-search", "i3"),
    "pjKVZ": ("flow-i-advanced-search", "i4-m"),
    # Desktop filter rail redesign (v2 Design System) — replaces mobile bottom-sheet misuse on lg+
    "vpDLh": ("flow-i-advanced-search", "i5-d"),  # rail persistent (hero) — re-merged onto feat/ux3-2-1 (#89 .pen frames reconstructed post main-merge)
    "VwTvy": ("flow-i-advanced-search", "i6-d"),  # rail collapsed + filtered no-results — re-merged
    "SgncH": ("flow-i-advanced-search", "i7-d"),  # rail states spec (genre loading / load-failed) — re-merged
    # Flow J — 設計決策 spec (PosterCard info-density & polish)
    "XlFIq": ("flow-j-specs", "j1-d"),
    # Subtitle-status badge spec (story sub-1-7a) — the 5 new subtitle_status values'
    # badge/icon treatment, the transient-vs-terminal ruling, and the copy resolutions.
    "ZpQaw": ("flow-j-specs", "j2-d"),
    # Episode-row subtitle entry spec (9R-UX) — action/status decoupling ruling,
    # has_local_file gate, 10-status dialog behavior table.
    "Z54xAd": ("flow-j-specs", "j3-d"),
    # On-add auto-generation free/paid boundary spec (9R-10b) — 2026-08-07/08-19 rulings,
    # two-tier table, control states, LibraryCard footer, copy rationale.
    "sPzZT": ("flow-j-specs", "j4-d"),
    # Pipeline-disabled honest-state spec (9R-10b M4) — VIDO_SUBTITLE_PIPELINE_MODE=legacy:
    # 4-way decision matrix, disabled Modal field + notice bar, LibraryCard footer tri-state.
    "alrIw": ("flow-j-specs", "j5-d"),
    # .nfo localization confirm/result spec (9R-13b) — movie additive (no overwrite)
    # vs TV single-slot backup-and-replace, per-episode opt-in, AI-quota notice.
    "zMYsL": ("flow-j-specs", "j6-d"),
    # Settings form-card width ruling (max-w-2xl -> max-w-3xl) — real-width math with
    # two stacked sidebars, why C4-D's fill_container ratio is a stale assumption.
    "JBKis": ("flow-j-specs", "j7-d"),
    # Design system reference docs (top of canvas, no flow code)
    "8SSzc": ("design-system", "design-system-reference"),
    "sJzat": ("design-system", "component-library"),
    # UX Redesign Phase 1b — Design Language v2 + Navigation Shell v2
    "V2Kez": ("design-system", "design-language-v2"),
    "CLo58": ("design-system", "navigation-shell-v2"),
    # UX Redesign Phase 2 — A′ Browse v2 pilot (sidebar shell · integrated toolbar+chips · four states)
    "vZpT8": ("flow-a-browse-v2", "a1p-d"),
    "EsoIv": ("flow-a-browse-v2", "a2p-d"),
    "LcHBs": ("flow-a-browse-v2", "a3p-d"),
    "b1H71g": ("flow-a-browse-v2", "a4p-d"),
    "R3FqJc": ("flow-a-browse-v2", "a7p-d"),
    "dVGIa": ("flow-a-browse-v2", "a8p-d"),
    "BfGVZ": ("flow-a-browse-v2", "a1p-m"),
    "qBWQC": ("flow-a-browse-v2", "a2p-m"),
    "h1v1U6": ("flow-a-browse-v2", "a3p-m"),
    "Bz0YN": ("flow-a-browse-v2", "a6p-m"),
    # UX Redesign Phase 2 — B′ Detail v2 pilot (full-page backdrop hero · four states · Epic 12 fail-soft)
    "uRGu2": ("flow-b-detail-v2", "b3p-d"),
    "N2fmG6": ("flow-b-detail-v2", "b4p-d"),
    "Z42zy": ("flow-b-detail-v2", "b6p-d"),
    "Tqy3E": ("flow-b-detail-v2", "b7p-d"),
    "UH0sk": ("flow-b-detail-v2", "b8p-d"),
    "SzNRb": ("flow-b-detail-v2", "b3p-m"),
    # flow-h-homepage-v2 — Phase-3 ux3-1-1 (Home v2 redesign: own-content above Hero+Explore, D3)
    "yixu1": ("flow-h-homepage-v2", "h1-d"),
    "uCfjb": ("flow-h-homepage-v2", "h2-m"),
    "nnGs6": ("flow-h-homepage-v2", "h4-d"),
    "Z7OJB": ("flow-h-homepage-v2", "h5-d"),
    "xCQA7": ("flow-h-homepage-v2", "h6-d"),
    # flow-k-activity-v2 — Phase-3 ux3-2-1 (Activity hub v2: net-new D4-1 destination,
    # explain-why rows aggregating scan/subtitle/AI/parse + downloads-summary; four states)
    "kMeWS": ("flow-k-activity-v2", "a1-d"),
    "QIwY1": ("flow-k-activity-v2", "a2-m"),
    "suCiI": ("flow-k-activity-v2", "a4-d"),
    "DZnSv": ("flow-k-activity-v2", "a5-d"),
    "M6ra92": ("flow-k-activity-v2", "a6-d"),
    # flow-i-discover-v2 — Phase-3 ux3-3-1 (Discover v2: active power-filter tool; Epic 11
    # chips/presets/instant-search → v2; D3 no-dashboard boundary; reserves Epic 13 Requests;
    # 地區/串流平台 reserved-disabled per Rule-24; four states)
    "fxCVk": ("flow-i-discover-v2", "i1-d"),
    "hi6WD": ("flow-i-discover-v2", "i2-m"),
    "m0Zew": ("flow-i-discover-v2", "i3-d"),
    "m4fY7c": ("flow-i-discover-v2", "i4-d"),
    "kzzjc": ("flow-i-discover-v2", "i4-m"),
    "nLrzc": ("flow-i-discover-v2", "i5-d"),
    "YYEBd": ("flow-i-discover-v2", "i6-d"),
    "S3qke": ("flow-i-discover-v2", "i7-d"),
    "KdnVw": ("flow-i-discover-v2", "i8-d"),
    # flow-d-downloads-v2 — Downloads deep-operation page (design-ahead spec: card actions,
    # batch select, pagination; six backend filter values; qBittorrent fail-soft; NZBGet inert
    # placeholder). Replaces the deprecated flow-d-downloads (d1-d/d1-m/d2-d/d3-m).
    "cK1KF": ("flow-d-downloads-v2", "d1-d-v2"),
    "tx6U1": ("flow-d-downloads-v2", "d2-d-v2"),
    "lCFq2": ("flow-d-downloads-v2", "d3-d-v2"),
    "T95wy": ("flow-d-downloads-v2", "d4-d-v2"),
    "dVPuY": ("flow-d-downloads-v2", "d5-d-v2"),
    "UNVRU": ("flow-d-downloads-v2", "d6-d-v2"),
    "uMDjw": ("flow-d-downloads-v2", "d1-m-v2"),
    # v2.1 rework (Sally review): desktop List default + Table view + list toolbar;
    # mobile sheet-first actions + long-name split + detail sheet.
    "w3ipb": ("flow-d-downloads-v2", "d7-d-v2"),
    "jDgxJ": ("flow-d-downloads-v2", "d8-m-v2"),
    "DrYXb": ("flow-d-downloads-v2", "d9-m-v2"),
    # mobile sort sheet — the 排序 bottom-sheet the D1-M top-bar sort button opens
    "JxMWL": ("flow-d-downloads-v2", "d10-m-v2"),
    # flow-l-requests-v2 — Epic 13 story 13-0 Request System (Overseerr/Jellyseerr replacement:
    # one-click request from Discover/detail, season/episode granularity tree, status tracking
    # hosted inside Discover 想要清單 — NOT a new sidebar destination; five statuses only)
    "K7fiy": ("flow-l-requests-v2", "l1-d-v2"),
    "VH3Tq": ("flow-l-requests-v2", "l2-d-v2"),
    "He04g": ("flow-l-requests-v2", "l3-d-v2"),
    "n7isVa": ("flow-l-requests-v2", "l4-m-v2"),
    "ER39x": ("flow-l-requests-v2", "l5-d-v2"),
    "x4CNb": ("flow-l-requests-v2", "l6-d-v2"),
    "oopme": ("flow-l-requests-v2", "l7-d-v2"),
    "G0xib": ("flow-l-requests-v2", "l8-d-v2"),
    # flow-f-subtitle-v2 — story 9R-UX (generation-centric subtitle management, ADR Route C:
    # 生成字幕 primary, fetch dormant, glossary keystone; old flow-f f1-f3 + flow-g g1-g3
    # superseded in place). F11 = exploratory full-page variant (not story spec — pending ruling).
    "r1EY9": ("flow-f-subtitle-v2", "f1-d-v2"),
    "JkdfH": ("flow-f-subtitle-v2", "f1-m-v2"),
    "S9Rbrq": ("flow-f-subtitle-v2", "f2-d-v2"),
    "JbXai": ("flow-f-subtitle-v2", "f3-d-v2"),
    "k8sJl4": ("flow-f-subtitle-v2", "f3-m-v2"),
    "U8rRtv": ("flow-f-subtitle-v2", "f4-d-v2"),
    "f6ZxY": ("flow-f-subtitle-v2", "f5-d-v2"),
    "dlfMR": ("flow-f-subtitle-v2", "f6-d-v2"),
    "buepS": ("flow-f-subtitle-v2", "f6-m-v2"),
    "A85GFD": ("flow-f-subtitle-v2", "f7-d-v2"),
    "i9Nun1": ("flow-f-subtitle-v2", "f8-d-v2"),
    "H717g": ("flow-f-subtitle-v2", "f8-m-v2"),
    "JMqPg": ("flow-f-subtitle-v2", "f9-d-v2"),
    "olDlj": ("flow-f-subtitle-v2", "f10-d-v2"),
    "l8FsB": ("flow-f-subtitle-v2", "f11-d-v2"),
    # ux3-ai-1 — generation workspace (F11 validated spec: running / mobile / budget-ceiling / states)
    "PXB0z": ("flow-f-subtitle-v2", "f11-m-v2"),
    "iH98f": ("flow-f-subtitle-v2", "f12-d-v2"),
    "F7ohe": ("flow-f-subtitle-v2", "f13-d-v2"),
    # prompt-cost-consent — 產生字幕 cost-consent flow (F14 analyze / F15 candidate list
    # desktop+mobile / F16 cost confirm / F17 scan-complete toast with subtitle entry)
    "nBT3M": ("flow-f-subtitle-v2", "f14-d-v2"),
    "pwMzT": ("flow-f-subtitle-v2", "f15-d-v2"),
    "fdu4y": ("flow-f-subtitle-v2", "f15-m-v2"),
    "gmOt6": ("flow-f-subtitle-v2", "f16-d-v2"),
    "I3Wb0p": ("flow-f-subtitle-v2", "f17-d-v2"),
    # prompt-cost-consent round 2 — over-budget states + empty state
    "zBik1": ("flow-f-subtitle-v2", "f18-d-v2"),
    "KThbY": ("flow-f-subtitle-v2", "f19-d-v2"),
    "D7MOm": ("flow-f-subtitle-v2", "f20-d-v2"),
    # flow-h-homepage-v3 — home identity rework (home-v3-identity-brief.md: 讀數帶 4 格 +
    # 自家片庫靜態 hero + TMDb 尾巴濾已擁有; H7 = TMDb 降級態; H8 = 讀數帶金額顯示規則 spec).
    # Supersedes flow-h-homepage-v2 (v2 frames renamed 〔已淘汰→v3〕, deleted after v3 ships).
    "k2Otv": ("flow-h-homepage-v3", "h1-d-v3"),
    "EoCQ4": ("flow-h-homepage-v3", "h7-d-v3"),
    "uGCAU": ("flow-h-homepage-v3", "h2-m-v3"),
    "iWUSV": ("flow-h-homepage-v3", "h8-spec-v3"),
}


def start_mcp_server():
    # Pencil 1.1.61 removed the `--http`/`--http-port` flags; the MCP server now
    # only speaks newline-delimited JSON-RPC over stdio, connecting to the running
    # Pencil.app as a named agent. Spawn it in stdio mode and talk over the pipes.
    proc = subprocess.Popen(
        [MCP_BIN, "--app", "desktop", "--agent", "screenshot-export"],
        stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
        text=True, bufsize=1,
    )
    return proc


def mcp_send(proc, req_id, method, params):
    msg = {"jsonrpc": "2.0", "method": method, "params": params}
    if req_id is not None:
        msg["id"] = req_id
    proc.stdin.write(json.dumps(msg) + "\n")
    proc.stdin.flush()


def mcp_call(proc, req_id, method, params, max_lines=500):
    """Send a JSON-RPC request over stdio and return the matching response.

    Reads newline-delimited JSON from stdout, skipping notifications/log lines
    until the response whose `id` matches req_id is found.
    """
    mcp_send(proc, req_id, method, params)
    for _ in range(max_lines):
        line = proc.stdout.readline()
        if not line:
            return None
        line = line.strip()
        if not line.startswith("{"):
            continue
        try:
            msg = json.loads(line)
        except json.JSONDecodeError:
            continue
        if msg.get("id") == req_id:
            return msg
    return None


def main():
    if MCP_BIN is None:
        print("ERROR: Pen.app / Pencil.app MCP server not found. Looked in:")
        for candidate in MCP_BIN_CANDIDATES:
            print(f"  - {candidate}")
        sys.exit(1)

    print("Starting Pencil MCP server (stdio)...")
    proc = start_mcp_server()

    try:
        # Initialize
        resp = mcp_call(proc, 1, "initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "screenshot-export", "version": "1.0"},
        })
        if not resp:
            print("ERROR: Failed to connect to Pencil MCP server")
            sys.exit(1)
        print("Connected")

        # Send initialized notification
        mcp_send(proc, None, "notifications/initialized", {})

        # Create output directories
        for flow_dir, _ in SCREENS.values():
            os.makedirs(os.path.join(OUT_DIR, flow_dir), exist_ok=True)

        # Export screenshots via execute+Export in chunks (get_screenshot is gone
        # since Pen 1.2.5). Export writes <nodeId>.png per node into tmpdir.
        tmpdir = tempfile.mkdtemp(prefix="pen-export-")
        failed_chunks = 0
        node_ids = list(SCREENS.keys())
        chunk_size = 20
        for ci in range(0, len(node_ids), chunk_size):
            chunk = node_ids[ci:ci + chunk_size]
            js = f'Export({json.dumps(chunk)}, "png", {json.dumps(tmpdir)}, {{scale: 1}})'
            resp = mcp_call(proc, ci + 10, "tools/call", {
                "name": "execute",
                "arguments": {"filePath": PEN_FILE, "input": js},
            })
            if not resp or resp.get("error"):
                failed_chunks += 1
                print(f"  FAIL: export chunk {ci // chunk_size + 1} - {resp.get('error') if resp else 'no response'}")

        saved = 0
        for node_id, (flow_dir, filename) in SCREENS.items():
            src = os.path.join(tmpdir, f"{node_id}.png")
            if not os.path.exists(src):
                print(f"  SKIP: {filename} - not exported")
                continue
            dst = os.path.join(OUT_DIR, flow_dir, f"{filename}.png")
            shutil.move(src, dst)
            # A silent downscale failure would leave a full-size PNG behind and
            # still look like success, so the return code is checked.
            sips = subprocess.run(["sips", "-Z", "400", dst], capture_output=True)
            if sips.returncode != 0:
                print(f"  WARN: {flow_dir}/{filename}.png - sips downscale failed, PNG left at full size")
            print(f"  OK: {flow_dir}/{filename}.png ({os.path.getsize(dst) // 1024} KB)")
            saved += 1
        shutil.rmtree(tmpdir, ignore_errors=True)

        print(f"\nDone! Saved {saved}/{len(SCREENS)} screenshots to {OUT_DIR}")

        # A PARTIAL export must NOT look like success. The whole point of this
        # script is that the committed PNGs match the .pen; exiting 0 after
        # writing 136/156 is precisely how a stale screenshot survives review
        # (it did, 2026-08-20 — b4p-d shipped one revision behind the design).
        if failed_chunks or saved != len(SCREENS):
            print(f"ERROR: incomplete export ({saved}/{len(SCREENS)}, "
                  f"{failed_chunks} failed chunk(s)) - do NOT commit these screenshots")
            sys.exit(1)

    finally:
        proc.terminate()
        proc.wait(timeout=5)


if __name__ == "__main__":
    main()
