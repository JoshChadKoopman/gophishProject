package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
)

// Groups returns a list of groups if requested via GET.
// If requested via POST, APIGroups creates a new group and returns a reference to it.
func (as *Server) Groups(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		gs, err := models.GetGroups(getOrgScope(r))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "No groups found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, gs, http.StatusOK)
	//POST: Create a new group and return it as JSON
	case r.Method == "POST":
		g := models.Group{}
		// Put the request into a group
		err := json.NewDecoder(r.Body).Decode(&g)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON structure"}, http.StatusBadRequest)
			return
		}
		scope := getOrgScope(r)
		_, err = models.GetGroupByName(g.Name, scope)
		if err != gorm.ErrRecordNotFound {
			JSONResponse(w, models.Response{Success: false, Message: "Group name already in use"}, http.StatusConflict)
			return
		}
		g.ModifiedDate = time.Now().UTC()
		g.UserId = scope.UserId
		g.OrgId = scope.OrgId
		err = models.PostGroup(&g)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, g, http.StatusCreated)
	}
}

// GroupsSummary returns a summary of the groups owned by the current user.
func (as *Server) GroupsSummary(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		gs, err := models.GetGroupSummaries(getOrgScope(r))
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, gs, http.StatusOK)
	}
}

// Group returns details about the requested group.
// If the group is not valid, Group returns null.
func (as *Server) Group(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	scope := getOrgScope(r)
	g, err := models.GetGroup(id, scope)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Group not found"}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		JSONResponse(w, g, http.StatusOK)
	case r.Method == "DELETE":
		err = models.DeleteGroup(&g)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting group"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Group deleted successfully!"}, http.StatusOK)
	case r.Method == "PUT":
		// Change this to get from URL and uid (don't bother with id in r.Body)
		g = models.Group{}
		err = json.NewDecoder(r.Body).Decode(&g)
		if err != nil {
			log.Errorf("error decoding group: %v", err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		if g.Id != id {
			JSONResponse(w, models.Response{Success: false, Message: "Error: /:id and group_id mismatch"}, http.StatusInternalServerError)
			return
		}
		g.ModifiedDate = time.Now().UTC()
		g.UserId = ctx.Get(r, "user").(models.User).Id
		g.OrgId = scope.OrgId
		err = models.PutGroup(&g)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, g, http.StatusOK)
	}
}

// GroupSummary returns a summary of the groups owned by the current user.
func (as *Server) GroupSummary(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		vars := mux.Vars(r)
		id, _ := strconv.ParseInt(vars["id"], 0, 64)
		g, err := models.GetGroupSummary(id, getOrgScope(r))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Group not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, g, http.StatusOK)
	}
}
