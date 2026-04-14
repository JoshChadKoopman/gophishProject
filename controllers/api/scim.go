package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// scimBaseURL extracts the base URL for building SCIM resource locations.
func scimBaseURL(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// scimOrgID extracts the org ID set by the SCIM auth middleware.
func scimOrgID(r *http.Request) int64 {
	return ctx.Get(r, "scim_org_id").(int64)
}

// scimJSON writes a SCIM JSON response.
func scimJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// scimErrorResponse writes a SCIM error.
func scimErrorResponse(w http.ResponseWriter, status int, detail string) {
	scimJSON(w, models.SCIMErrorResponse{
		Schemas: []string{models.SCIMSchemaError},
		Detail:  detail,
		Status:  strconv.Itoa(status),
	}, status)
}

// --- Service Provider Config ---

// SCIMServiceProviderConfig returns the SCIM service provider configuration (RFC 7643 §5).
func (as *Server) SCIMServiceProviderConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"schemas":            []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"documentationUri":   "https://docs.nivoxis.com/scim",
		"patch":              map[string]bool{"supported": true},
		"bulk":               map[string]interface{}{"supported": false, "maxOperations": 0, "maxPayloadSize": 0},
		"filter":             map[string]interface{}{"supported": true, "maxResults": 200},
		"changePassword":     map[string]bool{"supported": false},
		"sort":               map[string]bool{"supported": false},
		"etag":               map[string]bool{"supported": false},
		"authenticationSchemes": []map[string]string{{
			"type":        "oauthbearertoken",
			"name":        "OAuth Bearer Token",
			"description": "Authentication scheme using the OAuth Bearer Token Standard",
		}},
	}
	scimJSON(w, config, http.StatusOK)
}

// SCIMResourceTypes returns the supported SCIM resource types.
func (as *Server) SCIMResourceTypes(w http.ResponseWriter, r *http.Request) {
	types := []map[string]interface{}{
		{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			"id":          "User",
			"name":        "User",
			"endpoint":    "/scim/v2/Users",
			"schema":      models.SCIMSchemaUser,
			"schemaExtensions": []map[string]interface{}{{
				"schema":   models.SCIMSchemaEnterpriseUser,
				"required": false,
			}},
		},
		{
			"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			"id":       "Group",
			"name":     "Group",
			"endpoint": "/scim/v2/Groups",
			"schema":   models.SCIMSchemaGroup,
		},
	}
	scimJSON(w, types, http.StatusOK)
}

// --- SCIM Users ---

