package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// TrainingQuiz handles CRUD for quizzes attached to training presentations.
// GET: any authenticated user (correct_option stripped for non-admins)
// POST: requires manage_training
// DELETE: requires manage_training
func (as *Server) TrainingQuiz(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	presId, _ := strconv.ParseInt(vars["id"], 0, 64)

	switch {
	case r.Method == "GET":
		quiz, err := models.GetQuizByPresentationId(presId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "No quiz found"}, http.StatusNotFound)
			return
		}
		// Strip correct answers for non-admin users to prevent cheating
		hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
		if !hasPermission {
			for i := range quiz.Questions {
				quiz.Questions[i].CorrectOption = -1
			}
		}
		JSONResponse(w, quiz, http.StatusOK)

	case r.Method == "POST":
		hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
		if !hasPermission {
			JSONResponse(w, models.Response{Success: false, Message: "Permission denied"}, http.StatusForbidden)
			return
		}

		var req struct {
			PassPercentage int                    `json:"pass_percentage"`
			Questions      []models.QuizQuestion  `json:"questions"`
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
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

		// Delete existing quiz if any, then create new one
		models.DeleteQuiz(presId)

		quiz := &models.Quiz{
			PresentationId: presId,
			PassPercentage: req.PassPercentage,
			CreatedBy:      user.Id,
		}
		err = models.PostQuiz(quiz)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		err = models.SaveQuizQuestions(quiz.Id, req.Questions)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		// Return the created quiz with questions
		quiz.Questions = req.Questions
		JSONResponse(w, quiz, http.StatusCreated)

	case r.Method == "DELETE":
		hasPermission, _ := user.HasPermission(models.PermissionManageTraining)
		if !hasPermission {
			JSONResponse(w, models.Response{Success: false, Message: "Permission denied"}, http.StatusForbidden)
			return
		}
		err := models.DeleteQuiz(presId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Quiz deleted"}, http.StatusOK)
	}
}

// TrainingQuizAttempt handles quiz submissions and retrieval of attempts.
// POST: any authenticated user — submit answers and get graded
// GET: any authenticated user — get own attempts
func (as *Server) TrainingQuizAttempt(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	presId, _ := strconv.ParseInt(vars["id"], 0, 64)

	switch {
	case r.Method == "GET":
		quiz, err := models.GetQuizByPresentationId(presId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "No quiz found"}, http.StatusNotFound)
			return
		}
		attempts, err := models.GetQuizAttempts(user.Id, quiz.Id)
		if err != nil {
			JSONResponse(w, []models.QuizAttempt{}, http.StatusOK)
			return
		}
		JSONResponse(w, attempts, http.StatusOK)

	case r.Method == "POST":
		quiz, err := models.GetQuizByPresentationId(presId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "No quiz found"}, http.StatusNotFound)
			return
		}

		var req struct {
			Answers []int `json:"answers"`
		}
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
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

		// Grade the quiz
		score := 0
		total := len(quiz.Questions)
		for i, q := range quiz.Questions {
			if req.Answers[i] == q.CorrectOption {
				score++
			}
		}

		// Integer math to avoid float precision issues
		passed := false
		if total > 0 {
			passed = (score * 100 / total) >= quiz.PassPercentage
		}

		answersJSON, _ := json.Marshal(req.Answers)
		attempt := &models.QuizAttempt{
			QuizId:         quiz.Id,
			UserId:         user.Id,
			Score:          score,
			TotalQuestions: total,
			Passed:         passed,
			Answers:        string(answersJSON),
		}
		err = models.PostQuizAttempt(attempt)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		// Build response
		resp := map[string]interface{}{
			"attempt": attempt,
			"passed":  passed,
			"score":   score,
			"total":   total,
		}

		// If passed, issue certificate and update assignment
		if passed {
			cert, err := models.IssueCertificate(user.Id, presId, attempt.Id)
			if err == nil {
				resp["certificate"] = cert
			}
			// Update assignment status if one exists
			models.UpdateAssignmentStatus(user.Id, presId, models.AssignmentStatusCompleted)
			// Update course progress to complete
			cp, cpErr := models.GetCourseProgress(user.Id, presId)
			if cpErr == nil && cp.Status != "complete" {
				cp.Status = "complete"
				models.SaveCourseProgress(&cp)
			}
		}

		JSONResponse(w, resp, http.StatusOK)
	}
}
