package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

// setupScenarioTest initialises an in-memory database for scenario tests.
func setupScenarioTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM story_progress")
	db.Exec("DELETE FROM story_choices")
	db.Exec("DELETE FROM story_nodes")
	db.Exec("DELETE FROM story_scenarios")
	return func() {
		db.Exec("DELETE FROM story_progress")
		db.Exec("DELETE FROM story_choices")
		db.Exec("DELETE FROM story_nodes")
		db.Exec("DELETE FROM story_scenarios")
	}
}

// buildSampleScenario returns a small branching scenario: start -> two
// choices, one leading to a success ending, the other to a failure ending.
// Choice NextNodeId values are filled in after the nodes exist by calling
// PostStoryScenario via a helper.
func buildSampleScenario(t *testing.T) StoryScenario {
	t.Helper()
	s := StoryScenario{
		Title:         "Suspicious email",
		Description:   "Decide how to handle a phishing attempt",
		Category:      "credential_harvesting",
		Difficulty:    2,
		PassThreshold: 70,
		CreatedBy:     1,
	}
	nodes := []StoryNode{
		{
			NodeKey:  "start",
			NodeType: StoryNodeChoice,
			Title:    "Inbox",
			Body:     "You receive an email from 'IT Support' asking you to reset your password.",
			Choices: []StoryChoice{
				{Label: "Click the link", IsCorrect: false, ScoreDelta: -10},
				{Label: "Report to IT", IsCorrect: true, ScoreDelta: 10},
			},
		},
		{
			NodeKey:    "good_end",
			NodeType:   StoryNodeEnding,
			Title:      "Good call",
			Body:       "You reported the phishing attempt and IT confirmed it.",
			IsTerminal: true,
			Outcome:    OutcomeSuccess,
			ScoreDelta: 20,
		},
		{
			NodeKey:    "bad_end",
			NodeType:   StoryNodeEnding,
			Title:      "Compromised",
			Body:       "You entered your credentials on a fake page.",
			IsTerminal: true,
			Outcome:    OutcomeFailure,
			ScoreDelta: -20,
		},
	}

	// Call PostStoryScenario first so nodes get IDs; the caller will resolve
	// choice targets and then persist via SaveStoryChoice.
	if err := PostStoryScenario(&s, nodes, "start"); err != nil {
		t.Fatalf("PostStoryScenario: %v", err)
	}

	full, err := GetStoryScenario(s.Id)
	if err != nil {
		t.Fatalf("GetStoryScenario: %v", err)
	}

	// Map node keys to ids, then wire choice targets.
	keyToId := map[string]int64{}
	for _, n := range full.Nodes {
		keyToId[n.NodeKey] = n.Id
	}
	startNode, err := GetStoryNodeByKey(s.Id, "start")
	if err != nil {
		t.Fatalf("GetStoryNodeByKey: %v", err)
	}
	startNode.Choices[0].NextNodeId = keyToId["bad_end"]
	startNode.Choices[1].NextNodeId = keyToId["good_end"]
	if err := SaveStoryChoice(&startNode.Choices[0]); err != nil {
		t.Fatalf("SaveStoryChoice: %v", err)
	}
	if err := SaveStoryChoice(&startNode.Choices[1]); err != nil {
		t.Fatalf("SaveStoryChoice: %v", err)
	}
	return full
}

func TestPostAndGetStoryScenario(t *testing.T) {
	teardown := setupScenarioTest(t)
	defer teardown()

	s := buildSampleScenario(t)
	if s.Id == 0 {
		t.Fatal("expected scenario to have an ID")
	}
	if s.StartNodeId == 0 {
		t.Fatal("expected start_node_id to be resolved from start key")
	}
	if len(s.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(s.Nodes))
	}

	// Start node should have two choices.
	start, err := GetStoryNodeByKey(s.Id, "start")
	if err != nil {
		t.Fatalf("GetStoryNodeByKey: %v", err)
	}
	if len(start.Choices) != 2 {
		t.Fatalf("expected start node to have 2 choices, got %d", len(start.Choices))
	}
}

func TestStoryProgressHappyPath(t *testing.T) {
	teardown := setupScenarioTest(t)
	defer teardown()

	s := buildSampleScenario(t)

	progress, err := GetOrCreateStoryProgress(42, s.Id)
	if err != nil {
		t.Fatalf("GetOrCreateStoryProgress: %v", err)
	}
	if progress.CurrentNodeId != s.StartNodeId {
		t.Fatalf("expected progress to start at node %d, got %d", s.StartNodeId, progress.CurrentNodeId)
	}

	// Pick the "correct" choice (Report to IT).
	start, _ := GetStoryNodeByKey(s.Id, "start")
	correctChoice := start.Choices[1]

	next, err := AdvanceStoryProgress(&progress, correctChoice.Id)
	if err != nil {
		t.Fatalf("AdvanceStoryProgress: %v", err)
	}
	if next.Outcome != OutcomeSuccess {
		t.Fatalf("expected success outcome, got %q", next.Outcome)
	}
	if progress.Status != StoryProgressCompleted {
		t.Fatalf("expected progress to be completed, got %q", progress.Status)
	}
	if progress.Score <= 0 {
		t.Fatalf("expected positive score, got %d", progress.Score)
	}
}

