package models

import (
	"encoding/json"
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
)

// Story node type constants — control how the frontend renders a node in the
// branching narrative engine.
const (
	// StoryNodeChoice presents the user with a set of choices that branch the
	// narrative based on which option they pick.
	StoryNodeChoice = "choice"
	// StoryNodeInfo is a plain narrative beat with a single "continue" path.
	// Used for exposition and setup between decisions.
	StoryNodeInfo = "info"
	// StoryNodeEnding terminates the scenario with a pass/fail outcome.
	StoryNodeEnding = "ending"
)

// Scenario outcome constants used on terminal nodes.
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
	OutcomeNeutral = "neutral"
)

// Progress status constants.
const (
	StoryProgressInProgress = "in_progress"
	StoryProgressCompleted  = "completed"
	StoryProgressAbandoned  = "abandoned"
)

// StoryScenario represents a branching narrative training scenario. Each
// scenario is a small interactive story where the learner makes security
// decisions and sees consequences, providing phished.io-style immersive
// training beyond passive video+quiz content.
type StoryScenario struct {
	Id             int64       `json:"id" gorm:"column:id; primary_key:yes"`
	PresentationId int64       `json:"presentation_id" gorm:"column:presentation_id"`
	Title          string      `json:"title" gorm:"column:title"`
	Description    string      `json:"description" gorm:"column:description;type:text"`
	Category       string      `json:"category" gorm:"column:category"`
	Difficulty     int         `json:"difficulty" gorm:"column:difficulty"`
	StartNodeId    int64       `json:"start_node_id" gorm:"column:start_node_id"`
	PassThreshold  int         `json:"pass_threshold" gorm:"column:pass_threshold"`
	CreatedBy      int64       `json:"created_by" gorm:"column:created_by"`
	CreatedDate    time.Time   `json:"created_date" gorm:"column:created_date"`
	ModifiedDate   time.Time   `json:"modified_date" gorm:"column:modified_date"`
	Nodes          []StoryNode `json:"nodes,omitempty" gorm:"-"`
}

// TableName ensures GORM pluralizes correctly.
func (StoryScenario) TableName() string { return "story_scenarios" }

// StoryNode is a beat in the narrative. A node may present exposition
// (node_type=info), a branching decision (node_type=choice), or a final
// outcome (node_type=ending).
type StoryNode struct {
	Id         int64         `json:"id" gorm:"column:id; primary_key:yes"`
	ScenarioId int64         `json:"scenario_id" gorm:"column:scenario_id"`
	NodeKey    string        `json:"node_key" gorm:"column:node_key"`
	NodeType   string        `json:"node_type" gorm:"column:node_type"`
	Title      string        `json:"title" gorm:"column:title"`
	Body       string        `json:"body" gorm:"column:body;type:text"`
	MediaURL   string        `json:"media_url" gorm:"column:media_url"`
	ScoreDelta int           `json:"score_delta" gorm:"column:score_delta"`
	IsTerminal bool          `json:"is_terminal" gorm:"column:is_terminal"`
	Outcome    string        `json:"outcome" gorm:"column:outcome"`
	Choices    []StoryChoice `json:"choices,omitempty" gorm:"-"`
}

func (StoryNode) TableName() string { return "story_nodes" }

// StoryChoice is an edge in the narrative graph. Each choice represents a
// decision the learner can take, which leads to another node and optionally
// adjusts their running score.
type StoryChoice struct {
	Id         int64  `json:"id" gorm:"column:id; primary_key:yes"`
	NodeId     int64  `json:"node_id" gorm:"column:node_id"`
	Label      string `json:"label" gorm:"column:label"`
	NextNodeId int64  `json:"next_node_id" gorm:"column:next_node_id"`
	ScoreDelta int    `json:"score_delta" gorm:"column:score_delta"`
	Feedback   string `json:"feedback" gorm:"column:feedback;type:text"`
	IsCorrect  bool   `json:"is_correct" gorm:"column:is_correct"`
	SortOrder  int    `json:"sort_order" gorm:"column:sort_order"`
}

func (StoryChoice) TableName() string { return "story_choices" }

// StoryProgress records a learner's state inside a scenario: the current
// node, their running score, the full path of node keys they've visited,
// and the overall status.
type StoryProgress struct {
	Id            int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId        int64     `json:"user_id" gorm:"column:user_id"`
	ScenarioId    int64     `json:"scenario_id" gorm:"column:scenario_id"`
	CurrentNodeId int64     `json:"current_node_id" gorm:"column:current_node_id"`
	Score         int       `json:"score" gorm:"column:score"`
	Path          string    `json:"path" gorm:"column:path;type:text"`
	Status        string    `json:"status" gorm:"column:status"`
	StartedDate   time.Time `json:"started_date" gorm:"column:started_date"`
	CompletedDate time.Time `json:"completed_date" gorm:"column:completed_date"`
}

func (StoryProgress) TableName() string { return "story_progress" }

// PathKeys deserializes the JSON-encoded path slice.
func (p *StoryProgress) PathKeys() []string {
	if p.Path == "" {
		return nil
	}
	var keys []string
	_ = json.Unmarshal([]byte(p.Path), &keys)
	return keys
}

