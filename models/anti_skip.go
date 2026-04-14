package models

import (
	"errors"
	"fmt"
	"time"

	log "github.com/gophish/gophish/logger"
)

// Default anti-skip policy values.
const (
	DefaultMinDwellSeconds   = 10
	DefaultMinScrollDepth    = 80
	DefaultEnforceSequential = true
	DefaultAllowBack         = true
)

// PageEngagement records per-page evidence that a user actually engaged with content.
type PageEngagement struct {
	Id              int64     `json:"id" gorm:"primary_key"`
	UserId          int64     `json:"user_id" gorm:"column:user_id"`
	PresentationId  int64     `json:"presentation_id" gorm:"column:presentation_id"`
	PageIndex       int       `json:"page_index" gorm:"column:page_index"`
	EnteredAt       time.Time `json:"entered_at" gorm:"column:entered_at"`
	DwellSeconds    int       `json:"dwell_seconds" gorm:"column:dwell_seconds"`
	ScrollDepthPct  int       `json:"scroll_depth_pct" gorm:"column:scroll_depth_pct"`
	InteractionType string    `json:"interaction_type" gorm:"column:interaction_type"`
	Acknowledged    bool      `json:"acknowledged" gorm:"column:acknowledged"`
	CreatedDate     time.Time `json:"created_date" gorm:"column:created_date"`
}

// TableName overrides the default GORM table name.
func (PageEngagement) TableName() string {
	return "page_engagement"
}