func TestStoryProgressInvalidChoice(t *testing.T) {
	teardown := setupScenarioTest(t)
	defer teardown()

	s := buildSampleScenario(t)
	progress, err := GetOrCreateStoryProgress(99, s.Id)
	if err != nil {
		t.Fatalf("GetOrCreateStoryProgress: %v", err)
	}

	_, err = AdvanceStoryProgress(&progress, 999999)
	if err != ErrInvalidChoice {
		t.Fatalf("expected ErrInvalidChoice, got %v", err)
	}
}

func TestStoryProgressResumes(t *testing.T) {
	teardown := setupScenarioTest(t)
	defer teardown()

	s := buildSampleScenario(t)

	p1, err := GetOrCreateStoryProgress(7, s.Id)
	if err != nil {
		t.Fatalf("first GetOrCreateStoryProgress: %v", err)
	}
	// Ask again — should return the same in-progress record.
	p2, err := GetOrCreateStoryProgress(7, s.Id)
	if err != nil {
		t.Fatalf("second GetOrCreateStoryProgress: %v", err)
	}
	if p1.Id != p2.Id {
		t.Fatalf("expected same progress record, got %d vs %d", p1.Id, p2.Id)
	}
}

func TestDeleteStoryScenarioCascades(t *testing.T) {
	teardown := setupScenarioTest(t)
	defer teardown()

	s := buildSampleScenario(t)
	if err := DeleteStoryScenario(s.Id); err != nil {
		t.Fatalf("DeleteStoryScenario: %v", err)
	}

	if _, err := GetStoryScenario(s.Id); err == nil {
		t.Fatal("expected scenario lookup to fail after delete")
	}
	// Nodes and choices should be gone too.
	var nodeCount int
	db.Table("story_nodes").Where(queryWhereScenarioID, s.Id).Count(&nodeCount)
	if nodeCount != 0 {
		t.Fatalf("expected 0 nodes after delete, got %d", nodeCount)
	}
}

// ---------- QuizQuestion grading ----------

func TestGradeAnswerMultipleChoice(t *testing.T) {
	q := QuizQuestion{
		QuestionType:  QuestionTypeMultipleChoice,
		CorrectOption: 2,
	}
	if !q.GradeAnswer([]int{2}) {
		t.Fatal("expected correct grading for exact match")
	}
	if q.GradeAnswer([]int{1}) {
		t.Fatal("expected wrong answer to be graded false")
	}
	if q.GradeAnswer([]int{}) {
		t.Fatal("expected empty answer to be graded false")
	}
	if q.GradeAnswer([]int{2, 3}) {
		t.Fatal("expected multi-answer on single-choice question to be false")
	}
}

func TestGradeAnswerTrueFalse(t *testing.T) {
	q := QuizQuestion{
		QuestionType:  QuestionTypeTrueFalse,
		CorrectOption: 0,
	}
	if !q.GradeAnswer([]int{0}) {
		t.Fatal("expected True to be correct")
	}
	if q.GradeAnswer([]int{1}) {
		t.Fatal("expected False to be wrong")
	}
}

func TestGradeAnswerMultiSelect(t *testing.T) {
	q := QuizQuestion{
		QuestionType:   QuestionTypeMultiSelect,
		CorrectOptions: "[0,2]",
	}
	if !q.GradeAnswer([]int{0, 2}) {
		t.Fatal("expected exact set match to be correct")
	}
	if !q.GradeAnswer([]int{2, 0}) {
		t.Fatal("expected unordered exact set to be correct")
	}
	if q.GradeAnswer([]int{0}) {
		t.Fatal("expected partial answer to be wrong")
	}
	if q.GradeAnswer([]int{0, 2, 3}) {
		t.Fatal("expected superset answer to be wrong")
	}
	if q.GradeAnswer([]int{1, 2}) {
		t.Fatal("expected mismatched set to be wrong")
	}
}

func TestNormalizeQuestionType(t *testing.T) {
	cases := map[string]string{
		"":                         QuestionTypeMultipleChoice,
		"unknown":                  QuestionTypeMultipleChoice,
		QuestionTypeMultipleChoice: QuestionTypeMultipleChoice,
		QuestionTypeTrueFalse:      QuestionTypeTrueFalse,
		QuestionTypeMultiSelect:    QuestionTypeMultiSelect,
	}
	for input, want := range cases {
		if got := NormalizeQuestionType(input); got != want {
			t.Fatalf("NormalizeQuestionType(%q) = %q, want %q", input, got, want)
		}
	}
}
