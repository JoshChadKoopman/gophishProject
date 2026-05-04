package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// errAssetNotFound is the shared 404 message for training asset lookups.
const errAssetNotFound = "Training asset not found"

// TrainingAssets handles GET (list) and POST (upload) for the custom training
// builder's asset collection under a specific presentation.
//
// GET:  any authenticated user with access to the course (org-scoped read)
// POST: requires PermissionManageTraining AND FeatureCustomTrainingBuilder
//       (enforced via middleware on the route definition)
func (as *Server) TrainingAssets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	presId, _ := strconv.ParseInt(vars["id"], 0, 64)

	// Make sure the presentation exists and is visible to this org.
	if _, err := models.GetTrainingPresentation(presId, getOrgScope(r)); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errTrainingNotFound}, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		assets, err := models.GetTrainingAssets(presId, getOrgScope(r))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, assets, http.StatusOK)
	case http.MethodPost:
		as.handleTrainingAssetUpload(w, r, presId)
	}
}

func (as *Server) handleTrainingAssetUpload(w http.ResponseWriter, r *http.Request, presId int64) {
	user := ctx.Get(r, "user").(models.User)
	hasAdmin, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasAdmin {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "File too large. Maximum size is 50MB."}, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "File is required"}, http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get(headerContentType)
	if !allowedTrainingTypes[contentType] && !allowedThumbnailTypes[contentType] {
		JSONResponse(w, models.Response{Success: false, Message: "File type not allowed. Upload PDF, PowerPoint, ODP, video, or image files."}, http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(trainingUploadDir, 0755); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error creating upload directory"}, http.StatusInternalServerError)
		return
	}

	filePath, written, err := saveUploadedFile(file, header.Filename)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error saving file"}, http.StatusInternalServerError)
		return
	}

	title := r.FormValue("title")
	if title == "" {
		title = header.Filename
	}
	asset := &models.TrainingAsset{
		PresentationId: presId,
		OrgId:          user.OrgId,
		Title:          title,
		Description:    r.FormValue("description"),
		FileName:       header.Filename,
		FilePath:       filePath,
		FileSize:       written,
		ContentType:    contentType,
		AssetType:      models.ClassifyAssetType(contentType),
		UploadedBy:     user.Id,
	}

	if err := models.PostTrainingAsset(asset); err != nil {
		os.Remove(filePath)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, asset, http.StatusCreated)
}

// TrainingAsset handles PUT and DELETE for a single training asset.
// PUT:    requires PermissionManageTraining — metadata updates (title/description)
// DELETE: requires PermissionManageTraining — removes row and file blob
func (as *Server) TrainingAsset(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	asset, err := models.GetTrainingAsset(id, getOrgScope(r))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errAssetNotFound}, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		JSONResponse(w, asset, http.StatusOK)
	case http.MethodPut:
		handleAssetUpdate(w, r, user, asset)
	case http.MethodDelete:
		handleAssetDelete(w, user, asset)
	}
}

func handleAssetUpdate(w http.ResponseWriter, r *http.Request, user models.User, asset models.TrainingAsset) {
	hasAdmin, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasAdmin {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}
	var patch struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	asset.Title = patch.Title
	asset.Description = patch.Description
	if patch.SortOrder > 0 {
		asset.SortOrder = patch.SortOrder
	}
	if err := models.PutTrainingAsset(&asset); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, asset, http.StatusOK)
}

func handleAssetDelete(w http.ResponseWriter, user models.User, asset models.TrainingAsset) {
	hasAdmin, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasAdmin {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}
	if asset.FilePath != "" {
		if err := os.Remove(asset.FilePath); err != nil && !os.IsNotExist(err) {
			log.Error(err)
		}
	}
	if err := models.DeleteTrainingAsset(asset.Id, models.OrgScope{OrgId: asset.OrgId}); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Asset deleted"}, http.StatusOK)
}

// TrainingAssetReorder updates sort_order across a set of assets inside a
// presentation in one call. Body: {"asset_ids": [3, 1, 2]}.
func (as *Server) TrainingAssetReorder(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasAdmin, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasAdmin {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	presId, _ := strconv.ParseInt(vars["id"], 0, 64)

	// Ensure the caller actually has access to this presentation.
	if _, err := models.GetTrainingPresentation(presId, getOrgScope(r)); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errTrainingNotFound}, http.StatusNotFound)
		return
	}

	var req struct {
		AssetIds []int64 `json:"asset_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	if err := models.ReorderTrainingAssets(presId, req.AssetIds, models.OrgScope{OrgId: user.OrgId}); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Asset order updated"}, http.StatusOK)
}
