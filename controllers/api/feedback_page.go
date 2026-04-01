package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
)

// FeedbackPages handles requests for the /api/feedback_pages/ endpoint
func (as *Server) FeedbackPages(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		fps, err := models.GetFeedbackPages(getOrgScope(r))
		if err != nil {
			log.Error(err)
		}
		JSONResponse(w, fps, http.StatusOK)
	case r.Method == "POST":
		fp := models.FeedbackPage{}
		err := json.NewDecoder(r.Body).Decode(&fp)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		scope := getOrgScope(r)
		_, err = models.GetFeedbackPageByName(fp.Name, scope)
		if err != gorm.ErrRecordNotFound {
			JSONResponse(w, models.Response{Success: false, Message: "Feedback page name already in use"}, http.StatusConflict)
			return
		}
		fp.UserId = scope.UserId
		fp.OrgId = scope.OrgId
		err = models.PostFeedbackPage(&fp)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, fp, http.StatusCreated)
	}
}

// FeedbackPage handles GET, PUT, DELETE for /api/feedback_pages/{id}
func (as *Server) FeedbackPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	scope := getOrgScope(r)
	fp, err := models.GetFeedbackPage(id, scope)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Feedback page not found"}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		JSONResponse(w, fp, http.StatusOK)
	case r.Method == "DELETE":
		err = models.DeleteFeedbackPage(id, scope)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting feedback page"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Feedback page deleted successfully"}, http.StatusOK)
	case r.Method == "PUT":
		fp = models.FeedbackPage{}
		err = json.NewDecoder(r.Body).Decode(&fp)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		if fp.Id != id {
			JSONResponse(w, models.Response{Success: false, Message: "/:id and feedback page ID mismatch"}, http.StatusBadRequest)
			return
		}
		fp.UserId = scope.UserId
		fp.OrgId = scope.OrgId
		err = models.PutFeedbackPage(&fp)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error updating feedback page: " + err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, fp, http.StatusOK)
	}
}

// FeedbackPageDefault returns the default built-in feedback page HTML
// for the requested language.
func (as *Server) FeedbackPageDefault(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "en"
	}
	html := models.DefaultFeedbackHTML(lang)
	JSONResponse(w, struct {
		HTML     string `json:"html"`
		Language string `json:"language"`
	}{HTML: html, Language: lang}, http.StatusOK)
}
