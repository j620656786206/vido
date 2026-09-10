#!/usr/bin/env python3
"""設計 token 漂移檢查器。

這個專案把同一組色彩 token 寫在四個地方：

  1. apps/web/src/styles.css                     ← 實作值（:root 夜行 / [data-theme=light] 日巡）
  2. DESIGN.md                                   ← 設計系統文件的 frontmatter
  3. _bmad-output/design-context-pack.md         ← 給 Pencil 內建 AI 的 primer（它讀不到 repo）
  4. .impeccable/design.json                     ← impeccable 產生的側車檔

2026-09 的教訓：它們漂移了半年沒人發現，設計稿因此整整落後一套色盤。四份手動同步
一定會漏——第一次修的時候就漏了 design.json 裡的十進位 rgba()。

**這支腳本刻意不決定「誰是對的」。** 它只回報「四份不一致」，因為那是一個設計決策，
不是機械替換：有時候是文件過期，有時候是程式碼實作錯了。`--apply` 只在你已經判斷
「styles.css 的值就是想要的設計」時才用，而且只會改 DESIGN.md（另外兩份帶著手寫
註解，值得人工處理）。

用法:
    python3 scripts/check-design-tokens.py           # 檢查，不一致就 exit 1
    python3 scripts/check-design-tokens.py --apply   # 用 styles.css 的值改寫 DESIGN.md
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
STYLES = ROOT / "apps/web/src/styles.css"
DESIGN_MD = ROOT / "DESIGN.md"
CONTEXT_PACK = ROOT / "_bmad-output/design-context-pack.md"
IMPECCABLE = ROOT / ".impeccable/design.json"

# 陰影用的中性色不屬於色盤，不納入比對。
SHADOW_INK = {"#000000", "#0c1512", "#16231d"}


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def styles_tokens() -> tuple[dict[str, str], dict[str, str]]:
    """從 styles.css 取出夜行與日巡兩組 token。"""
    css = read(STYLES)

    def block(pattern: str, label: str) -> str:
        m = re.search(pattern, css, re.M)
        if not m:
            sys.exit(f"styles.css 找不到 {label} 區塊")
        return m.group(1)

    dark = block(r"^:root\s*\{([\s\S]*?)^\}", ":root（夜行）")
    light = block(r"^\[data-theme=['\"]light['\"]\]\s*\{([\s\S]*?)^\}", "[data-theme=light]（日巡）")

    def pairs(chunk: str) -> dict[str, str]:
        return {
            name: value.lower()
            for name, value in re.findall(r"--([a-z0-9-]+):\s*(#[0-9a-fA-F]{6,8})\b", chunk)
        }

    return pairs(dark), pairs(light)


def design_md_colors() -> dict[str, str]:
    """DESIGN.md frontmatter 的 colors: 區塊。"""
    fm = read(DESIGN_MD).split("---", 2)[1]
    m = re.search(r"^colors:\n((?:[ \t]+.*\n)+)", fm, re.M)
    if not m:
        sys.exit("DESIGN.md frontmatter 找不到 colors: 區塊")
    return {
        name: value.lower()
        for name, value in re.findall(r"^\s+([a-z0-9-]+):\s*'(#[0-9a-fA-F]{6,8})'", m.group(1), re.M)
    }


def context_pack_colors() -> tuple[dict[str, str], dict[str, str]]:
    """design-context-pack.md §3 的雙欄色表（夜行在前、日巡在後）。"""
    text = read(CONTEXT_PACK)
    dark: dict[str, str] = {}
    light: dict[str, str] = {}
    for line in text.splitlines():
        m = re.search(r"--([a-z0-9-]+)\s+(#[0-9a-fA-F]{6,8})(?:\s+(#[0-9a-fA-F]{6,8}))?", line)
        if not m:
            continue
        name, first, second = m.group(1), m.group(2).lower(), m.group(3)
        dark[name] = first
        if second:
            light[name] = second.lower()
    return dark, light


def impeccable_stray_colors(palette: set[str]) -> list[str]:
    """design.json 裡不屬於現行色盤的色值（含十進位 rgba，這是上次漏掉的那種）。"""
    text = read(IMPECCABLE)
    stray: list[str] = []

    for hexv in {h.lower() for h in re.findall(r"#[0-9a-fA-F]{6}\b", text)}:
        if hexv not in palette and hexv not in SHADOW_INK and hexv != "#ffffff":
            stray.append(hexv)

    for rgba in set(re.findall(r"rgba\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,[^)]*\)", text)):
        hexv = "#{:02x}{:02x}{:02x}".format(*(int(c) for c in rgba))
        if hexv in SHADOW_INK or hexv == "#ffffff":
            continue
        if hexv not in palette:
            stray.append(f"rgba{rgba} (= {hexv})")

    return sorted(stray)


def compare(label: str, expected: dict[str, str], actual: dict[str, str]) -> list[str]:
    problems = []
    for name, want in expected.items():
        got = actual.get(name)
        if got is None:
            problems.append(f"  {label}: 缺少 --{name}（styles.css 是 {want}）")
        elif got != want:
            problems.append(f"  {label}: --{name} 寫著 {got}，styles.css 是 {want}")
    for name in actual:
        if name not in expected:
            problems.append(f"  {label}: --{name} 在 styles.css 不存在")
    return problems


def apply_to_design_md(dark: dict[str, str]) -> None:
    text = read(DESIGN_MD)
    block = "colors:\n" + "".join(f"  {name}: '{value}'\n" for name, value in dark.items())
    new = re.sub(r"^colors:\n(?:[ \t]+.*\n)+", block, text, count=1, flags=re.M)
    if new == text:
        print("DESIGN.md 沒有需要改的地方")
        return
    DESIGN_MD.write_text(new, encoding="utf-8")
    print(f"已用 styles.css 的 {len(dark)} 個值改寫 DESIGN.md 的 colors:")
    print("記得跑 npx prettier --write DESIGN.md")


def main() -> int:
    dark, light = styles_tokens()

    if "--apply" in sys.argv:
        apply_to_design_md(dark)
        return 0

    problems: list[str] = []
    problems += compare("DESIGN.md", dark, design_md_colors())

    pack_dark, pack_light = context_pack_colors()
    problems += compare("context-pack（夜行）", dark, pack_dark)
    problems += compare("context-pack（日巡）", light, pack_light)

    palette = set(dark.values()) | set(light.values())
    stray = impeccable_stray_colors(palette)
    problems += [f"  design.json: {s} 不屬於現行色盤" for s in stray]

    if problems:
        print("設計 token 對不上：\n")
        print("\n".join(problems))
        print(f"\n共 {len(problems)} 處。")
        print("這不一定是文件錯了——也可能是 styles.css 的實作值該改。")
        print("先判斷哪一邊才是對的設計，再修對應那一邊。")
        return 1

    print(f"設計 token 一致：夜行 {len(dark)} 個、日巡 {len(light)} 個，四份文件對得上。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
