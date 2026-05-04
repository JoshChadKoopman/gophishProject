package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// trainingUploadDir is the directory where training presentations are stored
const trainingUploadDir = "./static/training_uploads"

// allowedTrainingTypes defines the allowed file MIME types for training uploads
var allowedTrainingTypes = map[string]bool{
	"application/pdf":               true,
	"application/vnd.ms-powerpoint": true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"application/vnd.oasis.opendocument.presentation":                           true,
	"video/mp4":       true,
	"video/webm":      true,
	"video/x-msvideo": true,
}

// allowedThumbnailTypes defines the allowed MIME types for thumbnail images
var allowedThumbnailTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// maxUploadSize is the maximum allowed file size (50MB)
const maxUploadSize = 50 << 20

// Shared string constants to avoid S1192 duplicate literal warnings.
const (
	headerContentType   = "Content-Type"
	errTrainingNotFound = "Training presentation not found"
)

// TrainingPresentations handles listing and creating training presentations
func (as *Server) TrainingPresentations(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		tps, err := models.GetTrainingPresentations(getOrgScope(r))
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, tps, http.StatusOK)

	case r.Method == "POST":
		as.handleTrainingUpload(w, r)
	}
}

// handleTrainingUpload processes the POST side of TrainingPresentations (file upload).
func (as *Server) handleTrainingUpload(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	hasAdmin, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasAdmin {
		JSONResponse(w, models.Response{Success: false, Message: "Only administrators can upload training presentations"}, http.StatusForbidden)
		return
	}

	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
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
	if !allowedTrainingTypes[contentType] {
		JSONResponse(w, models.Response{Success: false, Message: "File type not allowed. Please upload PDF, PowerPoint, ODP, or video files."}, http.StatusBadRequest)
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

	tp := models.TrainingPresentation{
		OrgId:        user.OrgId,
		Name:         r.FormValue("name"),
		Description:  r.FormValue("description"),
		FileName:     header.Filename,
		FilePath:     filePath,
		FileSize:     written,
		ContentType:  contentType,
		YouTubeURL:   r.FormValue("youtube_url"),
		ContentPages: r.FormValue("content_pages"),
		UploadedBy:   user.Id,
	}

	if thumbPath, ok := saveThumbnail(r, filePath, w); ok {
		tp.ThumbnailPath = thumbPath
	} else if thumbPath == "error" {
		return // saveThumbnail already wrote the error response
	}

	err = models.PostTrainingPresentation(&tp)
	if err != nil {
		os.Remove(filePath)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, tp, http.StatusCreated)
}

// saveUploadedFile writes a multipart file to disk with a unique name and returns the path and bytes written.
func saveUploadedFile(file io.Reader, originalName string) (string, int64, error) {
	ext := filepath.Ext(originalName)
	safeBaseName := strings.TrimSuffix(originalName, ext)
	safeBaseName = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == '\x00' {
			return '_'
		}
		return r
	}, safeBaseName)
	uniqueName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), safeBaseName, ext)
	filePath := filepath.Join(trainingUploadDir, uniqueName)

	dst, err := os.Create(filePath)
	if err != nil {
		return "", 0, err
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath)
		return "", 0, err
	}
	return filePath, written, nil
}

