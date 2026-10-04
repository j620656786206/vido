---
workflowType: 'prd'
workflow: 'edit'
stepsCompleted: ['step-e-01-discovery', 'step-e-02-review', 'step-e-03-edit']
lastEdited: '2026-10-02'
editHistory:
  - date: '2026-10-02'
    changes: 'New FR P1-040 (opt-in anonymous usage report); NFR-S7/S8 reworded; north-star metric added to success-criteria.md; tech-considerations exception note; index links.'
---

# PRD Amendment: Opt-in Anonymous Usage Report

> **Amendment to:** NFR-S7, NFR-S8 (privacy), Success Criteria (north-star metric); adds P1-040
> **Date:** 2026-10-02
> **Author:** John (PM)
> **Source decision:** founder-pm product decision, 2026-10-02 (Alexyu)
> **Status:** APPROVED — Alexyu, 2026-10-02

---

## 1. Background & Motivation

### Goal shift

Vido's goal is to be **needed**: on many users' NAS, Vido automatically produces Traditional Chinese subtitles every week without the user opening it. Vido is a background tool; success is measured by work done, not by visits.

### Gap

Vido has no reporting mechanism. Installs, active use and uninstalls are invisible. The current post-release metrics (GitHub stars, Docker pulls) measure attention, not use.

### Constraint

Self-hosters are privacy-sensitive, and NFR-S7/S8 previously ruled out any external reporting. This amendment keeps the privacy floor — **media library content never leaves the NAS** — and opens exactly one exception: an anonymous, opt-in, counts-only weekly report.

### North-star metric

**Weekly installs that automatically produced ≥1 Traditional Chinese subtitle.** Opt-in only, so the reported number is a lower bound of real usage.

---

## 2. Functional Requirements (Amendment)

### P1-040: 匿名使用回報（自選開啟）(NEW)

| ID | 功能 | Priority | Description |
|----|------|----------|-------------|
| P1-040-1 | 預設關閉、隨時可關 | P0 | Reporting is off by default. It turns on only by an explicit user action (Setup Wizard or Settings). The user can turn it off at any time; after turning it off, no further report is sent. |
| P1-040-2 | 設定精靈詢問一次 | P0 | The Setup Wizard asks once, stating what is sent and what is never sent. The pre-selected answer is "off". |
| P1-040-3 | 送出內容透明 | P0 | Settings shows the exact content of the most recent report, verbatim, with its send time. If nothing has been sent, it says so. |
| P1-040-4 | 回報內容 | P0 | A report contains only: (a) a random installation identifier — not derived from hardware, user, account or network identifiers; (b) Vido version; (c) number of subtitles automatically produced in the past 7 days. |
| P1-040-4b | 字幕來源分布 | P1 | Optionally, the count in P1-040-4(c) broken down by source path: embedded-subtitle translation / online source / speech recognition. |
| P1-040-5 | 頻率 | P0 | At most one report per installation per 7 days. |
| P1-040-6 | 失敗不打擾 | P0 | A failed send produces no user-visible error, does not affect any other feature, and is not retried before the next scheduled window. |
| P1-040-7 | 文件範例 | P1 | User documentation (EN + zh-TW) includes a literal example of a sent report and the list of what is never sent. |

**Field names and wire format** are intentionally not specified here; they are decided in architecture / story preparation.

---

## 3. Non-Functional Requirements

| ID | Requirement | Verification |
|----|-------------|--------------|
| NFR-T1 | A report contains no field outside the P1-040-4 / P1-040-4b allow-list | Automated test asserts the payload's field set equals the allow-list |
| NFR-T2 | No media-identifying data (titles, filenames, paths, metadata, API keys) appears in any report | Automated test with a populated library asserts none of these values occur in the payload |
| NFR-T3 | Reporting is off on a fresh install | Automated test: fresh install sends nothing |
| NFR-T4 | ≤1 report per installation per 7 days | Automated test with a controlled clock |
| NFR-T5 | Send failure causes 0 user-visible errors and 0 impact on other features | Automated test with the receiver unreachable |
| NFR-T6 | The receiving side does not store sender IP addresses | Receiver configuration review before launch |

---

## 4. Open Technical Questions (for Architecture / Story)

These are **not** decided by this PRD. Any answer is acceptable if Sections 2–3 hold.

1. **Receiver location.** Candidate: the self-hosted Umami already running on Alex's NAS (exposed via Cloudflare Tunnel).
2. **If Umami:** its database has no scheduled backup (see ai-context `ops-risk.md`); add a backup before collecting this data. Confirm Umami can be configured not to store IPs (NFR-T6).
3. **Field names, payload format, identifier generation.**

---

## 5. Timeline

- Does **not** block the 10-external-tester experiment — those 10 are asked directly by Alex.
- Must ship **before** recruiting testers 11–30.

---

## 6. Out of Scope

- Any media library data, viewing activity or per-title information
- Per-user behavior tracking or session analytics
- Crash / error reporting
- Storing sender IP addresses
- Enabling reporting without explicit user action

---

## 7. Success Criteria

1. A fresh install sends nothing until the user opts in.
2. An opted-in install sends at most one report per 7 days containing only allow-listed fields.
3. Settings shows the last report verbatim.
4. The north-star metric (Section 1) can be read from received reports.
