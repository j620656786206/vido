// Command douban-spike is the 9R-S3 live probe: does vido's Douban scraper
// return real zh metadata against the LIVE site today (search page is
// JS-rendered — the spike question), and does the subject-page scrape work?
//
// Run: go run ./cmd/douban-spike
// Pass: end-to-end parse of a real result; else drop Douban from the 在地化 chain.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/vido/api/internal/douban"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cfg := douban.DefaultConfig()
	client := douban.NewClient(cfg, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	fmt.Println("=== 9R-S3 Douban live probe ===")

	// Probe 1: search page (search.douban.com subject_search — JS-rendered?)
	searcher := douban.NewSearcher(client, logger)
	fmt.Println("\n[P1] Searcher.Search(\"全面啟動\", movie)")
	results, err := searcher.Search(ctx, "全面啟動", douban.MediaTypeMovie)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		fmt.Printf("  results: %d\n", len(results))
		for i, r := range results {
			if i >= 3 {
				break
			}
			fmt.Printf("  [%d] id=%s title=%q year=%d url=%s\n", i, r.ID, r.Title, r.Year, r.URL)
		}
	}

	// Probe 2: subject detail page (movie.douban.com/subject/3541415/ = Inception)
	scraper := douban.NewScraper(client, logger)
	fmt.Println("\n[P2] Scraper.ScrapeDetail(\"3541415\") — Inception subject page")
	detail, err := scraper.ScrapeDetail(ctx, "3541415")
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		fmt.Printf("  Title=%q Year=%d Rating=%.1f Genres=%v Director=%q\n",
			detail.Title, detail.Year, detail.Rating, detail.Genres, detail.Director)
		plot := detail.SummaryTraditional
		if plot == "" {
			plot = detail.Summary
		}
		if len(plot) > 90 {
			plot = plot[:90] + "…"
		}
		fmt.Printf("  Plot: %s\n", plot)
	}

	// Probe 3: review summary (short comments — the 豆瓣短評 block, Epic 12 F-6 path)
	fmt.Println("\n[P3] Scraper.ScrapeReviewSummary(\"3541415\")")
	rev, err := scraper.ScrapeReviewSummary(ctx, "3541415")
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		fmt.Printf("  total=%d top=%d\n", rev.TotalComments, len(rev.TopComments))
		if len(rev.TopComments) > 0 {
			c := rev.TopComments[0].Text
			if len(c) > 60 {
				c = c[:60] + "…"
			}
			fmt.Printf("  first: %s\n", c)
		}
	}

	// Probe 4: SearchByID (detail-as-search fallback path)
	fmt.Println("\n[P4] Searcher.SearchByID(\"3541415\")")
	one, err := searcher.SearchByID(ctx, "3541415")
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		fmt.Printf("  id=%s title=%q year=%d\n", one.ID, one.Title, one.Year)
	}

	m := client.GetMetrics()
	fmt.Printf("\n[metrics] total=%d ok=%d blocked=%d timeout=%d retries=%d\n",
		m.TotalRequests, m.SuccessfulRequests, m.BlockedRequests, m.TimeoutRequests, m.RetryCount)
}