// saveThumbnail handles the optional thumbnail upload. Returns ("", true) if no
// thumbnail was provided, (path, true) on success, or ("error", false) if an error
// response was already written.
func saveThumbnail(r *http.Request, mainFilePath string, w http.ResponseWriter) (string, bool) {
	thumbFile, thumbHeader, thumbErr := r.FormFile("thumbnail")
	if thumbErr != nil {
		return "", true // No thumbnail provided
	}
	defer thumbFile.Close()
	thumbContentType := thumbHeader.Header.Get(headerContentType)
	if !allowedThumbnailTypes[thumbContentType] {
		os.Remove(mainFilePath)
		JSONResponse(w, models.Response{Success: false, Message: "Thumbnail must be an image (JPEG, PNG, GIF, or WebP)."}, http.StatusBadRequest)
		return "error", false
	}
	thumbExt := filepath.Ext(thumbHeader.Filename)
	thumbUnique := fmt.Sprintf("thumb_%d%s", time.Now().UnixNano(), thumbExt)
	thumbPath := filepath.Join(trainingUploadDir, thumbUnique)
	thumbDst, err := os.Create(thumbPath)
	if err != nil {
		log.Error(err)
		os.Remove(mainFilePath)
		JSONResponse(w, models.Response{Success: false, Message: "Error saving thumbnail"}, http.StatusInternalServerError)
		return "error", false
	}
	defer thumbDst.Close()
	if _, err := io.Copy(thumbDst, thumbFile); err != nil {
		log.Error(err)
		os.Remove(mainFilePath)
		os.Remove(thumbPath)
		JSONResponse(w, models.Response{Success: false, Message: "Error saving thumbnail"}, http.StatusInternalServerError)
		return "error", false
	}
	return thumbPath, true
}

// TrainingPresentation handles getting, updating, and deleting a single training presentation
func (as *Server) TrainingPresentation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}
	tp, err := models.GetTrainingPresentation(id, getOrgScope(r))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errTrainingNotFound}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		JSONResponse(w, tp, http.StatusOK)
	case r.Method == "DELETE":
		as.handleTrainingDelete(w, r, tp, id)
	case r.Method == "PUT":
		as.handleTrainingUpdate(w, r, tp)
	}
}

func (as *Server) handleTrainingDelete(w http.ResponseWriter, r *http.Request, tp models.TrainingPresentation, id int64) {
	user := ctx.Get(r, "user").(models.User)
	hasAdmin, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasAdmin {
		JSONResponse(w, models.Response{Success: false, Message: "Only administrators can delete training presentations"}, http.StatusForbidden)
		return
	}
	if tp.FilePath != "" {
		os.Remove(tp.FilePath)
	}
	if tp.ThumbnailPath != "" {
		os.Remove(tp.ThumbnailPath)
	}
	if err := models.DeleteTrainingPresentation(id); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	log.Infof("Deleted training presentation with id: %d", id)
	JSONResponse(w, models.Response{Success: true, Message: "Training presentation deleted successfully!"}, http.StatusOK)
}

