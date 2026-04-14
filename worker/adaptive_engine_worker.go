package worker

import (
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// AdaptiveEngineCheckInterval is how often the adaptive engine worker runs.
// Defaults to every 6 hours; individual org configs control evaluation cadence.
const AdaptiveEngineCheckInterval = 6 * time.Hour

// StartAdaptiveEngineWorker launches a goroutine that periodically evaluates
// all users across all organizations and adjusts their training difficulty
// based on BRS scores, click-rate trends, quiz performance, and improvement trend.
func StartAdaptiveEngineWorker() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Adaptive Engine Worker: recovered from panic: %v", r)
		}
	}()
	log.Info("Adaptive Engine Worker Started - evaluating every 6 hours")

	// Initial run 5 minutes after startup to let other systems stabilize
	time.AfterFunc(5*time.Minute, func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Adaptive Engine Worker: recovered from panic in initial run: %v", r)
			}
		}()
		log.Info("Adaptive Engine Worker: running initial evaluation cycle")
		processAdaptiveEngineCycle()
	})

	for range time.Tick(AdaptiveEngineCheckInterval) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Adaptive Engine Worker: recovered from panic in cycle: %v", r)
				}
			}()
			processAdaptiveEngineCycle()
		}()
	}
}

// processAdaptiveEngineCycle iterates all organizations and runs the adaptive
// difficulty engine for each.
func processAdaptiveEngineCycle() {
	orgs, err := models.GetOrganizations()
	if err != nil {
		log.Errorf("Adaptive Engine Worker: error fetching organizations: %v", err)
		return
	}

	totalPromoted := 0
	totalDemoted := 0
	totalMaintained := 0

	for _, org := range orgs {
		cfg := models.GetAdaptiveEngineConfig(org.Id)
		if !cfg.Enabled {
			continue
		}

		evaluations, err := models.RunAdaptiveEngine(org.Id)
		if err != nil {
			log.Errorf("Adaptive Engine Worker: org %d (%s) error: %v", org.Id, org.Name, err)
			continue
		}

		// Record the run for audit trail
		models.RecordAdaptiveEngineRun(org.Id, evaluations, 0)

		for _, eval := range evaluations {
			switch eval.Action {
			case "promote":
				totalPromoted++
			case "demote":
				totalDemoted++
			default:
				totalMaintained++
			}
		}

		if len(evaluations) > 0 {
			log.Infof("Adaptive Engine Worker: org %d (%s) — evaluated %d users",
				org.Id, org.Name, len(evaluations))
		}
	}

	if totalPromoted > 0 || totalDemoted > 0 {
		log.Infof("Adaptive Engine Worker: cycle complete — promoted=%d, demoted=%d, maintained=%d",
			totalPromoted, totalDemoted, totalMaintained)
	}
}