// SCIMUsers handles GET (list) and POST (create) for /scim/v2/Users.
func (as *Server) SCIMUsers(w http.ResponseWriter, r *http.Request) {
	orgId := scimOrgID(r)
	base := scimBaseURL(r)

	switch r.Method {
	case http.MethodGet:
		as.scimListUsers(w, r, orgId, base)
	case http.MethodPost:
		as.scimCreateUser(w, r, orgId, base)
	default:
		scimErrorResponse(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

func (as *Server) scimListUsers(w http.ResponseWriter, r *http.Request, orgId int64, base string) {
	scope := models.OrgScope{OrgId: orgId}
	users, err := models.GetUsersByOrg(scope)
	if err != nil {
		scimErrorResponse(w, http.StatusInternalServerError, "Error fetching users")
		return
	}

	// Basic filter support: filter=userName eq "value"
	if filter := r.URL.Query().Get("filter"); filter != "" {
		users = applyUserFilter(users, filter)
	}

	startIndex := 1
	if v := r.URL.Query().Get("startIndex"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			startIndex = parsed
		}
	}
	count := len(users)
	if v := r.URL.Query().Get("count"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			count = parsed
		}
	}

	// Paginate
	start := startIndex - 1
	if start > len(users) {
		start = len(users)
	}
	end := start + count
	if end > len(users) {
		end = len(users)
	}
	page := users[start:end]

	resources := make([]models.SCIMUserResource, len(page))
	for i, u := range page {
		resources[i] = models.UserToSCIMResource(u, orgId, base)
	}

	scimJSON(w, models.SCIMListResponse{
		Schemas:      []string{models.SCIMSchemaListResponse},
		TotalResults: len(users),
		StartIndex:   startIndex,
		ItemsPerPage: len(resources),
		Resources:    resources,
	}, http.StatusOK)
}

// scimUserRequest represents the inbound SCIM User JSON from an IdP.
type scimUserRequest struct {
	Schemas    []string `json:"schemas"`
	ExternalId string   `json:"externalId"`
	UserName   string   `json:"userName"`
	Name       struct {
		GivenName  string `json:"givenName"`
		FamilyName string `json:"familyName"`
	} `json:"name"`
	Emails []struct {
		Value   string `json:"value"`
		Type    string `json:"type"`
		Primary bool   `json:"primary"`
	} `json:"emails"`
	Active     *bool  `json:"active"`
	Title      string `json:"title"`
	Enterprise *struct {
		Department string `json:"department"`
	} `json:"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"`
}

func (req *scimUserRequest) primaryEmail() string {
	for _, e := range req.Emails {
		if e.Primary {
			return e.Value
		}
	}
	if len(req.Emails) > 0 {
		return req.Emails[0].Value
	}
	return ""
}

func (req *scimUserRequest) department() string {
	if req.Enterprise != nil {
		return req.Enterprise.Department
	}
	return ""
}

func (as *Server) scimCreateUser(w http.ResponseWriter, r *http.Request, orgId int64, base string) {
	var req scimUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		scimErrorResponse(w, http.StatusBadRequest, ErrInvalidJSON)
		return
	}

	email := req.primaryEmail()
	if email == "" {
		scimErrorResponse(w, http.StatusBadRequest, "Email is required")
		return
	}
	username := req.UserName
	if username == "" {
		username = email
	}

	// Check if user already exists by externalId
	if req.ExternalId != "" {
		if existingId, err := models.GetInternalIDByExternalID(orgId, "User", req.ExternalId); err == nil {
			// User already provisioned — return existing
			u, err := models.GetUser(existingId)
			if err == nil {
				res := models.UserToSCIMResource(u, orgId, base)
				scimJSON(w, res, http.StatusOK)
				return
			}
		}
	}

	u, err := models.SCIMProvisionUser(orgId, username, email,
		req.Name.GivenName, req.Name.FamilyName, req.department(), req.Title)
	if err != nil {
		log.Errorf("SCIM create user: %v", err)
		scimErrorResponse(w, http.StatusConflict, fmt.Sprintf("Error creating user: %v", err))
		return
	}

	// Store external ID mapping
	if req.ExternalId != "" {
		models.SetSCIMExternalID(orgId, "User", req.ExternalId, u.Id)
	}

	models.SCIMLog(orgId, "CREATE", "User", strconv.FormatInt(u.Id, 10), u.Email)
	res := models.UserToSCIMResource(u, orgId, base)
	scimJSON(w, res, http.StatusCreated)
}

