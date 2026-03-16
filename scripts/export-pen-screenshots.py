#!/usr/bin/env python3
"""Export screenshots from ux-design.pen organized by user flow.

Usage:
    python3 scripts/export-pen-screenshots.py

Requires Pencil.app to be running. Starts a temporary MCP HTTP server,
captures all design screens, and saves PNGs to _bmad-output/screenshots/.
"""

import json
import base64
import os
import subprocess
import sys
import time
import signal

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PEN_FILE = os.path.join(PROJECT_ROOT, "ux-design.pen")
OUT_DIR = os.path.join(PROJECT_ROOT, "_bmad-output", "screenshots")
MCP_BIN = "/Applications/Pencil.app/Contents/Resources/app.asar.unpacked/out/mcp-server-darwin-x64"
MCP_PORT = 9876

# Screen ID -> (flow_folder, filename)
SCREENS = {
    # Flow A - Desktop browse
    "4VILE": ("flow-a-browse-desktop", "09a-empty-library-desktop"),
    "IpZhv": ("flow-a-browse-desktop", "09b-loading-skeleton-desktop"),
    "KNI8F": ("flow-a-browse-desktop", "01-library-grid-desktop"),
    "LZ8Ds": ("flow-a-browse-desktop", "06-list-view-desktop"),
    # Flow B - Desktop hover + detail
    "Qm662": ("flow-b-hover-detail-desktop", "01b-postercard-hover-state"),
    "auArc": ("flow-b-hover-detail-desktop", "04a-postercard-context-menu"),
    "RgSxQ": ("flow-b-hover-detail-desktop", "04-detail-panel-movie-desktop"),
    "407vK": ("flow-b-hover-detail-desktop", "04b-detail-panel-tv-series-desktop"),
    "7mdTJ": ("flow-b-hover-detail-desktop", "04c-detail-panel-context-menu"),
    # Flow C - Desktop search/filter/settings
    "rsAxf": ("flow-c-search-filter-settings-desktop", "07-search-filter-desktop"),
    "dcf67": ("flow-c-search-filter-settings-desktop", "08-batch-operations-desktop"),
    "7fE0b": ("flow-c-search-filter-settings-desktop", "01a-settings-gear-dropdown"),
    # Flow D - Mobile browse
    "OYqNo": ("flow-d-browse-mobile", "09a-m-empty-library-mobile"),
    "RxdY5": ("flow-d-browse-mobile", "09b-m-loading-skeleton-mobile"),
    "GOL63": ("flow-d-browse-mobile", "03-library-grid-mobile"),
    "3aSCw": ("flow-d-browse-mobile", "03a-m-sort-bottom-sheet"),
    "oypj1": ("flow-d-browse-mobile", "10-filter-bottom-sheet-mobile"),
    # Flow E - Mobile interaction
    "1UHzI": ("flow-e-interaction-mobile", "04a-m-postercard-context-menu-mobile"),
    "kcn1v": ("flow-e-interaction-mobile", "05-detail-panel-mobile"),
    "APfjC": ("flow-e-interaction-mobile", "04c-m-detail-context-menu-mobile"),
    # Flow F - Mobile batch/settings
    "0KOE7": ("flow-f-batch-settings-mobile", "08-m-batch-operations-mobile"),
    "IfrPQ": ("flow-f-batch-settings-mobile", "01a-m-settings-bottom-sheet-mobile"),
    # Flow G - Settings pages
    "6UCtX": ("flow-c-search-filter-settings-desktop", "10-settings-desktop"),
    "2H4OM": ("flow-f-batch-settings-mobile", "10-m-settings-mobile"),
}


def start_mcp_server():
    proc = subprocess.Popen(
        [MCP_BIN, "--app", "desktop", "--http", "--http-port", str(MCP_PORT)],
        stdout=subprocess.PIPE, stderr=subprocess.PIPE,
    )
    time.sleep(2)
    return proc


def mcp_call(session_id, req_id, method, params):
    headers = [
        "-H", "Content-Type: application/json",
        "-H", "Accept: application/json, text/event-stream",
    ]
    if session_id:
        headers += ["-H", f"Mcp-Session-Id: {session_id}"]

    req = json.dumps({"jsonrpc": "2.0", "id": req_id, "method": method, "params": params})
    result = subprocess.run(
        ["curl", "-s", "-i", "-X", "POST", f"http://localhost:{MCP_PORT}/mcp"] + headers + ["-d", req],
        capture_output=True, text=True, timeout=30,
    )
    # Extract session ID from headers
    new_session = session_id
    body = ""
    for line in result.stdout.split("\n"):
        if line.lower().startswith("mcp-session-id:"):
            new_session = line.split(":", 1)[1].strip()
        if line.startswith("{"):
            body = line
    return new_session, json.loads(body) if body else None


def main():
    if not os.path.exists(MCP_BIN):
        print("ERROR: Pencil.app not found at /Applications/Pencil.app")
        sys.exit(1)

    print("Starting Pencil MCP server...")
    proc = start_mcp_server()

    try:
        # Initialize
        session, resp = mcp_call(None, 1, "initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "screenshot-export", "version": "1.0"},
        })
        if not resp:
            print("ERROR: Failed to connect to Pencil MCP server")
            sys.exit(1)
        print(f"Connected (session: {session[:20]}...)")

        # Send initialized notification
        subprocess.run(
            ["curl", "-s", "-X", "POST", f"http://localhost:{MCP_PORT}/mcp",
             "-H", "Content-Type: application/json",
             "-H", f"Mcp-Session-Id: {session}",
             "-d", json.dumps({"jsonrpc": "2.0", "method": "notifications/initialized"})],
            capture_output=True, timeout=5,
        )

        # Create output directories
        for flow_dir, _ in SCREENS.values():
            os.makedirs(os.path.join(OUT_DIR, flow_dir), exist_ok=True)

        # Export screenshots
        saved = 0
        for i, (node_id, (flow_dir, filename)) in enumerate(SCREENS.items()):
            session, resp = mcp_call(session, i + 10, "tools/call", {
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
