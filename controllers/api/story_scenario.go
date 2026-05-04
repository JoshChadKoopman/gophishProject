package api

import (
	"encoding/json"
	"net/http"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

const errScenarioNotFound = "Scenario not found"

// scenarioNodeInput mirrors StoryNode on input but references choice
// destinations by NodeKey so callers can author a full graph in one POST
// without pre-knowing DB IDs.
type scenarioNodeInput struct {
	NodeKey    string              `json:"node_key"`
	NodeType   string              `json:"node_type"`
	Title      string              `json:"title"`
	Body       string              `json:"body"`
	MediaURL   string              `json:"media_url"`
	ScoreDelta int                 `json:"score_delta"`
	IsTerminal bool                `json:"is_terminal"`
	Outcome    string              `json:"outcome"`
	Choices    []scenarioChoiceIn `json:"choices"`
}

type scenarioChoiceIn struct {
	Label       string `json:"label"`
	NextNodeKey string `json:"next_node_key"`
	ScoreDelta  int    `json:"score_delta"`
	Feedback    string `json:"feedback"`
	IsCorrect   bool   `json:"is_correct"`
}

type scenarioInput struct {
	PresentationId int64               `json:"presentation_id"`
	Title          string              `json:"title"`
	Description    string              `json:"description"`
	Category       string              `json:"category"`
	Difficulty     int                 `json:"difficulty"`
	PassThreshold  int                 `json:"pass_threshold"`
	StartKey       string              `json:"start_key"`
	Nodes          []scenarioNodeInput `json:"nodes"`
}

// TrainingScenarios lists / creates branching narrative scenarios.
// GET:  any authenticated user
// POST: requires manage_training
func (as *Server) TrainingScenarios(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		scenarios, err := models.GetStoryScenarios()
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, scenarios, http.StatusOK)
	case http.MethodPost:
		hasPerm, _ := user.HasPermission(models.PermissionManageTraining)
		if !hasPerm {
			JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
			return
		}
		handleScenarioCreate(w, r, user)
	}
}

func handleScenarioCreate(w http.ResponseWriter, r *http.Request, user models.User) {
	var in scenarioInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	if len(in.Nodes) == 0 {
		JSONResponse(w, models.Response{Success: false, Message: "Scenario must have at least one node"}, http.StatusBadRequest)
		return
	}
	if in.StartKey == "" {
		in.StartKey = in.Nodes[0].NodeKey
	}
	if in.PassThreshold <= 0 || in.PassThreshold > 100 {
		in.PassThreshold = 70
	}

	scenario := &models.StoryScenario{
		PresentationId: in.PresentationId,
		Title:          in.Title,
		Description:    in.Description,
		Category:       in.Category,
		Difficulty:     in.Difficulty,
		PassThreshold:  in.PassThreshold,
		CreatedBy:      user.Id,
	}

	// Convert scenarioNodeInput into StoryNode slices. NextNodeId is left as 0
	// and resolved after the first-pass insert via node_key -> id.
	nodes := make([]models.StoryNode, len(in.Nodes))
	pendingChoices := make([][]scenarioChoiceIn, len(in.Nodes))
	for i, n := range in.Nodes {
		nodes[i] = models.StoryNode{
			NodeKey:    n.NodeKey,
			NodeType:   n.NodeType,
			Title:      n.Title,
			Body:       n.Body,
			MediaURL:   n.MediaURL,
			ScoreDelta: n.ScoreDelta,
			IsTerminal: n.IsTerminal,
			Outcome:    n.Outcome,
		}
		pendingChoices[i] = n.Choices
	}

	if err := models.PostStoryScenario(scenario, nodes, in.StartKey); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	// Resolve and persist choices with NextNodeId mapped from NextNodeKey.
	if err := saveScenarioChoices(scenario.Id, pendingChoices, in.Nodes); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	// Return the fully-hydrated scenario.
	full, err := models.GetStoryScenario(scenario.Id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, full, http.StatusCreated)
}

