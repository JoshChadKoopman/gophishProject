package api

import (
	"net/http"
	"strconv"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ── A/B Test API ────────────────────────────────────────────────
// Provides post-hoc analysis of A/B template variant tests per campaign.

// ABTestSummary handles GET /api/ab-test/{campaignId}
// Returns aggregated results for variant A vs B in the given campaign.
func (as *Server) ABTestSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	campaignId, _ := strconv.ParseInt(vars["campaignId"], 10, 64)
	if campaignId == 0 {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid campaign ID"}, http.StatusBadRequest)
		return
	}

	summaries, err := models.GetABTestSummary(campaignId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error retrieving A/B test results"}, http.StatusInternalServerError)
		return
	}

	// Compute the winner
	winner := computeABWinner(summaries)

	JSONResponse(w, map[string]interface{}{
		"campaign_id": campaignId,
		"variants":    summaries,
		"winner":      winner,
	}, http.StatusOK)
}

// abWinnerResult describes which variant performed better.
type abWinnerResult struct {
	VariantId      string  `json:"variant_id"`
	Reason         string  `json:"reason"`
	ClickRateDelta float64 `json:"click_rate_delta"`
	Significant    bool    `json:"statistically_significant"`
}

// computeABWinner determines which variant performed better based on
// click rate (lower = better for phishing simulations, indicates users
// detected the phish), and report rate (higher = better).
func computeABWinner(summaries []models.ABTestVariantSummary) abWinnerResult {
	if len(summaries) < 2 {
		return abWinnerResult{Reason: "Insufficient variants for comparison"}
	}

	a := summaries[0]
	b := summaries[1]

	// For phishing simulations: lower click rate = better (users detected the phish)
	// But from a training perspective: we want the variant that best challenges users
	// So the "winner" is the one with more learning value — lower click rate means
	// it was easier to detect, higher click rate means it was more effective as training.

	// Report rate is the key metric: higher report rate = better learning outcome
	aReportRate := 0.0
	bReportRate := 0.0
	if a.Total > 0 {
		aReportRate = float64(a.Reported) / float64(a.Total) * 100
	}
	if b.Total > 0 {
		bReportRate = float64(b.Reported) / float64(b.Total) * 100
	}

	delta := aReportRate - bReportRate
	result := abWinnerResult{
		ClickRateDelta: a.ClickRate - b.ClickRate,
	}

	// Statistical significance heuristic: need at least 30 per variant and >5% delta
	result.Significant = a.Total >= 30 && b.Total >= 30 && (delta > 5 || delta < -5)

	if delta > 0 {
		result.VariantId = a.VariantId
		result.Reason = "Variant " + a.VariantId + " had a higher report rate (" +
			strconv.FormatFloat(aReportRate, 'f', 1, 64) + "% vs " +
			strconv.FormatFloat(bReportRate, 'f', 1, 64) + "%)"
	} else if delta < 0 {
		result.VariantId = b.VariantId
		result.Reason = "Variant " + b.VariantId + " had a higher report rate (" +
			strconv.FormatFloat(bReportRate, 'f', 1, 64) + "% vs " +
			strconv.FormatFloat(aReportRate, 'f', 1, 64) + "%)"
	} else {
		result.Reason = "No meaningful difference between variants"
	}

	return result
}
