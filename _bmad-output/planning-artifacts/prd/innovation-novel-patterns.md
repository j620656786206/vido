# Innovation & Novel Patterns

## Detected Innovation Areas

**1. AI-Powered Fansub Filename Parsing (Market First)**

**Innovation Essence:**
Vido is the first media management tool to use Large Language Models (Gemini/Claude) to parse complex fansub naming conventions.

**Pain Point Addressed:**
Existing tools (Radarr, Sonarr, FileBot) rely on regular expressions and cannot handle:
- Fansub group tags: `[Leopard-Raws]`, `[Coalgirls]`, `【幻櫻字幕組】`
- Mixed-language titles: Japanese original + Chinese translation
- Non-standard episode markers: Absolute numbering, multi-episode packs, OVA/Movie labels
- Quality/encoding information: `[BD 1920x1080 x264 FLAC]`

**Technical Approach:**
```
Complex filename → LLM parsing → Structured information extraction → Search keyword generation
```

Example:
- Input: `【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4`
- AI Extraction:
  - Fansub group: 幻櫻字幕組 (ignore)
  - Title: 我的英雄學院
  - Episode: #1
  - Language: Traditional Chinese
- Search keywords: `"我的英雄學院" episode 1`

**Market Differentiation:**
- Radarr/Sonarr: Complete failure to parse → **Failed**
- FileBot: Relies on community scripts, low coverage → **Partial success**
- Vido: AI contextual understanding → **>93% success rate target**

---

**2. Multi-Source Resilience Architecture (Zero-Failure Design)**

**Innovation Essence:**
Four-layer metadata retrieval strategy ensuring "never complete failure":

```
Layer 1: TMDb API (zh-TW priority)
   ↓ (Failure - API down or not found)

Layer 2: Douban Web Scraper (Traditional Chinese)
   ↓ (Failure - Network issues or not found)

Layer 3: Wikipedia Search (Multilingual)
   - Search Traditional Chinese Wikipedia
   - Extract Infobox metadata (title, director, cast, year, genre)
   - ⚠️ No poster images (display default icon or use Layer 1/2 cache)
   ↓ (Failure - No matching entry)

Layer 4: AI Intelligent Search Assistant
   - Re-parse filename, generate multiple search keywords
   - Retry Layers 1-3 with new keywords
   ↓ (Still fails)

Layer 5: Manual Search Option
```

**Layer 3 Innovation: Wikipedia as Free Fallback**

Not "generating metadata from nothing," but:
1. Free, unlimited API access (MediaWiki API)
2. Multi-language support (zh.wikipedia.org for Traditional Chinese)
3. Rich text metadata from Infobox (director, cast, year, genre, plot)
4. High reliability (community-maintained, timely updates)
5. No poster images (limitation, but better than no metadata)

**Layer 4 Innovation: AI as Intelligent Search Assistant**

Not "generating metadata from nothing," but:
1. AI re-parses filename, extracts keywords from different angles
2. Generates multiple search strategies (original, translated, English, Romaji)
3. Auto-retries TMDb/Douban/Wikipedia searches
4. Selects best match from search results

**Example Scenario:**
- Filename: `鬼滅之刃.S01E26.mkv`
- TMDb search "鬼滅之刃" → No results (Simplified Chinese index)
- AI generates alternative searches:
  - "鬼灭之刃" (Simplified)
  - "Demon Slayer" (English)
  - "Kimetsu no Yaiba" (Japanese Romaji)
- TMDb search "Demon Slayer" → ✅ Found

**Market Differentiation:**
- Jellyfin/Plex: Single source, gives up on failure
- Vido: Multi-source + Wikipedia + AI retry, maximizes success rate

**Coverage Improvement:**
- Three-layer fallback (TMDb → Douban → Wikipedia): >98% metadata coverage
- Four-layer with AI retry: >99% metadata coverage (estimated)
- Even without posters, at least basic information available

---

