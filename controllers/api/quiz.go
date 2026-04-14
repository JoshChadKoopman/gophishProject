package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

const errQuizNotFound = "No quiz found"

// TrainingQuiz handles CRUD for quizzes attached to training presentations.
// GET: any authenticated user (correct answers stripped for non-admins)
// POST: requires manage_training
// DELETE: requires manage_training
func (as *Server) TrainingQuiz(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	presId, _ := strconv.ParseInt(vars["id"], 0, 64)

	switch r.Method {
	case http.MethodGet:
		handleQuizGet(w, user, presId)
	case http.MethodPost:
		handleQuizPost(w, r, user, presId)
	case http.MethodDelete:
		handleQuizDelete(w, user, presId)
	}
}

func handleQuizGet(w http.ResponseWriter, user models.User, presId int64) {
	quiz, err := models.GetQuizByPresentationId(presId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errQuizNotFound}, http.StatusNotFound)
		return
	}
	// Strip correct answers and explanations for non-admin users to prevent
	// cheating. Explanations are revealed via TrainingQuizAttempt after submit.
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		for i := range quiz.Questions {
			quiz.Questions[i].CorrectOption = -1
			quiz.Questions[i].CorrectOptions = ""
			quiz.Questions[i].Explanation = ""
		}
	}
	JSONResponse(w, quiz, http.StatusOK)
}

func handleQuizPost(w http.ResponseWriter, r *http.Request, user models.User, presId int64) {
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}

	var req struct {
		PassPercentage int                   `json:"pass_percentage"`
		Questions      []models.QuizQuestion `json:"questions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	if len(req.Questions) == 0 {
		JSONResponse(w, models.Response{Success: false, Message: "Quiz must have at least one question"}, http.StatusBadRequest)
		return
	}
	if req.PassPercentage <= 0 || req.PassPercentage > 100 {
		req.PassPercentage = 70
	}

	// Normalize question types (empty → multiple_choice for backward compat).
	for i := range req.Questions {
		req.Questions[i].QuestionType = models.NormalizeQuestionType(req.Questions[i].QuestionType)
	}

	// Delete existing quiz if any, then create new one.
	models.DeleteQuiz(presId)

	quiz := &models.Quiz{
		PresentationId: presId,
		PassPercentage: req.PassPercentage,
		CreatedBy:      user.Id,
	}
	if err := models.PostQuiz(quiz); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	if err := models.SaveQuizQuestions(quiz.Id, req.Questions); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	quiz.Questions = req.Questions
	JSONResponse(w, quiz, http.StatusCreated)
}

func handleQuizDelete(w http.ResponseWriter, user models.User, presId int64) {
	hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPermission {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}
	if err := models.DeleteQuiz(presId); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Quiz deleted"}, http.StatusOK)
}

// TrainingQuizAttempt handles quiz submissions and retrieval of attempts.
// POST: any authenticated user — submit answers and get graded
// GET: any authenticated user — get own attempts
func (as *Server) TrainingQuizAttempt(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	presId, _ := strconv.ParseInt(vars["id"], 0, 64)

	switch r.Method {
	case http.MethodGet:
		handleAttemptGet(w, user, presId)
	case http.MethodPost:
		handleAttemptPost(w, r, user, presId)
	}
}

func handleAttemptGet(w http.ResponseWriter, user models.User, presId int64) {
	quiz, err := models.GetQuizByPresentationId(presId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errQuizNotFound}, http.StatusNotFound)
		return
	}
	attempts, err := models.GetQuizAttempts(user.Id, quiz.Id)
	if err != nil {
		JSONResponse(w, []models.QuizAttempt{}, http.StatusOK)
		return
	}
	JSONResponse(w, attempts, http.StatusOK)
}

// quizAttemptRequest supports both legacy single-answer payloads and the new
// multi-select payload. A submitted answer is an array of selected indices
// (single element for multiple_choice/true_false, possibly multiple for
// multi_select). Older clients may still POST `answers: [0, 1, 2]`, which is
// auto-upgraded to `[[0], [1], [2]]`.
type quizAttemptRequest struct {
	Answers [][]int `json:"answers"`
}

// UnmarshalJSON accepts either [][]int (new) or []int (legacy) for Answers.
func (r *quizAttemptRequest) UnmarshalJSON(data []byte) error {
	// Try the new format first.
	var newFormat struct {
		Answers [][]int `json:"answers"`
	}
	if err := json.Unmarshal(data, &newFormat); err == nil && newFormat.Answers != nil {
		r.Answers = newFormat.Answers
		return nil
	}
	// Fall back to legacy []int format.
	var legacy struct {
		Answers []int `json:"answers"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}
	r.Answers = make([][]int, len(legacy.Answers))
	for i, a := range legacy.Answers {
		r.Answers[i] = []int{a}
	}
	return nil
}

func handleAttemptPost(w http.ResponseWriter, r *http.Request, user models.User, presId int64) {
	quiz, err := models.GetQuizByPresentationId(presId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errQuizNotFound}, http.StatusNotFound)
		return
	}

	var req quizAttemptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	if len(req.Answers) != len(quiz.Questions) {
		JSONResponse(w, models.Response{
			Success: false,
			Message: "Number of answers must match number of questions",
		}, http.StatusBadRequest)
		return
	}

	score, perQuestion := gradeQuizAttempt(quiz.Questions, req.Answers)
	total := len(quiz.Questions)
	passed := total > 0 && (score*100/total) >= quiz.PassPercentage

	answersJSON, _ := json.Marshal(req.Answers)
	attempt := &models.QuizAttempt{
		QuizId:         quiz.Id,
		UserId:         user.Id,
		Score:          score,
		TotalQuestions: total,
		Passed:         passed,
		Answers:        string(answersJSON),
	}
	if err := models.PostQuizAttempt(attempt); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"attempt":      attempt,
		"passed":       passed,
		"score":        score,
		"total":        total,
		"per_question": perQuestion,
	}
	if passed {
		finalizePassedQuiz(user.Id, presId, attempt.Id, resp)
	}
	JSONResponse(w, resp, http.StatusOK)
}

// perQuestionResult is included in the attempt response so the frontend can
// render the correct/incorrect state and explanation for each question.
type perQuestionResult struct {
	QuestionId  int64 `json:"question_id"`
	Correct     bool  `json:"correct"`
	Explanation string `json:"explanation,omitempty"`
}

func gradeQuizAttempt(questions []models.QuizQuestion, answers [][]int) (int, []perQuestionResult) {
	score := 0
	results := make([]perQuestionResult, len(questions))
	for i, q := range questions {
		correct := q.GradeAnswer(answers[i])
		if correct {
			score++
		}
		results[i] = perQuestionResult{
			QuestionId:  q.Id,
			Correct:     correct,
			Explanation: q.Explanation,
		}
	}
	return score, results
}

func finalizePassedQuiz(userId, presId, attemptId int64, resp map[string]interface{}) {
	cert, err := models.IssueCertificate(userId, presId, attemptId)
	if err == nil {
		resp["certificate"] = cert
	}
	models.UpdateAssignmentStatus(userId, presId, models.AssignmentStatusCompleted)
	cp, cpErr := models.GetCourseProgress(userId, presId)
	if cpErr == nil && cp.Status != "complete" {
		cp.Status = "complete"
		models.SaveCourseProgress(&cp)
	}
}