// SCIMUser handles GET, PUT, PATCH, DELETE for /scim/v2/Users/{id}.
func (as *Server) SCIMUser(w http.ResponseWriter, r *http.Request) {
	orgId := scimOrgID(r)
	base := scimBaseURL(r)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	u, err := models.GetUser(id)
	if err != nil || u.OrgId != orgId {
		scimErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		scimJSON(w, models.UserToSCIMResource(u, orgId, base), http.StatusOK)

	case http.MethodPut:
		var req scimUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			scimErrorResponse(w, http.StatusBadRequest, ErrInvalidJSON)
			return
		}
		email := req.primaryEmail()
		if email == "" {
			email = u.Email
		}
		active := true
		if req.Active != nil {
			active = *req.Active
		}
		if err := models.SCIMUpdateUser(&u, req.Name.GivenName, req.Name.FamilyName,
			email, req.department(), req.Title, active); err != nil {
			scimErrorResponse(w, http.StatusInternalServerError, "Error updating user")
			return
		}
		if req.ExternalId != "" {
			models.SetSCIMExternalID(orgId, "User", req.ExternalId, u.Id)
		}
		models.SCIMLog(orgId, "UPDATE", "User", strconv.FormatInt(u.Id, 10), u.Email)
		scimJSON(w, models.UserToSCIMResource(u, orgId, base), http.StatusOK)

	case http.MethodPatch:
		as.scimPatchUser(w, r, u, orgId, base)

	case http.MethodDelete:
		models.SCIMDeactivateUser(u.Id)
		models.DeleteSCIMExternalID(orgId, "User", u.Id)
		models.SCIMLog(orgId, "DELETE", "User", strconv.FormatInt(u.Id, 10), u.Email)
		w.WriteHeader(http.StatusNoContent)

	default:
		scimErrorResponse(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// scimPatchUser handles SCIM PATCH operations (RFC 7644 §3.5.2).
type scimPatchOp struct {
	Schemas    []string `json:"schemas"`
	Operations []struct {
		Op    string      `json:"op"`
		Path  string      `json:"path"`
		Value interface{} `json:"value"`
	} `json:"Operations"`
}

func (as *Server) scimPatchUser(w http.ResponseWriter, r *http.Request, u models.User, orgId int64, base string) {
	var patch scimPatchOp
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		scimErrorResponse(w, http.StatusBadRequest, ErrInvalidJSON)
		return
	}

	for _, op := range patch.Operations {
		switch strings.ToLower(op.Op) {
		case "replace":
			applyUserPatchReplace(&u, op.Path, op.Value)
		case "add":
			applyUserPatchReplace(&u, op.Path, op.Value)
		}
	}

	if err := models.PutUser(&u); err != nil {
		scimErrorResponse(w, http.StatusInternalServerError, "Error updating user")
		return
	}
	models.SCIMLog(orgId, "PATCH", "User", strconv.FormatInt(u.Id, 10), u.Email)
	scimJSON(w, models.UserToSCIMResource(u, orgId, base), http.StatusOK)
}

func applyUserPatchReplace(u *models.User, path string, value interface{}) {
	switch strings.ToLower(path) {
	case "active":
		if v, ok := value.(bool); ok {
			u.AccountLocked = !v
		}
	case "username", "userName":
		if v, ok := value.(string); ok {
			u.Username = v
		}
	case "name.givenname", "name.givenName":
		if v, ok := value.(string); ok {
			u.FirstName = v
		}
	case "name.familyname", "name.familyName":
		if v, ok := value.(string); ok {
			u.LastName = v
		}
	case "title":
		if v, ok := value.(string); ok {
			u.JobTitle = v
		}
	case "emails[type eq \"work\"].value", "emails":
		if v, ok := value.(string); ok {
			u.Email = v
		}
	case "":
		// No path — value is a map of attributes
		if m, ok := value.(map[string]interface{}); ok {
			if v, ok := m["active"].(bool); ok {
				u.AccountLocked = !v
			}
			if v, ok := m["userName"].(string); ok {
				u.Username = v
			}
		}
	}
}

// --- SCIM Groups ---

// SCIMGroups handles GET (list) and POST (create) for /scim/v2/Groups.
func (as *Server) SCIMGroups(w http.ResponseWriter, r *http.Request) {
	orgId := scimOrgID(r)
	base := scimBaseURL(r)

	switch r.Method {
	case http.MethodGet:
		as.scimListGroups(w, r, orgId, base)
	case http.MethodPost:
		as.scimCreateGroup(w, r, orgId, base)
	default:
		scimErrorResponse(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

func (as *Server) scimListGroups(w http.ResponseWriter, r *http.Request, orgId int64, base string) {
	scope := models.OrgScope{OrgId: orgId}
	groups, err := models.GetGroups(scope)
	if err != nil {
		scimErrorResponse(w, http.StatusInternalServerError, "Error fetching groups")
		return
	}

	if filter := r.URL.Query().Get("filter"); filter != "" {
		groups = applyGroupFilter(groups, filter)
	}

	resources := make([]models.SCIMGroupResource, len(groups))
	for i, g := range groups {
		resources[i] = models.GroupToSCIMResource(g, orgId, base)
	}

	scimJSON(w, models.SCIMListResponse{
		Schemas:      []string{models.SCIMSchemaListResponse},
		TotalResults: len(resources),
		StartIndex:   1,
		ItemsPerPage: len(resources),
		Resources:    resources,
	}, http.StatusOK)
}

type scimGroupRequest struct {
	Schemas     []string `json:"schemas"`
	ExternalId  string   `json:"externalId"`
	DisplayName string   `json:"displayName"`
	Members     []struct {
		Value   string `json:"value"`
		Display string `json:"display"`
	} `json:"members"`
}

func (as *Server) scimCreateGroup(w http.ResponseWriter, r *http.Request, orgId int64, base string) {
	var req scimGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		scimErrorResponse(w, http.StatusBadRequest, ErrInvalidJSON)
		return
	}
	if req.DisplayName == "" {
		scimErrorResponse(w, http.StatusBadRequest, "displayName is required")
		return
	}

	// Check for existing group by externalId
	if req.ExternalId != "" {
		if existingId, err := models.GetInternalIDByExternalID(orgId, "Group", req.ExternalId); err == nil {
			scope := models.OrgScope{OrgId: orgId}
			g, err := models.GetGroup(existingId, scope)
			if err == nil {
				scimJSON(w, models.GroupToSCIMResource(g, orgId, base), http.StatusOK)
				return
			}
		}
	}

	// Build targets from members
	targets := scimMembersToTargets(req.Members, orgId)

	g := models.Group{
		OrgId:   orgId,
		UserId:  0, // SCIM-provisioned, no owner user
		Name:    req.DisplayName,
		Targets: targets,
	}

	if err := models.PostGroup(&g); err != nil {
		log.Errorf("SCIM create group: %v", err)
		scimErrorResponse(w, http.StatusConflict, fmt.Sprintf("Error creating group: %v", err))
		return
	}

	if req.ExternalId != "" {
		models.SetSCIMExternalID(orgId, "Group", req.ExternalId, g.Id)
	}

	models.SCIMLog(orgId, "CREATE", "Group", strconv.FormatInt(g.Id, 10), g.Name)
	// Reload to get targets
	scope := models.OrgScope{OrgId: orgId}
	g, _ = models.GetGroup(g.Id, scope)
	scimJSON(w, models.GroupToSCIMResource(g, orgId, base), http.StatusCreated)
}

// SCIMGroup handles GET, PUT, PATCH, DELETE for /scim/v2/Groups/{id}.
func (as *Server) SCIMGroup(w http.ResponseWriter, r *http.Request) {
	orgId := scimOrgID(r)
	base := scimBaseURL(r)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	scope := models.OrgScope{OrgId: orgId}
	g, err := models.GetGroup(id, scope)
	if err != nil {
		scimErrorResponse(w, http.StatusNotFound, "Group not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		scimJSON(w, models.GroupToSCIMResource(g, orgId, base), http.StatusOK)

	case http.MethodPut:
		var req scimGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			scimErrorResponse(w, http.StatusBadRequest, ErrInvalidJSON)
			return
		}
		if req.DisplayName != "" {
			g.Name = req.DisplayName
		}
		g.Targets = scimMembersToTargets(req.Members, orgId)
		if len(g.Targets) == 0 {
			// SCIM allows empty groups, but GoPhish groups need at least one target.
			// Use a placeholder to keep the group alive.
			g.Targets = []models.Target{{BaseRecipient: models.BaseRecipient{
				Email: fmt.Sprintf("scim-placeholder-%d@placeholder.local", g.Id),
			}}}
		}
		if err := models.PutGroup(&g); err != nil {
			scimErrorResponse(w, http.StatusInternalServerError, "Error updating group")
			return
		}
		if req.ExternalId != "" {
			models.SetSCIMExternalID(orgId, "Group", req.ExternalId, g.Id)
		}
		models.SCIMLog(orgId, "UPDATE", "Group", strconv.FormatInt(g.Id, 10), g.Name)
		g, _ = models.GetGroup(g.Id, scope)
		scimJSON(w, models.GroupToSCIMResource(g, orgId, base), http.StatusOK)

	case http.MethodPatch:
		as.scimPatchGroup(w, r, g, orgId, base)

	case http.MethodDelete:
		models.DeleteGroup(&g)
		models.DeleteSCIMExternalID(orgId, "Group", g.Id)
		models.SCIMLog(orgId, "DELETE", "Group", strconv.FormatInt(g.Id, 10), g.Name)
		w.WriteHeader(http.StatusNoContent)

	default:
		scimErrorResponse(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

func (as *Server) scimPatchGroup(w http.ResponseWriter, r *http.Request, g models.Group, orgId int64, base string) {
	var patch scimPatchOp
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		scimErrorResponse(w, http.StatusBadRequest, ErrInvalidJSON)
		return
	}

	scope := models.OrgScope{OrgId: orgId}

	for _, op := range patch.Operations {
		switch strings.ToLower(op.Op) {
		case "replace":
			if strings.ToLower(op.Path) == "displayname" || strings.ToLower(op.Path) == "displayName" {
				if v, ok := op.Value.(string); ok {
					g.Name = v
				}
			}
			if op.Path == "members" {
				if members, ok := op.Value.([]interface{}); ok {
					g.Targets = scimRawMembersToTargets(members, orgId)
				}
			}
		case "add":
			if strings.ToLower(op.Path) == "members" {
				if members, ok := op.Value.([]interface{}); ok {
					newTargets := scimRawMembersToTargets(members, orgId)
					g.Targets = append(g.Targets, newTargets...)
				}
			}
		case "remove":
			if strings.HasPrefix(strings.ToLower(op.Path), "members") {
				// Remove member by value filter: members[value eq "123"]
				if valStr := extractFilterValue(op.Path); valStr != "" {
					filtered := make([]models.Target, 0, len(g.Targets))
					for _, t := range g.Targets {
						if strconv.FormatInt(t.Id, 10) != valStr {
							filtered = append(filtered, t)
						}
					}
					g.Targets = filtered
				}
			}
		}
	}

	if len(g.Targets) == 0 {
		g.Targets = []models.Target{{BaseRecipient: models.BaseRecipient{
			Email: fmt.Sprintf("scim-placeholder-%d@placeholder.local", g.Id),
		}}}
	}

	if err := models.PutGroup(&g); err != nil {
		scimErrorResponse(w, http.StatusInternalServerError, "Error updating group")
		return
	}
	models.SCIMLog(orgId, "PATCH", "Group", strconv.FormatInt(g.Id, 10), g.Name)
	g, _ = models.GetGroup(g.Id, scope)
	scimJSON(w, models.GroupToSCIMResource(g, orgId, base), http.StatusOK)
}

// --- SCIM Token Management (admin API) ---

// SCIMTokens handles GET/POST for /api/scim/tokens (admin management of SCIM tokens).
func (as *Server) SCIMTokens(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		tokens, err := models.GetSCIMTokens(user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching tokens"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, tokens, http.StatusOK)

	case http.MethodPost:
		var req struct {
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		raw, token, err := models.CreateSCIMToken(user.OrgId, user.Id, req.Description)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error creating token"}, http.StatusInternalServerError)
			return
		}
		// Return the raw token exactly once
		JSONResponse(w, map[string]interface{}{
			"token":  raw,
			"record": token,
		}, http.StatusCreated)
	}
}

// SCIMToken handles DELETE for /api/scim/tokens/{id}.
func (as *Server) SCIMToken(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	if r.Method == http.MethodDelete {
		if err := models.DeleteSCIMToken(id, user.OrgId); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting token"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Token deleted"}, http.StatusOK)
		return
	}
	JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
}

// --- Helpers ---

// applyUserFilter implements basic SCIM filter: userName eq "value" or emails.value eq "value".
func applyUserFilter(users []models.User, filter string) []models.User {
	parts := strings.SplitN(filter, " eq ", 2)
	if len(parts) != 2 {
		return users
	}
	attr := strings.TrimSpace(parts[0])
	val := strings.Trim(strings.TrimSpace(parts[1]), "\"")

	var filtered []models.User
	for _, u := range users {
		switch strings.ToLower(attr) {
		case "username", "userName":
			if strings.EqualFold(u.Username, val) {
				filtered = append(filtered, u)
			}
		case "emails.value", "emails[type eq \"work\"].value":
			if strings.EqualFold(u.Email, val) {
				filtered = append(filtered, u)
			}
		case "externalid", "externalId":
			extId := models.GetSCIMExternalID(u.OrgId, "User", u.Id)
			if extId == val {
				filtered = append(filtered, u)
			}
		}
	}
	return filtered
}

// applyGroupFilter implements basic SCIM filter for groups.
func applyGroupFilter(groups []models.Group, filter string) []models.Group {
	parts := strings.SplitN(filter, " eq ", 2)
	if len(parts) != 2 {
		return groups
	}
	attr := strings.TrimSpace(parts[0])
	val := strings.Trim(strings.TrimSpace(parts[1]), "\"")

	var filtered []models.Group
	for _, g := range groups {
		switch strings.ToLower(attr) {
		case "displayname", "displayName":
			if strings.EqualFold(g.Name, val) {
				filtered = append(filtered, g)
			}
		case "externalid", "externalId":
			extId := models.GetSCIMExternalID(g.OrgId, "Group", g.Id)
			if extId == val {
				filtered = append(filtered, g)
			}
		}
	}
	return filtered
}

// scimMembersToTargets resolves SCIM member references to GoPhish Target records.
func scimMembersToTargets(members []struct {
	Value   string `json:"value"`
	Display string `json:"display"`
}, orgId int64) []models.Target {
	targets := make([]models.Target, 0, len(members))
	for _, m := range members {
		uid, err := strconv.ParseInt(m.Value, 10, 64)
		if err != nil {
			continue
		}
		u, err := models.GetUser(uid)
		if err != nil || u.OrgId != orgId {
			continue
		}
		targets = append(targets, models.Target{
			BaseRecipient: models.BaseRecipient{
				Email:     u.Email,
				FirstName: u.FirstName,
				LastName:  u.LastName,
				Position:  u.JobTitle,
			},
		})
	}
	return targets
}

// scimRawMembersToTargets converts raw JSON member objects to targets.
func scimRawMembersToTargets(members []interface{}, orgId int64) []models.Target {
	targets := make([]models.Target, 0, len(members))
	for _, raw := range members {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		valStr, _ := m["value"].(string)
		uid, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			continue
		}
		u, err := models.GetUser(uid)
		if err != nil || u.OrgId != orgId {
			continue
		}
		targets = append(targets, models.Target{
			BaseRecipient: models.BaseRecipient{
				Email:     u.Email,
				FirstName: u.FirstName,
				LastName:  u.LastName,
				Position:  u.JobTitle,
			},
		})
	}
	return targets
}

// extractFilterValue extracts a value from SCIM path filters like: members[value eq "123"]
func extractFilterValue(path string) string {
	start := strings.Index(path, "\"")
	end := strings.LastIndex(path, "\"")
	if start >= 0 && end > start {
		return path[start+1 : end]
	}
	return ""
}
