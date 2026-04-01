package worker

import (
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// BRSRecalcInterval is how often the BRS background worker runs.
const BRSRecalcInterval = 6 * time.Hour

// StartBRSWorker launches a goroutine that recalculates all BRS scores
// every BRSRecalcInterval. It is called from DefaultWorker.Start().
func StartBRSWorker() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("BRS Worker: recovered from panic: %v", r)
		}
	}()
	log.Info("BRS Worker Started - recalculating every 6 hours")
	// Run an initial calculation shortly after startup
	time.AfterFunc(30*time.Second, func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("BRS Worker: recovered from panic in initial run: %v", r)
			}
		}()
		log.Info("BRS Worker: running initial recalculation")
		if err := models.RecalculateAllBRS(); err != nil {
			log.Errorf("BRS Worker: initial recalculation failed: %v", err)
		}
	})

	for range time.Tick(BRSRecalcInterval) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("BRS Worker: recovered from panic in cycle: %v", r)
				}
			}()
			log.Info("BRS Worker: starting scheduled recalculation")
			if err := models.RecalculateAllBRS(); err != nil {
				log.Errorf("BRS Worker: recalculation failed: %v", err)
			}
			log.Info("BRS Worker: recalculation complete")
		}()
	}
}
