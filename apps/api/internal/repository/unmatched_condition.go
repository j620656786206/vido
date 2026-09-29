package repository

// unmatchedCondition is the ONE definition of 「未匹配」 (story
// disc-2026-09-unmatched-filter-vs-parse-status AC #1 [@contract-v1]): the
// enrichment gave up on the row AND no source — TMDb, 豆瓣, Wikipedia, NFO or
// the user's own 修改資訊 — ever supplied its data. Both the 未匹配 filter and
// the 未匹配 (N) count use it, so the list always holds exactly N rows.
//
// Neither column alone will do. tmdb_id misses every non-TMDb match.
// parse_status is the pipeline's state, not the data's: a re-parse writes only
// parse_status=pending, and a TMDb outage during it lands failed with the old
// data still on the row. A pending row is 整理中, never 未匹配 (⚖️ Alexyu
// 2026-09-29). alias is the table alias ("" when the query has none).
func unmatchedCondition(alias string) string {
	col := func(name string) string {
		if alias == "" {
			return name
		}
		return alias + "." + name
	}
	return "(" + col("parse_status") + " = 'failed' AND (" +
		col("metadata_source") + " IS NULL OR " + col("metadata_source") + " = ''))"
}
