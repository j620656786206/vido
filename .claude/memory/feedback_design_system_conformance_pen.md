---
name: feedback-design-system-conformance-pen
description: .pen 設計稿字體必須嚴守 DL-v2 規範（含複製摘錄），新元件必須登錄進 Component Library 文件框
metadata: 
  node_type: memory
  type: feedback
  originSessionId: df2956e4-434e-4728-8097-69852a99691b
---

Alexyu 於 2026-07-04（13-0 requests design review）給的兩條硬性要求：

1. **字體規範零容忍**：字體已定義在 design system（DL-v2 §3）與 guideline，Review 時不得以「從既有 flow 複製來的既有債」為由放行新 frame 裡的違規。混合字串如「107 分」要拆成數字（JetBrains Mono）＋單位（Noto Sans TC）兩個 text node（gap 3-4 的小 frame，全案已有 cnt/cntUnit 前例）。
2. **新元件必須登錄 Component Library**：story 裡新建的 reusable 元件（如 `Component/RequestRow-v2`）要加進 `.pen` 的 Component Library 文件框（node `sJzat`）。v2 登錄慣例：對應分區（如 `content-cards-v2` row `ISilG`）加一個 cell = 垂直 frame gap 8，內含〔元件 ref 實例（可縮窄，如 720→480）＋ Noto Sans TC 12 `$text-muted` 說明文字在下〕。

**Why:** design system 是單一真相來源；複製既有畫面時違規會擴散，元件不登錄則後續 story 找不到可複用的單元。

**How to apply:** 設計 story 的 token/font lint 對「新 frame 內的所有內容」生效（不管來源）；收尾清單加一項「新 reusable 元件已登錄 component library」。相關：[[project_pen_flow_layout_convention]]、[[feedback_pencil_spec_standalone_screen]]。