// saveScenarioChoices walks the input choices after nodes have been inserted,
// resolves NextNodeKey to an actual node ID within the same scenario, and
// writes each choice. This is the second pass of a two-pass insert: pass 1
// creates all nodes so they have IDs; pass 2 wires up the graph edges.
func saveScenarioChoices(scenarioId int64, pendingChoices [][]scenarioChoiceIn, inputs []scenarioNodeInput) error {
	// Re-fetch nodes to build the key->id map with real DB IDs.
	scenario, err := models.GetStoryScenario(scenarioId)
	if err != nil {
		return err
	}
	keyToId := map[string]int64{}
	for _, n := range scenario.Nodes {
		keyToId[n.NodeKey] = n.Id
	}

	for i, choices := range pendingChoices {
		parentId := keyToId[inputs[i].NodeKey]
		for j, c := range choices {
			nextId := keyToId[c.NextNodeKey]
			choice := models.StoryChoice{
				NodeId:     parentId,
				Label:      c.Label,
				NextNodeId: nextId,
				ScoreDelta: c.ScoreDelta,
				Feedback:   c.Feedback,
				IsCorrect:  c.IsCorrect,
				SortOrder:  j,
			}
			if err := models.SaveStoryChoice(&choice); err != nil {
				return err
			}
		}
	}
	return nil
}

// TrainingScenario handles GET/PUT/DELETE for a specific scenario.
// GET:    any authenticated user (full graph)
// PUT:    requires manage_training (metadata only)
// DELETE: requires manage_training
func (as *Server) TrainingScenario(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		scenario, err := models.GetStoryScenario(id)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: errScenarioNotFound}, http.StatusNotFound)
			return
		}
		JSONResponse(w, scenario, http.StatusOK)
	case http.MethodPut:
		handleScenarioUpdate(w, r, user, id)
	case http.MethodDelete:
		handleScenarioDelete(w, user, id)
	}
}

func handleScenarioUpdate(w http.ResponseWriter, r *http.Request, user models.User, id int64) {
	hasPerm, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPerm {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}
	scenario, err := models.GetStoryScenario(id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: errScenarioNotFound}, http.StatusNotFound)
		return
	}
	var patch struct {
		Title         string `json:"title"`
		Description   string `json:"description"`
		Category      string `json:"category"`
		Difficulty    int    `json:"difficulty"`
		PassThreshold int    `json:"pass_threshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	scenario.Title = patch.Title
	scenario.Description = patch.Description
	scenario.Category = patch.Category
	scenario.Difficulty = patch.Difficulty
	if patch.PassThreshold > 0 && patch.PassThreshold <= 100 {
		scenario.PassThreshold = patch.PassThreshold
	}
	if err := models.PutStoryScenario(&scenario); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, scenario, http.StatusOK)
}

func handleScenarioDelete(w http.ResponseWriter, user models.User, id int64) {
	hasPerm, _ := user.HasPermission(models.PermissionManageTraining)
	if !hasPerm {
		JSONResponse(w, models.Response{Success: false, Message: ErrPermissionDenied}, http.StatusForbidden)
		return
	}
	if err := models.DeleteStoryScenario(id); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Scenario deleted"}, http.StatusOK)
}

// TrainingScenarioStart seeds (or resumes) a user's progress and returns the
// current node so the frontend can render the first beat.
func (as *Server) TrainingScenarioStart(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	progress, err := models.GetOrCreateStoryProgress(user.Id, id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	current, err := models.GetStoryNode(progress.CurrentNodeId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{
		"progress": progress,
		"node":     current,
	}, http.StatusOK)
}

// TrainingScenarioChoose advances the user's progress by applying a choice
// from the current node. Body: {"choice_id": 123}.
func (as *Server) TrainingScenarioChoose(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	var req struct {
		ChoiceId int64 `json:"choice_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	progress, err := models.GetOrCreateStoryProgress(user.Id, id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	next, err := models.AdvanceStoryProgress(&progress, req.ChoiceId)
	if err != nil {
		status := http.StatusBadRequest
		if err == models.ErrInvalidChoice {
			status = http.StatusBadRequest
		}
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, status)
		return
	}

	resp := map[string]interface{}{
		"progress": progress,
		"node":     next,
		"finished": next.IsTerminal,
	}
	JSONResponse(w, resp, http.StatusOK)
}

// TrainingScenarioHistory returns all of the current user's attempts at a scenario.
func (as *Server) TrainingScenarioHistory(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	history, err := models.GetStoryProgressHistory(user.Id, id)
	if err != nil {
		JSONResponse(w, []models.StoryProgress{}, http.StatusOK)
		return
	}
	JSONResponse(w, history, http.StatusOK)
}
