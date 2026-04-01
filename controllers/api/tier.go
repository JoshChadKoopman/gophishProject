package api

import (
	"net/http"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
)

// Tiers returns a list of all available subscription tiers with their features.
func (as *Server) Tiers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tiers, err := models.GetSubscriptionTiers()
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, tiers, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// OrgFeatures returns the feature slugs enabled for the authenticated user's organization.
func (as *Server) OrgFeatures(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		user := ctx.Get(r, "user").(models.User)
		features := models.GetOrgFeatures(user.OrgId)
		JSONResponse(w, features, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
	}
}
