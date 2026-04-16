package worker

import (
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// ReportRollupInterval is how often the reporting rollup worker runs.
// Defaults to every 4 hours, with the main full-day rollup happening nightly.
const ReportRollupInterval = 4 * time.Hour

// ReportRollupBackfillDays controls how many days to backfill on first startup.
const ReportRollupBackfillDays = 90

// StartReportRollupWorker launches a goroutine that pre-computes daily
// reporting metrics into the report_daily_metrics table.
//
// This replaces expensive real-time queries on campaign/result/ticket tables
// with fast lookups against a materialized rollup table. The pattern is
// similar to BRSRecalcWorker.
//
// On first startup (or if the table is empty), it backfills the last 90 days.
// Then it runs every 4 hours, always refreshing today's and yesterday's data
// (yesterday may have late-arriving events).
func StartReportRollupWorker() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("ReportRollup Worker: recovered from panic: %v", r)
		}
	}()
	log.Info("ReportRollup Worker Started — rollups every 4 hours, backfill on first run")

	// Initial backfill after a short delay to let DB migrations complete
	time.AfterFunc(45*time.Second, func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("ReportRollup Worker: recovered from panic in initial run: %v", r)
			}
		}()

		// Check if we need a full backfill
		if !models.HasRollupData(0) {
			log.Infof("ReportRollup Worker: no rollup data found — backfilling last %d days", ReportRollupBackfillDays)
			if err := models.BackfillDailyMetrics(ReportRollupBackfillDays); err != nil {
				log.Errorf("ReportRollup Worker: backfill error: %v", err)
			}
			log.Info("ReportRollup Worker: backfill complete")
		} else {
			// Just refresh today + yesterday
			log.Info("ReportRollup Worker: running initial rollup for today + yesterday")
			rollupRecentDays()
		}
	})

	// Periodic rollup
	for range time.Tick(ReportRollupInterval) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("ReportRollup Worker: recovered from panic in cycle: %v", r)
				}
			}()
			log.Info("ReportRollup Worker: starting scheduled rollup")
			rollupRecentDays()
			log.Info("ReportRollup Worker: rollup cycle complete")
		}()
	}
}

// rollupRecentDays refreshes today's and yesterday's rollup rows for all orgs.
// Yesterday is always re-computed because late-arriving events (e.g., email
// open tracking, delayed IMAP scans) may update counts after midnight.
func rollupRecentDays() {
	now := time.Now().UTC()
	yesterday := now.AddDate(0, 0, -1)

	if err := models.ComputeAllOrgsDailyMetrics(yesterday); err != nil {
		log.Errorf("ReportRollup Worker: error computing yesterday (%s): %v",
			yesterday.Format("2006-01-02"), err)
	}
	if err := models.ComputeAllOrgsDailyMetrics(now); err != nil {
		log.Errorf("ReportRollup Worker: error computing today (%s): %v",
			now.Format("2006-01-02"), err)
	}
}
