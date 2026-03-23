# Success Criteria

## Phase 1 Validation Metrics

These metrics must be met before Phase 1 (Subtitle Core MVP) can be considered complete.

| Metric | Target |
|--------|--------|
| Standard filename parsing success rate | >99% |
| Fansub filename parsing success rate | >95% |
| Traditional Chinese subtitle search hit rate | >85% |
| No false-positive Simplified Chinese subtitles | 100% |
| Cross-strait terminology correction accuracy | >95% |
| Docker startup to usable | <10 seconds |

## Phase 1 Post-Release (3 Months)

| Metric | Target |
|--------|--------|
| GitHub Stars | >100 |
| Docker Hub pulls | >500 |
| Community-reported zh-TW subtitle bugs | <5 |
| Personal dogfooding: replaces Bazarr in daily use | Yes |

## Phase 2 Post-Release (6 Months)

| Metric | Target |
|--------|--------|
| GitHub Stars | >500 |
| Docker Hub pulls | >2,000 |
| Organic discussion on PTT/巴哈 NAS boards | Yes |
| Personal dogfooding: replaces Seerr in daily use | Yes |

## Phase 4 Complete (12 Months)

| Metric | Target |
|--------|--------|
| GitHub Stars | >1,000 |
| Docker Hub pulls | >5,000 |
| External contributors | >5 |
| Recommended tool in zh-TW NAS communities | Yes |

## Technical Success Metrics

| Metric | Target |
|--------|--------|
| API response time (p95) | <500ms |
| Grid scrolling frame rate | 60 FPS |
| Docker image size | <500MB |
| SQLite query time (p95) | <300ms |
| Zero downtime on metadata source failures | Yes (multi-source fallback) |
