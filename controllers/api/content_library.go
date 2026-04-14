package api

import (
	"encoding/json"
	"net/http"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// contentSummary is a lightweight representation of built-in content for listing.
type contentSummary struct {
	Slug             string   `json:"slug"`
	Title            string   `json:"title"`
	Category         string   `json:"category"`
	DifficultyLevel  int      `json:"difficulty_level"`
	Description      string   `json:"description"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	Tags             []string `json:"tags"`
	ComplianceMapped []string `json:"compliance_mapped"`
	PageCount        int      `json:"page_count"`
	HasQuiz          bool     `json:"has_quiz"`
	QuestionCount    int      `json:"question_count"`
	NanolearningTip  string   `json:"nanolearning_tip"`
}

// filterByCategory returns only items matching the given category.
func filterByCategory(library []models.BuiltInTrainingContent, category string) []models.BuiltInTrainingContent {
	filtered := []models.BuiltInTrainingContent{}
	for _, c := range library {
		if c.Category == category {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// parseDifficultyLevel converts a difficulty query string to an int (0 if invalid).
func parseDifficultyLevel(difficulty string) int {
	switch difficulty {
	case "1":
		return 1
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	default:
		return 0
	}
}

// filterByDifficulty returns only items matching the given difficulty level.
func filterByDifficulty(library []models.BuiltInTrainingContent, level int) []models.BuiltInTrainingContent {
	filtered := []models.BuiltInTrainingContent{}
	for _, c := range library {
		if c.DifficultyLevel == level {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// buildContentSummaries converts full content items to lightweight summaries.
func buildContentSummaries(library []models.BuiltInTrainingContent) []contentSummary {
	summaries := make([]contentSummary, len(library))
	for i, c := range library {
		qCount := 0
		if c.Quiz != nil {
			qCount = len(c.Quiz.Questions)
		}
		summaries[i] = contentSummary{
			Slug:             c.Slug,
			Title:            c.Title,
			Category:         c.Category,
			DifficultyLevel:  c.DifficultyLevel,
			Description:      c.Description,
			EstimatedMinutes: c.EstimatedMinutes,
			Tags:             c.Tags,
			ComplianceMapped: c.ComplianceMapped,
			PageCount:        len(c.Pages),
			HasQuiz:          c.Quiz != nil,
			QuestionCount:    qCount,
			NanolearningTip:  c.NanolearningTip,
		}
	}
	return summaries
}

// ContentLibrary handles GET /api/training/content-library — browse built-in content.
// Supports query params: ?category=phishing&difficulty=1
func (as *Server) ContentLibrary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	library := models.GetBuiltInContentLibrary()

	// Optional filters
	if category := r.URL.Query().Get("category"); category != "" {
		library = filterByCategory(library, category)
	}

	if difficulty := r.URL.Query().Get("difficulty"); difficulty != "" {
		if level := parseDifficultyLevel(difficulty); level > 0 {
			library = filterByDifficulty(library, level)
		}
	}

	JSONResponse(w, buildContentSummaries(library), http.StatusOK)
}

// ContentLibraryDetail handles GET /api/training/content-library/{slug} — full content detail.
func (as *Server) ContentLibraryDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		// Try path parameter
		parts := splitPath(r.URL.Path)
		if len(parts) > 0 {
			slug = parts[len(parts)-1]
		}
	}

	content := models.GetBuiltInContentBySlug(slug)
	if content == nil {
		JSONResponse(w, models.Response{Success: false, Message: "Content not found"}, http.StatusNotFound)
		return
	}

	JSONResponse(w, content, http.StatusOK)
}

// ContentLibraryCategories handles GET /api/training/content-library/categories.
func (as *Server) ContentLibraryCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	categories := models.GetContentCategories()
	JSONResponse(w, categories, http.StatusOK)
}

// ContentLibrarySeed handles POST /api/training/content-library/seed — seed all built-in content into org.
func (as *Server) ContentLibrarySeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: "Only administrators can seed content"}, http.StatusForbidden)
		return
	}

	result, err := models.SeedBuiltInContent(user.OrgId, user.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to seed content: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, result, http.StatusOK)
}

// ContentLibrarySeedSingle handles POST /api/training/content-library/seed/{slug} — seed a single item.
func (as *Server) ContentLibrarySeedSingle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: "Only administrators can seed content"}, http.StatusForbidden)
		return
	}

	// Parse slug from request body
	var req struct {
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Slug == "" {
		JSONResponse(w, models.Response{Success: false, Message: "slug is required"}, http.StatusBadRequest)
		return
	}

	content := models.GetBuiltInContentBySlug(req.Slug)
	if content == nil {
		JSONResponse(w, models.Response{Success: false, Message: "Content not found: " + req.Slug}, http.StatusNotFound)
		return
	}

	// Seed just this one — use the full seed but it'll skip already-existing ones
	result, err := models.SeedBuiltInContent(user.OrgId, user.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Failed to seed content"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, result, http.StatusOK)
}

// splitPath is a helper to split URL paths.
func splitPath(path string) []string {
	parts := []string{}
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
