#!/usr/bin/env python3
"""Export screenshots from ux-design.pen organized by user flow.

Usage:
    python3 scripts/export-pen-screenshots.py

Requires Pencil.app to be running. Spawns the Pencil MCP server in stdio mode
(Pencil 1.1.61 removed the old `--http` transport), captures each design screen,
and saves PNGs to _bmad-output/screenshots/.

Layout convention (2026-06-05 A–J merged-block rework):
  Screens are named with flow codes `{Flow}{seq}-{D|M}` (desktop/mobile) on the
  canvas. Screenshots mirror that: one folder per flow, filename == lowercased code.
  See .claude/memory/project_pen_flow_layout_convention.md.
"""

import json
import base64
import os
import subprocess
import sys

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PEN_FILE = os.path.join(PROJECT_ROOT, "ux-design.pen")
OUT_DIR = os.path.join(PROJECT_ROOT, "_bmad-output", "screenshots")
MCP_BIN = "/Applications/Pencil.app/Contents/Resources/app.asar.unpacked/out/mcp-server-darwin-arm64"

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
    "YEqii": ("flow-i-advanced-search", "i5-d"),  # rail persistent (hero)
    "SPMwD": ("flow-i-advanced-search", "i6-d"),  # rail collapsed + filtered no-results
    "m3yZy": ("flow-i-advanced-search", "i7-d"),  # rail states spec (genre loading / load-failed)
    # Flow J — 設計決策 spec (PosterCard info-density & polish)
    "XlFIq": ("flow-j-specs", "j1-d"),
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
    if not os.path.exists(MCP_BIN):
        print("ERROR: Pencil.app not found at /Applications/Pencil.app")
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

        # Export screenshots
        saved = 0
        for i, (node_id, (flow_dir, filename)) in enumerate(SCREENS.items()):
            resp = mcp_call(proc, i + 10, "tools/call", {
                "name": "get_screenshot",
                "arguments": {"filePath": PEN_FILE, "nodeId": node_id},
            })
            if not resp:
                print(f"  FAIL: {filename} - no response")
                continue

            contents = resp.get("result", {}).get("content", [])
            for item in contents:
                if item.get("type") == "image":
                    img_data = base64.b64decode(item["data"])
                    path = os.path.join(OUT_DIR, flow_dir, f"{filename}.png")
                    with open(path, "wb") as f:
                        f.write(img_data)
                    print(f"  OK: {flow_dir}/{filename}.png ({len(img_data) // 1024} KB)")
                    saved += 1
                    break
            else:
                print(f"  SKIP: {filename} - no image data")

        print(f"\nDone! Saved {saved}/{len(SCREENS)} screenshots to {OUT_DIR}")

    finally:
        proc.terminate()
        proc.wait(timeout=5)


if __name__ == "__main__":
    main()
