package worker

import (
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// ContentLibraryUpdateInterval is how often the content library updater runs.
// Defaults to once per day.
const ContentLibraryUpdateInterval = 24 * time.Hour

// StartContentLibraryUpdater launches a goroutine that periodically checks
// for new built-in training content and seeds it into organizations that
// have already been initialized with the content library.
//
// This ensures that when the platform is updated with new microlearning
// sessions, compliance modules, or nanolearning tips, all existing organizations
// automatically receive the new content without manual intervention.
func StartContentLibraryUpdater() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Content Library Updater: recovered from panic: %v", r)
		}
	}()
	log.Info("Content Library Updater Started - checking daily for new content")

	// Initial run 10 minutes after startup
	time.AfterFunc(10*time.Minute, func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Content Library Updater: recovered from panic in initial run: %v", r)
			}
		}()
		log.Info("Content Library Updater: running initial content sync")
		processContentLibraryUpdate()
	})

	for range time.Tick(ContentLibraryUpdateInterval) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Content Library Updater: recovered from panic in cycle: %v", r)
				}
			}()
			processContentLibraryUpdate()
		}()
	}
}

// processContentLibraryUpdate iterates all organizations and seeds any new
// built-in content that they don't already have. It detects "initialized" orgs
// by checking whether they have at least one Nivoxis-prefixed presentation.
// Each org's ContentUpdateConfig controls whether auto-updates are enabled.
// All results are logged for audit purposes.
func processContentLibraryUpdate() {
	orgs, err := models.GetOrganizations()
	if err != nil {
		log.Errorf("Content Library Updater: error fetching organizations: %v", err)
		return
	}

	totalOrgsUpdated := 0
	totalNewContent := 0

	for _, org := range orgs {
		// Check if this org has ever been seeded with built-in content
		if !models.OrgHasBuiltInContent(org.Id) {
			continue
		}

		// Check org-level update config
		updateCfg := models.GetContentUpdateConfig(org.Id)
		if !updateCfg.Enabled {
			models.RecordContentUpdate(&models.ContentUpdateLog{
				OrgId:   org.Id,
				OrgName: org.Name,
				Status:  "skipped",
			})
			continue
		}

		// Find a system/admin user for this org to use as the "uploader"
		systemUser := models.GetOrgSystemUser(org.Id)
		if systemUser == 0 {
			log.Errorf("Content Library Updater: org %d has no admin user, skipping", org.Id)
			models.RecordContentUpdate(&models.ContentUpdateLog{
				OrgId:        org.Id,
				OrgName:      org.Name,
				Status:       "error",
				ErrorMessage: "no admin user found for organization",
			})
			continue
		}

		result, err := models.SeedBuiltInContent(org.Id, systemUser)
		if err != nil {
			log.Errorf("Content Library Updater: org %d error: %v", org.Id, err)
			models.RecordContentUpdate(&models.ContentUpdateLog{
				OrgId:        org.Id,
				OrgName:      org.Name,
				Status:       "error",
				ErrorMessage: err.Error(),
			})
			continue
		}

		newItems := result.CoursesCreated + result.SessionsCreated + result.QuizzesCreated
		status := "success"
		if result.CoursesCreated > 0 && result.Skipped > 0 {
			status = "partial"
		}

		models.RecordContentUpdate(&models.ContentUpdateLog{
			OrgId:         org.Id,
			OrgName:       org.Name,
			CoursesAdded:  result.CoursesCreated,
			SessionsAdded: result.SessionsCreated,
			QuizzesAdded:  result.QuizzesCreated,
			Skipped:       result.Skipped,
			Status:        status,
		})

		if newItems > 0 {
			totalOrgsUpdated++
			totalNewContent += result.CoursesCreated
			log.Infof("Content Library Updater: org %d (%s) — seeded %d new courses, %d sessions, %d quizzes",
				org.Id, org.Name, result.CoursesCreated, result.SessionsCreated, result.QuizzesCreated)
		}
	}

	if totalOrgsUpdated > 0 {
		log.Infof("Content Library Updater: cycle complete — updated %d orgs with %d new content items",
			totalOrgsUpdated, totalNewContent)
	}
}