// AppendPath adds a node key to the learner's path and re-encodes it.
func (p *StoryProgress) AppendPath(key string) {
	keys := p.PathKeys()
	keys = append(keys, key)
	data, _ := json.Marshal(keys)
	p.Path = string(data)
}

var (
	// ErrScenarioNotFound is returned when a scenario lookup misses.
	ErrScenarioNotFound = errors.New("Scenario not found")
	// ErrStoryNodeNotFound is returned when a node lookup misses.
	ErrStoryNodeNotFound = errors.New("Story node not found")
	// ErrInvalidChoice is returned when a chosen choice does not belong to
	// the user's current node.
	ErrInvalidChoice = errors.New("Invalid choice for current node")
)

const (
	queryWhereScenarioID = "scenario_id=?"
	queryWhereNodeID     = "node_id=?"
	orderBySortOrderAsc  = "sort_order asc"
)

// GetStoryScenarios returns all scenarios, ordered by creation date.
func GetStoryScenarios() ([]StoryScenario, error) {
	scenarios := []StoryScenario{}
	err := db.Order("created_date desc").Find(&scenarios).Error
	return scenarios, err
}

// GetStoryScenario returns a scenario by ID with its nodes and choices loaded.
func GetStoryScenario(id int64) (StoryScenario, error) {
	s := StoryScenario{}
	if err := db.Where("id=?", id).First(&s).Error; err != nil {
		return s, err
	}
	nodes, err := getStoryNodesForScenario(s.Id)
	if err != nil {
		return s, err
	}
	s.Nodes = nodes
	return s, nil
}

// GetStoryScenarioByPresentation returns the scenario attached to a presentation.
func GetStoryScenarioByPresentation(presentationId int64) (StoryScenario, error) {
	s := StoryScenario{}
	if err := db.Where(queryWherePresentationID, presentationId).First(&s).Error; err != nil {
		return s, err
	}
	nodes, err := getStoryNodesForScenario(s.Id)
	if err != nil {
		return s, err
	}
	s.Nodes = nodes
	return s, nil
}

// getStoryNodesForScenario fetches all nodes for a scenario and hydrates each
// with its choices.
func getStoryNodesForScenario(scenarioId int64) ([]StoryNode, error) {
	nodes := []StoryNode{}
	if err := db.Where(queryWhereScenarioID, scenarioId).Find(&nodes).Error; err != nil {
		return nil, err
	}
	for i := range nodes {
		choices := []StoryChoice{}
		if err := db.Where(queryWhereNodeID, nodes[i].Id).Order(orderBySortOrderAsc).Find(&choices).Error; err != nil {
			return nil, err
		}
		nodes[i].Choices = choices
	}
	return nodes, nil
}

// GetStoryNode returns a single node (with its choices) by ID.
func GetStoryNode(id int64) (StoryNode, error) {
	n := StoryNode{}
	if err := db.Where("id=?", id).First(&n).Error; err != nil {
		return n, err
	}
	choices := []StoryChoice{}
	err := db.Where(queryWhereNodeID, n.Id).Order(orderBySortOrderAsc).Find(&choices).Error
	n.Choices = choices
	return n, err
}

// GetStoryNodeByKey looks up a node within a scenario by its node_key.
func GetStoryNodeByKey(scenarioId int64, key string) (StoryNode, error) {
	n := StoryNode{}
	err := db.Where("scenario_id=? AND node_key=?", scenarioId, key).First(&n).Error
	if err != nil {
		return n, err
	}
	choices := []StoryChoice{}
	err = db.Where(queryWhereNodeID, n.Id).Order(orderBySortOrderAsc).Find(&choices).Error
	n.Choices = choices
	return n, err
}