**3. Traditional Chinese Priority Strategy (Execution Innovation)**

**Innovation Essence:**
Not a technical breakthrough, but market positioning and execution innovation.

**Differentiated Execution:**
- TMDb API default language: `zh-TW` (not `zh-CN`)
- Douban scraper: Traditional Chinese priority display
- Wikipedia: Search zh.wikipedia.org first
- AI prompts: Explicitly request Traditional Chinese output
- UI/UX: All metadata displays Traditional Chinese first

**Market Opportunity:**
Existing tools' Traditional Chinese support:
- Plex/Jellyfin: Mixed Simplified/Traditional, poor experience
- Radarr/Sonarr: Primarily English, very limited Chinese support
- Vido: Traditional Chinese as first-class citizen

---

**4. Pluggable Integration Architecture (v4)**

**Innovation Essence:**
Vido uses Go interfaces to create a pluggable integration layer for external services, allowing it to function as both a standalone tool and a unified UI for existing *arr stack setups.

**Plugin Types:**
```go
type MediaServerPlugin interface {  // Plex, Jellyfin
    SyncLibrary() ([]MediaItem, error)
    GetWatchHistory(userID string) ([]WatchRecord, error)
}

type DownloaderPlugin interface {   // qBittorrent, NZBGet
    AddDownload(request DownloadRequest) (string, error)
    GetStatus(id string) (DownloadStatus, error)
}

type DVRPlugin interface {          // Sonarr, Radarr
    AddMovie(tmdbID int, qualityProfile string) error
    AddSeries(tmdbID int, qualityProfile string, seasons []int) error
}
```

**Market Differentiation:**
- MediaManager: Requires *arr stack
- Sublarr: Requires *arr stack
- Vido: Works standalone OR with *arr stack (pluggable)

---

**5. Traditional Chinese Subtitle Scoring Engine (v4)**

**Innovation Essence:**
Multi-factor subtitle scoring algorithm that prioritizes Traditional Chinese quality:

```
Score = Language Match (40%) + Resolution Match (20%) + Source Trust (20%) + Group Reputation (10%) + Download Count (10%)
```

**Key Innovation — Content-Based Language Detection:**
- Analyze subtitle file content (not just filename) to determine 簡體/繁體
- Prevents the #1 Bazarr bug: Simplified Chinese subtitles blocking Traditional Chinese downloads
- OpenCC post-processing with cross-strait terminology correction (軟件→軟體, 內存→記憶體)

---

**6. SSE Hub for Real-Time NAS Events (v4)**

**Innovation Essence:**
Server-Sent Events hub replaces polling for real-time progress tracking across all Vido operations (downloads, scans, subtitle processing). Single connection, multiple event types, zero external dependencies.

---

## Market Context & Competitive Landscape

**Competitor Analysis:**

| Tool | Fansub Parsing | Metadata Sources | Traditional Chinese | Fallback Mechanism |
|------|---------------|------------------|---------------------|-------------------|
| **Radarr/Sonarr** | ❌ Failed | TMDb only | ⚠️ Limited | ❌ None |
| **FileBot** | ⚠️ Community scripts | TMDb/TVDb | ⚠️ Limited | ❌ None |
| **Jellyfin** | ❌ Failed | TMDb only | ⚠️ Mixed | ❌ None |
| **Plex** | ❌ Failed | Proprietary DB | ⚠️ Mixed | ⚠️ Limited |
| **Vido** | ✅ AI | TMDb→Douban→Wikipedia→AI | ✅ Native | ✅ Four-layer |

**Market Gap:**
- NAS user community (especially Taiwan/Hong Kong) has long complained about metadata issues
- Fansub naming is a common complaint on forums/Reddit
- Existing tools' Chinese support seen as "afterthought" rather than core feature

**Vido's Positioning:**
"The first media management tool designed specifically for Traditional Chinese users and fansub content"

---

## Validation Approach

**AI Parsing Accuracy Validation:**

