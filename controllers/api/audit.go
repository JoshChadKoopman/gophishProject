package api

import (
	"net/http"
	"strconv"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// AuditLog handles GET /api/audit-log.
// Returns paginated and optionally filtered audit log entries.
// Supports query params: limit, offset, action, actor, date_from, date_to.
// Requires PermissionViewReports — enforced by route registration in server.go.
func (as *Server) AuditLog(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit := 25
		offset := 0
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				offset = parsed
			}
		}
		action := r.URL.Query().Get("action")
		actor := r.URL.Query().Get("actor")
		dateFrom := r.URL.Query().Get("date_from")
		dateTo := r.URL.Query().Get("date_to")

		resp, err := models.GetAuditLogsFiltered(getOrgScope(r), limit, offset, action, actor, dateFrom, dateTo)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Failed to retrieve audit logs"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, resp, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}