func (as *Server) handleTrainingUpdate(w http.ResponseWriter, r *http.Request, tp models.TrainingPresentation) {
	user := ctx.Get(r, "user").(models.User)
	hasAdmin, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasAdmin {
		JSONResponse(w, models.Response{Success: false, Message: "Only administrators can modify training presentations"}, http.StatusForbidden)
		return
	}
	updateData := struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		YouTubeURL   string `json:"youtube_url"`
		ContentPages string `json:"content_pages"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	tp.Name = updateData.Name
	tp.Description = updateData.Description
	tp.YouTubeURL = updateData.YouTubeURL
	tp.ContentPages = updateData.ContentPages
	if err := models.PutTrainingPresentation(&tp); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, tp, http.StatusOK)
}

// TrainingPresentationDownload handles serving the file for download/viewing
func (as *Server) TrainingPresentationDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}
	tp, err := models.GetTrainingPresentation(id, getOrgScope(r))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errTrainingNotFound}, http.StatusNotFound)
		return
	}

	// Check that the file exists
	if _, err := os.Stat(tp.FilePath); os.IsNotExist(err) {
		JSONResponse(w, models.Response{Success: false, Message: "File not found on disk"}, http.StatusNotFound)
		return
	}

	w.Header().Set(headerContentType, tp.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", tp.FileName))
	http.ServeFile(w, r, tp.FilePath)
}

// TrainingPresentationThumbnail serves the thumbnail image for a training presentation
func (as *Server) TrainingPresentationThumbnail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}
	tp, err := models.GetTrainingPresentation(id, getOrgScope(r))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errTrainingNotFound}, http.StatusNotFound)
		return
	}

	if tp.ThumbnailPath == "" {
		http.NotFound(w, r)
		return
	}

	if _, err := os.Stat(tp.ThumbnailPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, tp.ThumbnailPath)
}

// extractedPage represents a single extracted slide/page
type extractedPage struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	MediaType string `json:"media_type,omitempty"`
	MediaURL  string `json:"media_url,omitempty"`
}

// extractSlidesResponse is the response for slide extraction
type extractSlidesResponse struct {
	Pages []extractedPage `json:"pages"`
}

// countPDFPages counts the number of pages in a PDF file by looking for /Type /Page entries
func countPDFPages(filePath string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return 0, err
	}
	// Count occurrences of "/Type /Page" that are NOT "/Type /Pages"
	// The pattern /Type /Page\b (followed by non 's') is standard in PDF objects
	re := regexp.MustCompile(`/Type\s*/Page[^s]`)
	matches := re.FindAll(data, -1)
	count := len(matches)
	if count == 0 {
		// Fallback: try /Type/Page pattern (some PDFs skip space)
		re2 := regexp.MustCompile(`/Type/Page[^s]`)
		matches2 := re2.FindAll(data, -1)
		count = len(matches2)
	}
	if count == 0 {
		count = 1 // At minimum, assume 1 page
	}
	return count, nil
}

// countPPTXSlides counts the slides in a PPTX (ZIP) file by finding slide XML entries
func countPPTXSlides(filePath string) (int, []string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return 0, nil, err
	}
	defer r.Close()

	slidePattern := regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)
	var slideNames []string
	for _, f := range r.File {
		if slidePattern.MatchString(f.Name) {
			slideNames = append(slideNames, f.Name)
		}
	}
	// Sort slide names numerically
	sort.Slice(slideNames, func(i, j int) bool {
		ni := extractSlideNumber(slideNames[i])
		nj := extractSlideNumber(slideNames[j])
		return ni < nj
	})
	return len(slideNames), slideNames, nil
}

func extractSlideNumber(name string) int {
	re := regexp.MustCompile(`slide(\d+)\.xml`)
	m := re.FindStringSubmatch(name)
	if len(m) > 1 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

// extractSlideText extracts visible text from a PPTX slide XML
func extractSlideText(r *zip.ReadCloser, slidePath string) string {
	for _, f := range r.File {
		if f.Name == slidePath {
			return readSlideXMLText(f)
		}
	}
	return ""
}

// readSlideXMLText reads a single zip file entry and extracts <a:t> text runs.
func readSlideXMLText(f *zip.File) string {
	rc, err := f.Open()
	if err != nil {
		return ""
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`<a:t[^>]*>([^<]+)</a:t>`)
	matches := re.FindAllSubmatch(data, -1)
	var parts []string
	for _, m := range matches {
		txt := strings.TrimSpace(string(m[1]))
		if txt != "" {
			parts = append(parts, txt)
		}
	}
	return strings.Join(parts, "\n")
}

// extractSlidesFromFile builds extracted pages from a file on disk
func extractSlidesFromFile(filePath, contentType string) ([]extractedPage, error) {
	if strings.Contains(contentType, "pdf") {
		return extractPDFPages(filePath)
	}
	if strings.Contains(contentType, "presentation") || strings.Contains(contentType, "powerpoint") {
		return extractPPTXPages(filePath)
	}
	return nil, fmt.Errorf("auto-extract is only supported for PDF and PowerPoint files")
}

func extractPDFPages(filePath string) ([]extractedPage, error) {
	count, err := countPDFPages(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read PDF: %v", err)
	}
	pages := make([]extractedPage, 0, count)
	for i := 1; i <= count; i++ {
		pages = append(pages, extractedPage{
			Title: fmt.Sprintf("Slide %d", i),
			Body:  fmt.Sprintf("Content from slide %d of the PDF presentation.", i),
		})
	}
	return pages, nil
}

func extractPPTXPages(filePath string) ([]extractedPage, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open PPTX: %v", err)
	}
	defer r.Close()

	_, slideNames, err := countPPTXSlides(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not count slides: %v", err)
	}
	if len(slideNames) == 0 {
		return nil, fmt.Errorf("no slides found in the presentation")
	}
	pages := make([]extractedPage, 0, len(slideNames))
	for i, sn := range slideNames {
		text := extractSlideText(r, sn)
		body := text
		if body == "" {
			body = fmt.Sprintf("Content from slide %d.", i+1)
		}
		pages = append(pages, extractedPage{
			Title: fmt.Sprintf("Slide %d", i+1),
			Body:  body,
		})
	}
	return pages, nil
}

// TrainingExtractSlidesUpload extracts slides from an uploaded file (multipart POST)
func (as *Server) TrainingExtractSlidesUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	hasAdmin, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasAdmin {
		JSONResponse(w, models.Response{Success: false, Message: "Only administrators can extract slides"}, http.StatusForbidden)
		return
	}

	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "File too large"}, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "File is required"}, http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get(headerContentType)

	// Write to a temp file for processing
	tmpFile, err := os.CreateTemp("", "slide-extract-*"+filepath.Ext(header.Filename))
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error processing file"}, http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, file); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error processing file"}, http.StatusInternalServerError)
		return
	}
	tmpFile.Close()

	pages, err := extractSlidesFromFile(tmpFile.Name(), contentType)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	JSONResponse(w, extractSlidesResponse{Pages: pages}, http.StatusOK)
}

// TrainingExtractSlidesExisting extracts slides from an already-uploaded presentation
func (as *Server) TrainingExtractSlidesExisting(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	hasAdmin, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasAdmin {
		JSONResponse(w, models.Response{Success: false, Message: "Only administrators can extract slides"}, http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}
	tp, err := models.GetTrainingPresentation(id, getOrgScope(r))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errTrainingNotFound}, http.StatusNotFound)
		return
	}

	if _, err := os.Stat(tp.FilePath); os.IsNotExist(err) {
		JSONResponse(w, models.Response{Success: false, Message: "File not found on disk"}, http.StatusNotFound)
		return
	}

	pages, err := extractSlidesFromFile(tp.FilePath, tp.ContentType)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	JSONResponse(w, extractSlidesResponse{Pages: pages}, http.StatusOK)
}

// CourseProgressResponse is the combined response for user courses with presentation details
type CourseProgressResponse struct {
	Presentation models.TrainingPresentation `json:"presentation"`
	Progress     models.CourseProgress       `json:"progress"`
	ProgressPct  int                         `json:"progress_pct"`
	HasQuiz      bool                        `json:"has_quiz"`
	QuizPassed   bool                        `json:"quiz_passed"`
	Assignment   *models.CourseAssignment    `json:"assignment,omitempty"`
	Certificate  *models.Certificate         `json:"certificate,omitempty"`
}

// TrainingCourseProgress handles getting and updating course progress for the current user
func (as *Server) TrainingCourseProgress(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	presId, _ := strconv.ParseInt(vars["id"], 0, 64)

	switch {
	case r.Method == "GET":
		cp, err := models.GetCourseProgress(user.Id, presId)
		if err != nil {
			JSONResponse(w, models.CourseProgress{
				UserId:         user.Id,
				PresentationId: presId,
				Status:         "no_progress",
			}, http.StatusOK)
			return
		}
		JSONResponse(w, cp, http.StatusOK)

	case r.Method == "PUT":
		as.handleCourseProgressUpdate(w, r, user, presId)
	}
}

func (as *Server) handleCourseProgressUpdate(w http.ResponseWriter, r *http.Request, user models.User, presId int64) {
	var updateData struct {
		CurrentPage int    `json:"current_page"`
		TotalPages  int    `json:"total_pages"`
		Status      string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	// Anti-skip gate: if requesting "complete", verify all pages were engaged
	if updateData.Status == "complete" && updateData.TotalPages > 0 {
		gate := models.ValidateCourseCompletion(user.Id, presId, updateData.TotalPages)
		if !gate.Allowed {
			JSONResponse(w, models.Response{
				Success: false,
				Message: gate.Reason,
				Data:    gate,
			}, http.StatusForbidden)
			return
		}
	}

	cp, err := models.GetCourseProgress(user.Id, presId)
	if err != nil {
		cp = models.CourseProgress{
			UserId:         user.Id,
			PresentationId: presId,
		}
	}
	cp.CurrentPage = updateData.CurrentPage
	cp.TotalPages = updateData.TotalPages
	cp.Status = updateData.Status
	if updateData.Status == "complete" {
		cp.CompletedDate = time.Now().UTC()
	}
	if err = models.SaveCourseProgress(&cp); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	if updateData.Status == "complete" && !models.QuizExistsForPresentation(presId) {
		if _, certErr := models.GetCertificateForCourse(user.Id, presId); certErr != nil {
			models.IssueCertificate(user.Id, presId, 0)
		}
		models.UpdateAssignmentStatus(user.Id, presId, models.AssignmentStatusCompleted)
	}
	JSONResponse(w, cp, http.StatusOK)
}

// TrainingMyCourses returns all presentations with progress info for the current user
func (as *Server) TrainingMyCourses(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	tps, err := models.GetTrainingPresentations(getOrgScope(r))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	progressRecords, err := models.GetUserCourseProgress(user.Id)
	if err != nil {
		progressRecords = []models.CourseProgress{}
	}

	progressMap := make(map[int64]models.CourseProgress)
	for _, cp := range progressRecords {
		progressMap[cp.PresentationId] = cp
	}

	result := make([]CourseProgressResponse, 0, len(tps))
	for _, tp := range tps {
		result = append(result, buildCourseProgressResponse(tp, user.Id, progressMap))
	}

	JSONResponse(w, result, http.StatusOK)
}

// buildCourseProgressResponse creates a single CourseProgressResponse for a
// presentation/user combination.
func buildCourseProgressResponse(tp models.TrainingPresentation, userId int64, progressMap map[int64]models.CourseProgress) CourseProgressResponse {
	cp, exists := progressMap[tp.Id]
	if !exists {
		cp = models.CourseProgress{
			UserId:         userId,
			PresentationId: tp.Id,
			Status:         "no_progress",
		}
	}

	pct := calcProgressPct(cp)

	cpr := CourseProgressResponse{
		Presentation: tp,
		Progress:     cp,
		ProgressPct:  pct,
		HasQuiz:      models.QuizExistsForPresentation(tp.Id),
	}

	if cpr.HasQuiz {
		quiz, qErr := models.GetQuizByPresentationId(tp.Id)
		if qErr == nil {
			if _, aErr := models.GetLatestPassedAttempt(userId, quiz.Id); aErr == nil {
				cpr.QuizPassed = true
			}
		}
	}

	if assignment, aErr := models.GetAssignment(userId, tp.Id); aErr == nil {
		cpr.Assignment = &assignment
	}
	if cert, cErr := models.GetCertificateForCourse(userId, tp.Id); cErr == nil {
		cpr.Certificate = &cert
	}

	return cpr
}

// calcProgressPct calculates the progress percentage from a CourseProgress record.
func calcProgressPct(cp models.CourseProgress) int {
	if cp.Status == "complete" {
		return 100
	}
	if cp.TotalPages > 0 {
		pct := int(float64(cp.CurrentPage) / float64(cp.TotalPages) * 100)
		if pct > 100 {
			return 100
		}
		return pct
	}
	return 0
}
