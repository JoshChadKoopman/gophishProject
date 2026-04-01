package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// Orgs handles GET /api/orgs/ (list all) and POST /api/orgs/ (create).
// Requires modify_system permission (superadmin).
func (as *Server) Orgs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		orgs, err := models.GetOrganizations()
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, orgs, http.StatusOK)
	case http.MethodPost:
		o := models.Organization{}
		err := json.NewDecoder(r.Body).Decode(&o)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON"}, http.StatusBadRequest)
			return
		}
		err = models.PostOrganization(&o)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, o, http.StatusCreated)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// Org handles GET/PUT/DELETE /api/orgs/{id}.
// GET: superadmin can view any org; org_admin can view their own.
// PUT: superadmin or org_admin of that org.
// DELETE: superadmin only.
func (as *Server) Org(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)

	o, err := models.GetOrganization(id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Organization not found"}, http.StatusNotFound)
		return
	}

	// Non-superadmins can only access their own org
	user := ctx.Get(r, "user").(models.User)
	isSuperAdmin := user.Role.Slug == models.RoleSuperAdmin
	if !isSuperAdmin && user.OrgId != id {
		JSONResponse(w, models.Response{Success: false, Message: http.StatusText(http.StatusForbidden)}, http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		JSONResponse(w, o, http.StatusOK)
	case http.MethodPut:
		update := models.Organization{}
		err := json.NewDecoder(r.Body).Decode(&update)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON"}, http.StatusBadRequest)
			return
		}
		// Preserve ID and created date
		update.Id = o.Id
		update.CreatedDate = o.CreatedDate
		// Only superadmins can change tier/quota fields
		if !isSuperAdmin {
			update.Tier = o.Tier
			update.MaxUsers = o.MaxUsers
			update.MaxCampaigns = o.MaxCampaigns
		}
		err = models.PutOrganization(&update)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, update, http.StatusOK)
	case http.MethodDelete:
		if !isSuperAdmin {
			JSONResponse(w, models.Response{Success: false, Message: "Only superadmins can delete organizations"}, http.StatusForbidden)
			return
		}
		if id == 1 {
			JSONResponse(w, models.Response{Success: false, Message: "Cannot delete the default organization"}, http.StatusBadRequest)
			return
		}
		err = models.DeleteOrganization(id)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Organization deleted successfully"}, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// OrgMembers handles GET /api/orgs/{id}/members (list) and POST (add member).
func (as *Server) OrgMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)

	_, err := models.GetOrganization(id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Organization not found"}, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		scope := models.OrgScope{OrgId: id, IsSuperAdmin: false}
		users, err := models.GetUsersByOrg(scope)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, users, http.StatusOK)
	case http.MethodPost:
		var req struct {
			UserId int64 `json:"user_id"`
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON"}, http.StatusBadRequest)
			return
		}
		user, err := models.GetUser(req.UserId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "User not found"}, http.StatusNotFound)
			return
		}
		user.OrgId = id
		err = models.PutUser(&user)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "User added to organization"}, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// OrgMember handles DELETE /api/orgs/{id}/members/{uid} to remove a member.
func (as *Server) OrgMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgId, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid, _ := strconv.ParseInt(vars["uid"], 0, 64)

	if r.Method != http.MethodDelete {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	user, err := models.GetUser(uid)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "User not found"}, http.StatusNotFound)
		return
	}
	if user.OrgId != orgId {
		JSONResponse(w, models.Response{Success: false, Message: "User is not in this organization"}, http.StatusBadRequest)
		return
	}
	// Move user to the default org (id=1)
	user.OrgId = 1
	err = models.PutUser(&user)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "User removed from organization"}, http.StatusOK)
}
