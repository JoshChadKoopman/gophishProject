package worker

import (
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// GamificationInterval is how often the gamification worker runs (nightly).
const GamificationInterval = 24 * time.Hour

// StartGamificationWorker launches a goroutine that recalculates leaderboards
// and expires stale streaks on a nightly schedule.
func StartGamificationWorker() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Gamification Worker: recovered from panic: %v", r)
		}
	}()
	log.Info("Gamification Worker Started - running every 24 hours")
	// Run initial calculation 2 minutes after startup
	time.AfterFunc(2*time.Minute, func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Gamification Worker: recovered from panic in initial run: %v", r)
			}
		}()
		log.Info("Gamification Worker: running initial cycle")
		runGamificationCycle()
	})

	for range time.Tick(GamificationInterval) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Gamification Worker: recovered from panic in cycle: %v", r)
				}
			}()
			log.Info("Gamification Worker: starting scheduled cycle")
			runGamificationCycle()
			log.Info("Gamification Worker: cycle complete")
		}()
	}
}

func runGamificationCycle() {
	// Expire stale streaks
	models.ExpireStaleStreaks()
	// Recalculate all leaderboards
	models.RecalculateAllLeaderboards()
}
