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

// SMSProviders handles requests for the /api/sms/ endpoint
func (as *Server) SMSProviders(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		ps, err := models.GetSMSProviders(getOrgScope(r))
		if err != nil {
			log.Error(err)
		}
		JSONResponse(w, ps, http.StatusOK)
	case r.Method == "POST":
		sp := models.SMSProvider{}
		err := json.NewDecoder(r.Body).Decode(&sp)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		scope := getOrgScope(r)
		_, err = models.GetSMSProviderByName(sp.Name, scope)
		if err != gorm.ErrRecordNotFound {
			JSONResponse(w, models.Response{Success: false, Message: "SMS provider name already in use"}, http.StatusConflict)
			return
		}
		err = models.PostSMSProvider(&sp, scope)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, sp, http.StatusCreated)
	}
}

// SMSProvider handles requests for the /api/sms/:id endpoint
func (as *Server) SMSProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	scope := getOrgScope(r)
	sp, err := models.GetSMSProvider(id, scope)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "SMS provider not found"}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		JSONResponse(w, sp, http.StatusOK)
	case r.Method == "DELETE":
		err = models.DeleteSMSProvider(id, scope)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting SMS provider"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "SMS provider deleted successfully"}, http.StatusOK)
	case r.Method == "PUT":
		sp = models.SMSProvider{}
		err = json.NewDecoder(r.Body).Decode(&sp)
		if err != nil {
			log.Error(err)
		}
		if sp.Id != id {
			JSONResponse(w, models.Response{Success: false, Message: "/:id and /:sms_id mismatch"}, http.StatusBadRequest)
			return
		}
		err = models.PutSMSProvider(&sp, scope)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, sp, http.StatusOK)
	}
}
