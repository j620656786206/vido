#!/usr/bin/env python3
"""設計稿節點存在性檢查（disc-2026-09-rule21-node-existence-unguarded）。

`apps/web/src` 的元件檔頭會寫它對應設計稿上的哪個節點：

    // Implements: Component/ButtonCost/Default (qAERt)
    // Design ref: ux-design.pen Screen E5-D (hUVYm) · E5-M (P0P82x)

ESLint 的 `local/implements-pen-node-id` 只檢查**格式**，不檢查那個節點在
`ux-design.pen` 裡還在不在。設計稿刪掉或換掉一張畫面後，檔頭照樣通過 lint，
看起來像是對齊過的，其實指向空氣——2026-09 一次查出 12 個檔案是這樣。

`.pen` 是加密的，CI 讀不到，所以 `export-pen-screenshots.py` 每次都把設計稿裡
所有 frame／group／元件實例的 id 寫進 `_bmad-output/pen-node-ids.json`（帶著
`.pen` 的 sha256）。這支腳本拿那份快照比對檔頭裡的每一個 id。

只想更新快照、不重新匯出截圖：
    python3 scripts/export-pen-screenshots.py --snapshot-only   # 要開著 Pencil

用法:
    python3 scripts/check-design-refs.py     # 有指向不存在節點的檔頭就 exit 1
"""

from __future__ import annotations

import hashlib
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "apps/web/src"
PEN_FILE = ROOT / "ux-design.pen"
SNAPSHOT = ROOT / "_bmad-output/pen-node-ids.json"

# The two header forms that name a node; the id is the parenthesised token.
# Only the file's leading comment block counts, as in the ESLint rule.
MARKER_RE = re.compile(r"^(Implements:\s*Component/|Design ref:\s*ux-design\.pen\s+Screen\s)")
NODE_ID_RE = re.compile(r"\(([A-Za-z0-9]+)\)")
COMMENT_PREFIX_RE = re.compile(r"^\s*(//+|/\*+|\*+/?)\s?")


def leading_comment_lines(text: str) -> list[str]:
    """The comment lines before the first line of code (blank lines skipped)."""
    out = []
    in_block = False
    for raw in text.splitlines():
        line = raw.strip()
        if not line:
            continue
        if in_block or line.startswith(("//", "/*", "*")):
            if line.startswith("/*"):
                in_block = True
            if in_block and "*/" in line:
                in_block = False
            out.append(COMMENT_PREFIX_RE.sub("", line).strip())
            continue
        break
    return out


def referenced_ids(path: Path) -> list[str]:
    ids = []
    for line in leading_comment_lines(path.read_text(encoding="utf-8")):
        if MARKER_RE.match(line):
            ids.extend(NODE_ID_RE.findall(line))
    return ids


def source_files() -> list[Path]:
    return sorted(
        p
        for p in SRC.rglob("*")
        if p.suffix in (".ts", ".tsx") and ".spec." not in p.name and p.is_file()
    )


def main() -> int:
    if not SNAPSHOT.exists():
        print(f"找不到 {SNAPSHOT.relative_to(ROOT)}——開著 Pencil 跑 "
              "python3 scripts/export-pen-screenshots.py --snapshot-only")
        return 1
    snap = json.loads(SNAPSHOT.read_text(encoding="utf-8"))
    problems = []
    if PEN_FILE.exists():
        actual = hashlib.sha256(PEN_FILE.read_bytes()).hexdigest()
        if snap.get("penSha256") != actual:
            problems.append(
                "  pen-node-ids.json: 快照過期——ux-design.pen 改過但沒重跑 "
                "scripts/export-pen-screenshots.py，快照裡的節點不是設計稿現在的樣子"
            )
    known = set(snap.get("nodeIds", []))

    checked = 0
    for path in source_files():
        for node_id in referenced_ids(path):
            checked += 1
            if node_id not in known:
                problems.append(
                    f"  {path.relative_to(ROOT)}: 檔頭指向 ({node_id})，設計稿裡沒有這個節點"
                )

    if problems:
        print("設計稿節點檢查失敗：")
        print("\n".join(problems))
        print("\n改法：換成設計稿上現在對應的節點；真的沒有對應畫面時，改用 "
              "`// Design ref: ux-design.pen — no current screen frame; <原因>`。")
        return 1
    print(f"設計稿節點檢查通過：{checked} 個檔頭節點都在設計稿裡（快照 {len(known)} 個節點）。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