// PostStoryScenario creates a scenario and inserts its nodes + choices in a
// single transaction. Choices' NextNodeId should already be resolved by the
// caller (e.g. via a node_key->id map after nodes are inserted). StartNodeId
// is resolved from startKey against the inserted node set if provided.
func PostStoryScenario(s *StoryScenario, nodes []StoryNode, startKey string) error {
	tx := db.Begin()
	s.CreatedDate = time.Now().UTC()
	s.ModifiedDate = s.CreatedDate
	if err := tx.Save(s).Error; err != nil {
		tx.Rollback()
		return err
	}

	keyToId, err := insertScenarioNodes(tx, s.Id, nodes)
	if err != nil {
		tx.Rollback()
		return err
	}
	if err := insertScenarioChoices(tx, keyToId, nodes); err != nil {
		tx.Rollback()
		return err
	}
	if err := resolveStartNode(tx, s, startKey, keyToId); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// insertScenarioNodes inserts nodes and returns a key->id map. The Choices
// field on each node is ignored by GORM (tag: gorm:"-") and is handled in a
// second pass by insertScenarioChoices once all nodes have IDs.
func insertScenarioNodes(tx *gorm.DB, scenarioId int64, nodes []StoryNode) (map[string]int64, error) {
	keyToId := map[string]int64{}
	for i := range nodes {
		nodes[i].ScenarioId = scenarioId
		if err := tx.Save(&nodes[i]).Error; err != nil {
			return nil, err
		}
		keyToId[nodes[i].NodeKey] = nodes[i].Id
	}
	return keyToId, nil
}

// insertScenarioChoices persists all choices from the node input slice,
// stamping NodeId from the key->id map and numbering via SortOrder.
func insertScenarioChoices(tx *gorm.DB, keyToId map[string]int64, nodes []StoryNode) error {
	for _, n := range nodes {
		parentId := keyToId[n.NodeKey]
		for i := range n.Choices {
			n.Choices[i].NodeId = parentId
			n.Choices[i].SortOrder = i
			if err := tx.Save(&n.Choices[i]).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func resolveStartNode(tx *gorm.DB, s *StoryScenario, startKey string, keyToId map[string]int64) error {
	if startKey == "" {
		return nil
	}
	id, ok := keyToId[startKey]
	if !ok {
		return nil
	}
	s.StartNodeId = id
	return tx.Save(s).Error
}

// PutStoryScenario updates a scenario's metadata. Nodes/choices are managed
// via dedicated endpoints to keep the data model tractable.
func PutStoryScenario(s *StoryScenario) error {
	s.ModifiedDate = time.Now().UTC()
	return db.Save(s).Error
}

// DeleteStoryScenario removes a scenario and all of its nodes, choices, and
// progress records.
func DeleteStoryScenario(id int64) error {
	// Gather node IDs so we can delete their choices.
	nodes := []StoryNode{}
	if err := db.Where(queryWhereScenarioID, id).Find(&nodes).Error; err != nil {
		return err
	}
	for _, n := range nodes {
		if err := db.Where(queryWhereNodeID, n.Id).Delete(&StoryChoice{}).Error; err != nil {
			log.Error(err)
		}
	}
	if err := db.Where(queryWhereScenarioID, id).Delete(&StoryNode{}).Error; err != nil {
		log.Error(err)
	}
	if err := db.Where(queryWhereScenarioID, id).Delete(&StoryProgress{}).Error; err != nil {
		log.Error(err)
	}
	return db.Where("id=?", id).Delete(&StoryScenario{}).Error
}

// GetOrCreateStoryProgress returns a user's in-progress attempt at a scenario,
// or creates a fresh one seeded at the start node.
func GetOrCreateStoryProgress(userId, scenarioId int64) (StoryProgress, error) {
	p := StoryProgress{}
	err := db.Where("user_id=? AND scenario_id=? AND status=?", userId, scenarioId, StoryProgressInProgress).
		Order("started_date desc").First(&p).Error
	if err == nil {
		return p, nil
	}

	// Start fresh.
	s, err := GetStoryScenario(scenarioId)
	if err != nil {
		return p, err
	}
	p = StoryProgress{
		UserId:        userId,
		ScenarioId:    scenarioId,
		CurrentNodeId: s.StartNodeId,
		Score:         0,
		Status:        StoryProgressInProgress,
		StartedDate:   time.Now().UTC(),
	}
	// Seed path with the start node's key if it exists.
	if start, nerr := GetStoryNode(s.StartNodeId); nerr == nil {
		p.AppendPath(start.NodeKey)
	}
	if err := db.Save(&p).Error; err != nil {
		return p, err
	}
	return p, nil
}

// AdvanceStoryProgress applies a learner's choice: validates the choice
// belongs to the current node, moves to the next node, updates score and
// path, and marks the progress complete on terminal nodes.
func AdvanceStoryProgress(p *StoryProgress, choiceId int64) (StoryNode, error) {
	// Load the current node and its choices.
	current, err := GetStoryNode(p.CurrentNodeId)
	if err != nil {
		return StoryNode{}, err
	}

	// Validate choice belongs to current node.
	var chosen *StoryChoice
	for i := range current.Choices {
		if current.Choices[i].Id == choiceId {
			chosen = &current.Choices[i]
			break
		}
	}
	if chosen == nil {
		return StoryNode{}, ErrInvalidChoice
	}

	// Advance to next node.
	next, err := GetStoryNode(chosen.NextNodeId)
	if err != nil {
		return StoryNode{}, err
	}

	p.Score += chosen.ScoreDelta + next.ScoreDelta
	p.CurrentNodeId = next.Id
	p.AppendPath(next.NodeKey)

	if next.IsTerminal {
		p.Status = StoryProgressCompleted
		p.CompletedDate = time.Now().UTC()
	}

	if err := db.Save(p).Error; err != nil {
		return StoryNode{}, err
	}
	return next, nil
}

// SaveStoryChoice inserts or updates a single choice. Used by the API layer
// during the second pass of scenario creation (after node IDs exist).
func SaveStoryChoice(c *StoryChoice) error {
	return db.Save(c).Error
}

// GetStoryProgressHistory returns all progress records for a user on a scenario.
func GetStoryProgressHistory(userId, scenarioId int64) ([]StoryProgress, error) {
	records := []StoryProgress{}
	err := db.Where("user_id=? AND scenario_id=?", userId, scenarioId).
		Order("started_date desc").Find(&records).Error
	return records, err
}
