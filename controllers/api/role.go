package api

import (
	"net/http"

	"github.com/gophish/gophish/models"
)

// Roles returns a list of all available roles in the system.
func (as *Server) Roles(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		roles, err := models.GetRoles()
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, roles, http.StatusOK)
	}
}