**1. Benchmark Dataset**
- Collect 1,000 real fansub filenames
- Categories:
  - Standard naming (300)
  - Chinese fansub groups (300)
  - Japanese fansub groups (300)
  - Extreme complex cases (100)
- Manually annotate correct parsing results

**2. Accuracy Metrics**
- **Overall accuracy target**: >95%
  - Standard naming: >99%
  - Fansub naming: >93%
  - Extreme cases: >80%
- **Measurement**:
  - Title match: Exact match vs partial match
  - Episode accuracy: Precise to episode number
  - Metadata retrieval success rate

**3. User Satisfaction**
- MVP phase (first 50 users) collect feedback
- Question: "What percentage of your files did Vido successfully parse?"
- Target: >90% users report "most" or "all" successful

**4. Cost Monitoring**
- Track AI API usage
- Per-file average cost target: <$0.05 USD
- Per-user monthly cost target: <$2 USD (assuming 50 new files/month)

**5. Wikipedia Effectiveness**
- Track Layer 3 hit rate
- Measure cases where Wikipedia provides metadata when TMDb/Douban fail
- Monitor user acceptance of "no poster" metadata entries

---

## Risk Mitigation

**Risk 1: AI Cost Too High**

**Scenario**: Users heavily use AI parsing, API costs explode

**Mitigation**:
- **Caching Strategy**: Similar filename parsing results cached for 30 days
- **Smart Triggering**: Only invoke AI when basic regex fails
- **User-Paid**: Users provide their own API keys (1.0 strategy)
- **Degradation Option**: Users can disable AI parsing, fall back to basic mode

**Trigger**: If average cost >$0.10/file, activate cost optimization

---

**Risk 2: AI Accuracy Lower Than Expected**

**Scenario**: AI parsing accuracy <90%, users frequently need manual correction

**Mitigation**:
- **Hybrid Mode**: Regex + AI dual validation
  - If both agree → High confidence
  - If disagree → Prompt user to choose
- **Learning Mechanism**: After user manual correction, system remembers mapping
- **Community Dataset**: Collect user-contributed parsing rules
- **Fallback Option**: If AI consistently fails, suggest manual search

**Trigger**: If benchmark <93% or user satisfaction <85%

---

**Risk 3: External API Dependencies (TMDb/Douban/Wikipedia)**

**Scenario**: TMDb changes API, Douban upgrades anti-scraping, Wikipedia blocks requests

**Mitigation**:
- **Multi-Source Architecture**: One fails, auto-switch to next
- **Data Caching**: Already-fetched metadata permanently saved (local first)
- **NFO Backup**: Metadata exported as NFO files, not dependent on external services
- **API Version Locking**: TMDb API v3 → v4 upgrade prepared in advance
- **Wikipedia Compliance**: Respect MediaWiki API guidelines, set proper User-Agent

**Trigger**: Regular health checks, warn when API failure rate >10%

---

**Risk 4: Traditional Chinese Market Too Small**

**Scenario**: Target market (Taiwan/Hong Kong NAS users) cannot sustain development costs

**Mitigation**:
- **Internationalization Path**: Architecture supports multiple languages (though 1.0 prioritizes Traditional Chinese)
- **Open Source Strategy**: Open source project can gain community contributions
- **Cross-Domain Application**: AI parsing technology applicable to other domains (music, games)

---

**Risk 5: Wikipedia Metadata Quality Issues**

**Scenario**: Wikipedia entries have inconsistent Infobox formats or missing information

**Mitigation**:
- **Robust Parsing**: Handle multiple Infobox template variations
- **Validation**: Cross-check extracted data for consistency
- **Graceful Degradation**: If Infobox parsing fails, still mark as "searched but no data"
- **User Feedback**: Allow users to report Wikipedia metadata issues
- **Fallback to AI**: If Wikipedia data quality poor, proceed to Layer 4

**Trigger**: If Wikipedia success rate <50%, consider adjusting priority order
