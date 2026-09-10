#!/usr/bin/env python3
"""設計 token 漂移檢查器。

這個專案把同一組設計 token 寫在五個地方：

  1. apps/web/src/styles.css                     ← 實作值（:root 夜行 / [data-theme=light] 日巡）
  2. DESIGN.md                                   ← 設計系統文件的 frontmatter
  3. _bmad-output/design-context-pack.md         ← 給 Pencil 內建 AI 的 primer（它讀不到 repo）
  4. .impeccable/design.json                     ← impeccable 產生的側車檔
  5. ux-design.pen                               ← 設計稿本身（**漂移半年沒人發現的就是這一份**）

第 5 份沒辦法在 CI 裡直接讀——`.pen` 是加密的，只有跑著的 Pencil.app 讀得到。
所以 `export-pen-screenshots.py` 每次都會把設計稿的變數 dump 成
`_bmad-output/pen-tokens.json`，這支腳本比對那份快照。快照裡帶著 `.pen` 的
sha256，改了設計稿卻沒重跑匯出腳本時，雜湊對不上，這裡會直接說出來。

除了顏色，現在也守門：圓角、間距階梯、字級與行高、以及「裸數字」——
gap／padding／fontSize／lineHeight 只要出現一個沒吃變數的數字就算漂移。

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

import hashlib
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
STYLES = ROOT / "apps/web/src/styles.css"
DESIGN_MD = ROOT / "DESIGN.md"
CONTEXT_PACK = ROOT / "_bmad-output/design-context-pack.md"
IMPECCABLE = ROOT / ".impeccable/design.json"
PEN_FILE = ROOT / "ux-design.pen"
PEN_TOKENS = ROOT / "_bmad-output/pen-tokens.json"

# frontmatter 的 typography 鍵 → 設計稿的角色名。
TYPE_ROLES = {
    "display": "Display", "h1": "H1", "h2": "H2", "h3": "H3", "h4": "H4",
    "body-lg": "BodyLg", "body": "Body", "label": "Label",
}

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


def pen_snapshot() -> dict | None:
    """設計稿變數快照。不存在就回 None（並在 main 裡報告）。"""
    if not PEN_TOKENS.exists():
        return None
    return json.loads(read(PEN_TOKENS))


def pen_value(entry: dict, mode: str = "dark", bp: str = "desktop"):
    """把一個變數定義解析成單一值。

    設計稿的變數可以掛兩個軸（mode: dark/light、bp: desktop/mobile）。沒掛軸的
    直接回值；掛了軸的挑指定組合，挑不到就退回第一筆——這與 Pencil 自己的解析
    規則一致（沒標主題的節點吃第一個值）。
    """
    v = entry.get("value")
    if not isinstance(v, list):
        return v
    for want in ({"mode": mode, "bp": bp}, {"mode": mode}, {"bp": bp}):
        for item in v:
            theme = item.get("theme") or {}
            if all(theme.get(k) == val for k, val in want.items()):
                return item.get("value")
    return v[0].get("value") if v else None


def check_pen_freshness(snap: dict) -> list[str]:
    if not PEN_FILE.exists():
        return ["  pen-tokens.json: 找不到 ux-design.pen，無法驗證快照是否過期"]
    actual = hashlib.sha256(PEN_FILE.read_bytes()).hexdigest()
    if snap.get("penSha256") != actual:
        return ["  pen-tokens.json: 快照過期——ux-design.pen 改過但沒重跑 "
                "scripts/export-pen-screenshots.py，快照裡的 token 不是設計稿現在的值"]
    return []


def check_pen_colors(snap: dict, dark: dict[str, str], light: dict[str, str]) -> list[str]:
    problems = []
    variables = snap.get("variables", {})
    for name, want in dark.items():
        entry = variables.get(name)
        if entry is None or entry.get("type") != "color":
            problems.append(f"  設計稿: 缺少顏色變數 {name}（styles.css 是 {want}）")
            continue
        got = str(pen_value(entry, "dark")).lower()
        if got != want:
            problems.append(f"  設計稿（夜行）: {name} 是 {got}，styles.css 是 {want}")
        got_l = str(pen_value(entry, "light")).lower()
        want_l = light.get(name)
        if want_l and got_l != want_l:
            problems.append(f"  設計稿（日巡）: {name} 是 {got_l}，styles.css 是 {want_l}")
    return problems


def check_pen_radius(snap: dict) -> list[str]:
    """圓角：設計稿的 radius-* 必須等於 styles.css 的 --radius-*。"""
    css = read(STYLES)
    want = {m[0]: int(m[1]) for m in re.findall(r"--radius-([a-z]+):\s*(\d+)px", css)}
    problems = []
    variables = snap.get("variables", {})
    for name, px in want.items():
        entry = variables.get(f"radius-{name}")
        if entry is None:
            problems.append(f"  設計稿: 缺少 radius-{name}（styles.css 是 {px}px）")
        elif pen_value(entry) != px:
            problems.append(f"  設計稿: radius-{name} 是 {pen_value(entry)}，styles.css 是 {px}")
    return problems


def frontmatter_block(key: str) -> str:
    fm = read(DESIGN_MD).split("---", 2)[1]
    m = re.search(rf"^{key}:\n((?:[ \t]+.*\n)+)", fm, re.M)
    if not m:
        sys.exit(f"DESIGN.md frontmatter 找不到 {key}: 區塊")
    return m.group(1)


def check_pen_spacing(snap: dict) -> list[str]:
    """間距：設計稿的 Space/* 必須與 DESIGN.md frontmatter 的階梯一字不差。"""
    want = {
        name: int(px)
        for name, px in re.findall(r"^\s+([a-z0-9-]+):\s*'(\d+)px'", frontmatter_block("spacing"), re.M)
    }
    variables = snap.get("variables", {})
    have = {k[len("Space/"):]: pen_value(v) for k, v in variables.items() if k.startswith("Space/")}
    problems = []
    for name, px in want.items():
        if name not in have:
            problems.append(f"  設計稿: 缺少 Space/{name}（DESIGN.md 是 {px}px）")
        elif have[name] != px:
            problems.append(f"  設計稿: Space/{name} 是 {have[name]}，DESIGN.md 是 {px}")
    for name in have:
        if name not in want:
            problems.append(f"  設計稿: Space/{name} 在 DESIGN.md 的階梯裡不存在")
    return problems


def check_pen_type(snap: dict) -> list[str]:
    """字級與行高：設計稿的 Type/*/Size（桌機）與 /Line 必須對上 frontmatter。"""
    block = frontmatter_block("typography")
    want: dict[str, dict[str, float]] = {}
    cur = None
    for line in block.splitlines():
        m = re.match(r"^  ([a-z0-9-]+):$", line)
        if m:
            cur = m.group(1)
            continue
        if cur is None:
            continue
        m = re.match(r"^\s+fontSize:\s*'([0-9.]+)rem'", line)
        if m:
            want.setdefault(cur, {})["size"] = float(m.group(1)) * 16
        m = re.match(r"^\s+lineHeight:\s*([0-9.]+)", line)
        if m:
            want.setdefault(cur, {})["line"] = float(m.group(1))

    variables = snap.get("variables", {})
    problems = []
    for fm_key, role in TYPE_ROLES.items():
        spec = want.get(fm_key)
        if not spec:
            continue
        size = variables.get(f"Type/{role}/Size")
        line = variables.get(f"Type/{role}/Line")
        if size is None or line is None:
            problems.append(f"  設計稿: 缺少 Type/{role}/Size 或 /Line")
            continue
        got_size = pen_value(size, bp="desktop")
        if float(got_size) != spec["size"]:
            problems.append(
                f"  設計稿: Type/{role}/Size（桌機）是 {got_size}，"
                f"DESIGN.md 的 {fm_key} 是 {spec['size']:g}px"
            )
        if abs(float(pen_value(line)) - spec["line"]) > 1e-6:
            problems.append(
                f"  設計稿: Type/{role}/Line 是 {pen_value(line)}，"
                f"DESIGN.md 的 {fm_key} 是 {spec['line']}"
            )
    return problems


def check_pen_raw_usage(snap: dict) -> list[str]:
    """裸數字：gap／padding／fontSize／lineHeight 一個都不准不吃變數。

    這條是「間距與字級有沒有漂移」唯一機器抓得到的方式。只要允許裸數字存在，
    檢查就只能靠人眼掃。
    """
    raw = snap.get("raw", {})
    labels = {"gap": "gap", "padding": "padding", "fontSize": "字級", "lineHeight": "行高"}
    return [
        f"  設計稿: 有 {count} 個{labels[key]}沒吃變數（應為 0）"
        for key, count in raw.items()
        if count and key in labels
    ]


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

    snap = pen_snapshot()
    if snap is None:
        problems.append("  設計稿: 找不到 _bmad-output/pen-tokens.json"
                        "——跑一次 scripts/export-pen-screenshots.py 產生它")
        pen_summary = ""
    else:
        problems += check_pen_freshness(snap)
        problems += check_pen_colors(snap, dark, light)
        problems += check_pen_radius(snap)
        problems += check_pen_spacing(snap)
        problems += check_pen_type(snap)
        problems += check_pen_raw_usage(snap)
        counts = snap.get("counts", {})
        pen_summary = (f"，設計稿 {len(snap.get('variables', {}))} 個變數、"
                       f"{counts.get('exportedScreens', '?')} 張畫面、"
                       f"{counts.get('masters', '?')} 個母版")

    if problems:
        print("設計 token 對不上：\n")
        print("\n".join(problems))
        print(f"\n共 {len(problems)} 處。")
        print("這不一定是文件錯了——也可能是 styles.css 的實作值該改。")
        print("先判斷哪一邊才是對的設計，再修對應那一邊。")
        return 1

    print(f"設計 token 一致：夜行 {len(dark)} 個、日巡 {len(light)} 個，五份來源對得上{pen_summary}。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
