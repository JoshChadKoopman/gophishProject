package api

import (
	"net/http"
	"strconv"

	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// TargetingProfile handles GET /api/targeting/profile/{id}
// Returns the adaptive targeting profile for a user, including recommended
// difficulty, weak/strong categories, and trend direction.
func (as *Server) TargetingProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	uid, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid user ID"}, http.StatusBadRequest)
		return
	}
	profile, err := models.GetUserTargetingProfile(uid)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Could not build targeting profile"}, http.StatusNotFound)
		return
	}
	JSONResponse(w, profile, http.StatusOK)
}