// AntiSkipPolicy defines per-presentation rules for anti-skip enforcement.
type AntiSkipPolicy struct {
	Id                  int64     `json:"id" gorm:"primary_key"`
	PresentationId      int64     `json:"presentation_id" gorm:"column:presentation_id"`
	MinDwellSeconds     int       `json:"min_dwell_seconds" gorm:"column:min_dwell_seconds"`
	RequireAcknowledge  bool      `json:"require_acknowledge" gorm:"column:require_acknowledge"`
	RequireScroll       bool      `json:"require_scroll" gorm:"column:require_scroll"`
	MinScrollDepthPct   int       `json:"min_scroll_depth_pct" gorm:"column:min_scroll_depth_pct"`
	EnforceSequential   bool      `json:"enforce_sequential" gorm:"column:enforce_sequential"`
	AllowBackNavigation bool      `json:"allow_back_navigation" gorm:"column:allow_back_navigation"`
	CreatedDate         time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate        time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// TableName overrides the default GORM table name.
func (AntiSkipPolicy) TableName() string {
	return "anti_skip_policy"
}

// PageAdvanceResult is the server response when a user requests to advance a page.
type PageAdvanceResult struct {
	Allowed       bool   `json:"allowed"`
	Reason        string `json:"reason,omitempty"`
	NextPage      int    `json:"next_page"`
	MinDwell      int    `json:"min_dwell_seconds"`
	RequireAck    bool   `json:"require_acknowledge"`
	RequireScroll bool   `json:"require_scroll"`
	MinScroll     int    `json:"min_scroll_depth_pct"`
	PagesUnlocked int    `json:"pages_unlocked"` // highest page index the user may visit
}

// PageEngagementUpdate is the client payload when reporting engagement on a page.
type PageEngagementUpdate struct {
	PageIndex      int  `json:"page_index"`
	DwellSeconds   int  `json:"dwell_seconds"`
	ScrollDepthPct int  `json:"scroll_depth_pct"`
	Acknowledged   bool `json:"acknowledged"`
}

// CompletionGateResult is the response for "can I complete this course?"
type CompletionGateResult struct {
	Allowed      bool   `json:"allowed"`
	Reason       string `json:"reason,omitempty"`
	MissingPages []int  `json:"missing_pages,omitempty"` // pages with insufficient engagement
	TotalPages   int    `json:"total_pages"`
	EngagedPages int    `json:"engaged_pages"`
}

// Shared WHERE clause for presentation_id lookups in anti-skip tables.
const queryWherePresentationIDAntiSkip = "presentation_id = ?"

// Errors
var (
	ErrPageSkipped       = errors.New("You must complete the current page before advancing")
	ErrInsufficientDwell = errors.New("Please spend more time reading this page before continuing")
	ErrAckRequired       = errors.New("Please confirm you have read this page")
	ErrScrollRequired    = errors.New("Please scroll through the entire page content")
	ErrNotAllPagesViewed = errors.New("You must view all pages before completing the course")
)

// GetAntiSkipPolicy returns the policy for a presentation, or defaults if none exists.
func GetAntiSkipPolicy(presentationId int64) AntiSkipPolicy {
	policy := AntiSkipPolicy{}
	err := db.Where(queryWherePresentationIDAntiSkip, presentationId).First(&policy).Error
	if err != nil {
		return defaultAntiSkipPolicy(presentationId)
	}
	return policy
}

func defaultAntiSkipPolicy(presentationId int64) AntiSkipPolicy {
	return AntiSkipPolicy{
		PresentationId:      presentationId,
		MinDwellSeconds:     DefaultMinDwellSeconds,
		RequireAcknowledge:  false,
		RequireScroll:       false,
		MinScrollDepthPct:   DefaultMinScrollDepth,
		EnforceSequential:   DefaultEnforceSequential,
		AllowBackNavigation: DefaultAllowBack,
	}
}

// SaveAntiSkipPolicy creates or updates a policy for a presentation.
func SaveAntiSkipPolicy(policy *AntiSkipPolicy) error {
	if policy.MinDwellSeconds < 0 {
		policy.MinDwellSeconds = 0
	}
	if policy.MinScrollDepthPct < 0 || policy.MinScrollDepthPct > 100 {
		policy.MinScrollDepthPct = DefaultMinScrollDepth
	}
	now := time.Now().UTC()
	policy.ModifiedDate = now
	existing := AntiSkipPolicy{}
	err := db.Where(queryWherePresentationIDAntiSkip, policy.PresentationId).First(&existing).Error
	if err == nil {
		policy.Id = existing.Id
		policy.CreatedDate = existing.CreatedDate
	} else {
		policy.CreatedDate = now
	}
	return db.Save(policy).Error
}

// DeleteAntiSkipPolicy removes a custom policy, reverting to defaults.
func DeleteAntiSkipPolicy(presentationId int64) error {
	return db.Where(queryWherePresentationIDAntiSkip, presentationId).Delete(&AntiSkipPolicy{}).Error
}

// RecordPageEngagement upserts engagement data for a user on a specific page.
func RecordPageEngagement(userId, presentationId int64, update PageEngagementUpdate) error {
	eng := PageEngagement{}
	err := db.Where("user_id = ? AND presentation_id = ? AND page_index = ?",
		userId, presentationId, update.PageIndex).First(&eng).Error

	now := time.Now().UTC()
	if err != nil {
		// New record
		eng = PageEngagement{
			UserId:         userId,
			PresentationId: presentationId,
			PageIndex:      update.PageIndex,
			EnteredAt:      now,
			CreatedDate:    now,
		}
	}

	// Accumulate dwell time (don't replace — add to existing)
	eng.DwellSeconds += update.DwellSeconds

	// Track max scroll depth
	if update.ScrollDepthPct > eng.ScrollDepthPct {
		eng.ScrollDepthPct = update.ScrollDepthPct
	}

	// Once acknowledged, stays acknowledged
	if update.Acknowledged {
		eng.Acknowledged = true
		eng.InteractionType = "acknowledge"
	}
	if eng.InteractionType == "" {
		eng.InteractionType = "timer"
	}

	return db.Save(&eng).Error
}

// GetPageEngagements returns all engagement records for a user on a presentation.
func GetPageEngagements(userId, presentationId int64) ([]PageEngagement, error) {
	var records []PageEngagement
	err := db.Where("user_id = ? AND presentation_id = ?", userId, presentationId).
		Order("page_index asc").Find(&records).Error
	return records, err
}

// ValidatePageAdvance checks whether a user may advance from currentPage to nextPage.
func ValidatePageAdvance(userId, presentationId int64, currentPage, nextPage, totalPages int) PageAdvanceResult {
	policy := GetAntiSkipPolicy(presentationId)

	result := PageAdvanceResult{
		MinDwell:      policy.MinDwellSeconds,
		RequireAck:    policy.RequireAcknowledge,
		RequireScroll: policy.RequireScroll,
		MinScroll:     policy.MinScrollDepthPct,
	}

	// Calculate the highest unlocked page
	result.PagesUnlocked = getHighestUnlockedPage(userId, presentationId, totalPages, policy)
	result.NextPage = nextPage

	// Allow backward navigation (always permitted if policy allows)
	if nextPage <= currentPage && policy.AllowBackNavigation {
		result.Allowed = true
		return result
	}

	// Sequential enforcement: can only go to the next page
	if policy.EnforceSequential && nextPage > currentPage+1 {
		result.Allowed = false
		result.Reason = ErrPageSkipped.Error()
		return result
	}

	// Check engagement on the current page before allowing advance
	reason := checkPageEngagement(userId, presentationId, currentPage, policy)
	if reason != "" {
		result.Allowed = false
		result.Reason = reason
		return result
	}

	result.Allowed = true
	return result
}

// ValidateCourseCompletion checks whether a user has sufficient engagement on ALL pages.
func ValidateCourseCompletion(userId, presentationId int64, totalPages int) CompletionGateResult {
	policy := GetAntiSkipPolicy(presentationId)

	result := CompletionGateResult{
		TotalPages: totalPages,
	}

	engagements, err := GetPageEngagements(userId, presentationId)
	if err != nil {
		result.Allowed = false
		result.Reason = "Could not verify page engagement"
		return result
	}

	engMap := make(map[int]PageEngagement, len(engagements))
	for _, e := range engagements {
		engMap[e.PageIndex] = e
	}

	for i := 0; i < totalPages; i++ {
		eng, exists := engMap[i]
		if !exists {
			result.MissingPages = append(result.MissingPages, i)
			continue
		}
		if reason := checkEngagementRecord(eng, policy); reason != "" {
			result.MissingPages = append(result.MissingPages, i)
		} else {
			result.EngagedPages++
		}
	}

	if len(result.MissingPages) > 0 {
		result.Allowed = false
		result.Reason = fmt.Sprintf("Insufficient engagement on %d of %d pages", len(result.MissingPages), totalPages)
	} else {
		result.Allowed = true
	}

	return result
}

// -- Internal helpers --------------------------------------------------------

// getHighestUnlockedPage determines the highest page index a user may visit.
func getHighestUnlockedPage(userId, presentationId int64, totalPages int, policy AntiSkipPolicy) int {
	if !policy.EnforceSequential {
		return totalPages - 1 // all pages unlocked
	}

	engagements, err := GetPageEngagements(userId, presentationId)
	if err != nil || len(engagements) == 0 {
		return 0 // only first page unlocked
	}

	highest := 0
	for _, eng := range engagements {
		if checkEngagementRecord(eng, policy) == "" && eng.PageIndex >= highest {
			highest = eng.PageIndex + 1
		}
	}

	if highest >= totalPages {
		highest = totalPages - 1
	}
	return highest
}

// checkPageEngagement verifies that a specific page has been sufficiently engaged.
func checkPageEngagement(userId, presentationId int64, pageIndex int, policy AntiSkipPolicy) string {
	eng := PageEngagement{}
	err := db.Where("user_id = ? AND presentation_id = ? AND page_index = ?",
		userId, presentationId, pageIndex).First(&eng).Error
	if err != nil {
		return ErrInsufficientDwell.Error()
	}
	return checkEngagementRecord(eng, policy)
}

// checkEngagementRecord validates a single engagement record against policy.
func checkEngagementRecord(eng PageEngagement, policy AntiSkipPolicy) string {
	if policy.MinDwellSeconds > 0 && eng.DwellSeconds < policy.MinDwellSeconds {
		return fmt.Sprintf("Please spend at least %d seconds on this page (%d so far)", policy.MinDwellSeconds, eng.DwellSeconds)
	}
	if policy.RequireAcknowledge && !eng.Acknowledged {
		return ErrAckRequired.Error()
	}
	if policy.RequireScroll && eng.ScrollDepthPct < policy.MinScrollDepthPct {
		return fmt.Sprintf("Please scroll to at least %d%% of the page (%d%% so far)", policy.MinScrollDepthPct, eng.ScrollDepthPct)
	}
	return ""
}

// ResetPageEngagement clears all engagement records for a user on a presentation.
// Used when a user restarts a course.
func ResetPageEngagement(userId, presentationId int64) error {
	return db.Where("user_id = ? AND presentation_id = ?", userId, presentationId).
		Delete(&PageEngagement{}).Error
}

// GetEngagementSummary returns a summary of engagement for admin/report views.
func GetEngagementSummary(presentationId int64) ([]EngagementSummaryRow, error) {
	var rows []EngagementSummaryRow
	err := db.Raw(`
		SELECT u.id as user_id, u.username, u.email,
			COUNT(pe.id) as pages_engaged,
			COALESCE(SUM(pe.dwell_seconds), 0) as total_dwell_seconds,
			COALESCE(AVG(pe.dwell_seconds), 0) as avg_dwell_seconds,
			COALESCE(AVG(pe.scroll_depth_pct), 0) as avg_scroll_depth
		FROM users u
		LEFT JOIN page_engagement pe ON u.id = pe.user_id AND pe.presentation_id = ?
		JOIN course_progress cp ON u.id = cp.user_id AND cp.presentation_id = ?
		GROUP BY u.id
		ORDER BY total_dwell_seconds DESC
	`, presentationId, presentationId).Scan(&rows).Error
	if err != nil {
		log.Errorf("anti_skip: engagement summary error: %v", err)
	}
	return rows, err
}

// EngagementSummaryRow is a single row in the admin engagement summary.
type EngagementSummaryRow struct {
	UserId            int64   `json:"user_id"`
	Username          string  `json:"username"`
	Email             string  `json:"email"`
	PagesEngaged      int     `json:"pages_engaged"`
	TotalDwellSeconds int     `json:"total_dwell_seconds"`
	AvgDwellSeconds   float64 `json:"avg_dwell_seconds"`
	AvgScrollDepth    float64 `json:"avg_scroll_depth"`
}
