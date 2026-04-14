package models

import (
	"math"
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

// Test constants to avoid duplicate literal warnings (SonarLint S1192).
const (
	errExpectedNonZeroID = "expected non-zero ID"
	errGetBRSTrend       = "GetBRSTrend failed: %v"
)

// setupRiskScoreTest initialises an in-memory DB for risk score (BRS) tests.
func setupRiskScoreTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM brs_history")
	db.Exec("DELETE FROM user_risk_scores")
	db.Exec("DELETE FROM department_risk_scores")
	return func() {
		db.Exec("DELETE FROM brs_history")
		db.Exec("DELETE FROM user_risk_scores")
		db.Exec("DELETE FROM department_risk_scores")
	}
}

// ---------- Weight constants ----------

func TestBRSWeightsSum(t *testing.T) {
	sum := WeightSimulation + WeightAcademy + WeightQuiz + WeightTrend + WeightConsistency
	if math.Abs(sum-1.0) > 0.0001 {
		t.Fatalf("BRS weights must sum to 1.0, got %f", sum)
	}
}

func TestBRSWeightsPositive(t *testing.T) {
	weights := []float64{WeightSimulation, WeightAcademy, WeightQuiz, WeightTrend, WeightConsistency}
	for _, w := range weights {
		if w <= 0 {
			t.Fatalf("BRS weight must be positive, got %f", w)
		}
	}
}

// ---------- Helper functions ----------

func TestClamp(t *testing.T) {
	tests := []struct {
		v, min, max, expected float64
	}{
		{50, 0, 100, 50},
		{-10, 0, 100, 0},
		{150, 0, 100, 100},
		{0, 0, 100, 0},
		{100, 0, 100, 100},
	}
	for _, tc := range tests {
		got := clamp(tc.v, tc.min, tc.max)
		if got != tc.expected {
			t.Errorf("clamp(%f, %f, %f) = %f, want %f", tc.v, tc.min, tc.max, got, tc.expected)
		}
	}
}

func TestAvg(t *testing.T) {
	tests := []struct {
		values   []float64
		expected float64
	}{
		{[]float64{10, 20, 30}, 20},
		{[]float64{100}, 100},
		{[]float64{}, 0},
	}
	for _, tc := range tests {
		got := avg(tc.values)
		if got != tc.expected {
			t.Errorf("avg(%v) = %f, want %f", tc.values, got, tc.expected)
		}
	}
}

func TestMedian(t *testing.T) {
	tests := []struct {
		values   []float64
		expected float64
	}{
		{[]float64{1, 3, 5}, 3},
		{[]float64{1, 2, 3, 4}, 2.5},
		{[]float64{42}, 42},
		{[]float64{}, 0},
	}
	for _, tc := range tests {
		got := median(tc.values)
		if got != tc.expected {
			t.Errorf("median(%v) = %f, want %f", tc.values, got, tc.expected)
		}
	}
}

func TestPercentileRank(t *testing.T) {
	sorted := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	tests := []struct {
		value    float64
		expected float64
	}{
		{10, 0},   // lowest value, 0% below
		{50, 40},  // 4 out of 10 below
		{100, 90}, // 9 out of 10 below
	}
	for _, tc := range tests {
		got := percentileRank(sorted, tc.value)
		if got != tc.expected {
			t.Errorf("percentileRank(sorted, %f) = %f, want %f", tc.value, got, tc.expected)
		}
	}
}

func TestPercentileRankEmpty(t *testing.T) {
	got := percentileRank([]float64{}, 50)
	if got != 0 {
		t.Fatalf("expected 0 for empty slice, got %f", got)
	}
}

// ---------- UserRiskScoreRecord TableName ----------

func TestUserRiskScoreRecordTableName(t *testing.T) {
	r := UserRiskScoreRecord{}
	if r.TableName() != "user_risk_scores" {
		t.Fatalf("expected table 'user_risk_scores', got %q", r.TableName())
	}
}

func TestBRSHistoryPointTableName(t *testing.T) {
	h := BRSHistoryPoint{}
	if h.TableName() != "brs_history" {
		t.Fatalf("expected table 'brs_history', got %q", h.TableName())
	}
}

// ---------- UserRiskScoreRecord CRUD ----------

func TestUserRiskScoreRecordCreate(t *testing.T) {
	teardown := setupRiskScoreTest(t)
	defer teardown()

	rec := UserRiskScoreRecord{
		UserId:           1,
		OrgId:            1,
		SimulationScore:  75.5,
		AcademyScore:     80.0,
		QuizScore:        90.0,
		TrendScore:       60.0,
		ConsistencyScore: 50.0,
		CompositeScore:   72.3,
		Percentile:       55.0,
		LastCalculated:   time.Now().UTC(),
	}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("failed to create UserRiskScoreRecord: %v", err)
	}
	if rec.Id == 0 {
		t.Fatal(errExpectedNonZeroID)
	}
}

func TestUserRiskScoreRecordQuery(t *testing.T) {
	teardown := setupRiskScoreTest(t)
	defer teardown()

	db.Create(&UserRiskScoreRecord{UserId: 1, OrgId: 1, CompositeScore: 72.3, LastCalculated: time.Now().UTC()})
	db.Create(&UserRiskScoreRecord{UserId: 2, OrgId: 1, CompositeScore: 85.1, LastCalculated: time.Now().UTC()})

	var records []UserRiskScoreRecord
	db.Where("org_id = ?", 1).Find(&records)
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
}

// ---------- BRSHistoryPoint CRUD ----------

func TestBRSHistoryPointCreate(t *testing.T) {
	teardown := setupRiskScoreTest(t)
	defer teardown()

	hp := BRSHistoryPoint{
		UserId:         1,
		CompositeScore: 72.3,
		CalculatedDate: time.Now().UTC(),
	}
	if err := db.Create(&hp).Error; err != nil {
		t.Fatalf("failed to create BRSHistoryPoint: %v", err)
	}
	if hp.Id == 0 {
		t.Fatal(errExpectedNonZeroID)
	}
}

// ---------- GetBRSTrend ----------

func TestGetBRSTrend(t *testing.T) {
	teardown := setupRiskScoreTest(t)
	defer teardown()

	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		db.Create(&BRSHistoryPoint{
			UserId:         1,
			CompositeScore: float64(50 + i*5),
			CalculatedDate: now.AddDate(0, 0, -i*10),
		})
	}

	trend, err := GetBRSTrend(1, 90)
	if err != nil {
		t.Fatalf(errGetBRSTrend, err)
	}
	if len(trend) != 5 {
		t.Fatalf("expected 5 trend points, got %d", len(trend))
	}
}

func TestGetBRSTrendDefaultDays(t *testing.T) {
	teardown := setupRiskScoreTest(t)
	defer teardown()

	db.Create(&BRSHistoryPoint{UserId: 1, CompositeScore: 50, CalculatedDate: time.Now().UTC()})

	// days=0 should default to 90
	trend, err := GetBRSTrend(1, 0)
	if err != nil {
		t.Fatalf(errGetBRSTrend, err)
	}
	if len(trend) != 1 {
		t.Fatalf("expected 1 trend point, got %d", len(trend))
	}
}

func TestGetBRSTrendEmpty(t *testing.T) {
	teardown := setupRiskScoreTest(t)
	defer teardown()

	trend, err := GetBRSTrend(999, 90)
	if err != nil {
		t.Fatalf(errGetBRSTrend, err)
	}
	if len(trend) != 0 {
		t.Fatalf("expected 0 trend points, got %d", len(trend))
	}
}

// ---------- GetBRSBenchmark ----------

func TestGetBRSBenchmark(t *testing.T) {
	teardown := setupRiskScoreTest(t)
	defer teardown()

	db.Create(&UserRiskScoreRecord{UserId: 1, OrgId: 1, CompositeScore: 60, LastCalculated: time.Now().UTC()})
	db.Create(&UserRiskScoreRecord{UserId: 2, OrgId: 1, CompositeScore: 80, LastCalculated: time.Now().UTC()})
	db.Create(&UserRiskScoreRecord{UserId: 3, OrgId: 2, CompositeScore: 90, LastCalculated: time.Now().UTC()})

	bench, err := GetBRSBenchmark(1)
	if err != nil {
		t.Fatalf("GetBRSBenchmark failed: %v", err)
	}
	if bench.OrgUserCount != 2 {
		t.Fatalf("expected OrgUserCount 2, got %d", bench.OrgUserCount)
	}
	if bench.OrgAvgScore != 70 {
		t.Fatalf("expected OrgAvgScore 70, got %f", bench.OrgAvgScore)
	}
	if bench.OrgMedian != 70 { // median of 60,80 = 70
		t.Fatalf("expected OrgMedian 70, got %f", bench.OrgMedian)
	}
	// Global should include all 3
	if bench.GlobalAvgScore == 0 {
		t.Fatal("expected non-zero GlobalAvgScore")
	}
}

func TestGetBRSBenchmarkEmpty(t *testing.T) {
	teardown := setupRiskScoreTest(t)
	defer teardown()

	bench, err := GetBRSBenchmark(999)
	if err != nil {
		t.Fatalf("GetBRSBenchmark failed: %v", err)
	}
	if bench.OrgUserCount != 0 {
		t.Fatalf("expected 0 users, got %d", bench.OrgUserCount)
	}
	if bench.OrgAvgScore != 0 {
		t.Fatalf("expected 0 avg, got %f", bench.OrgAvgScore)
	}
}

// ---------- DepartmentRiskScore CRUD ----------

func TestDepartmentRiskScoreCRUD(t *testing.T) {
	teardown := setupRiskScoreTest(t)
	defer teardown()

	drs := DepartmentRiskScore{
		OrgId:          1,
		Department:     "Engineering",
		CompositeScore: 72.5,
		UserCount:      10,
		LastCalculated: time.Now().UTC(),
	}
	if err := db.Create(&drs).Error; err != nil {
		t.Fatalf("failed to create DepartmentRiskScore: %v", err)
	}
	if drs.Id == 0 {
		t.Fatal(errExpectedNonZeroID)
	}

	var fetched DepartmentRiskScore
	db.Where("org_id = ? AND department = ?", 1, "Engineering").First(&fetched)
	if fetched.CompositeScore != 72.5 {
		t.Fatalf("expected 72.5, got %f", fetched.CompositeScore)
	}
}

// ---------- BRSUserDetail struct ----------

func TestBRSUserDetailFields(t *testing.T) {
	detail := BRSUserDetail{
		UserId:           1,
		Email:            "test@example.com",
		SimulationScore:  75,
		AcademyScore:     80,
		QuizScore:        90,
		TrendScore:       60,
		ConsistencyScore: 50,
		CompositeScore:   72.3,
		Percentile:       55,
	}
	if detail.UserId != 1 {
		t.Fatalf("expected UserId 1, got %d", detail.UserId)
	}
	if detail.CompositeScore != 72.3 {
		t.Fatalf("expected 72.3, got %f", detail.CompositeScore)
	}
}

// ---------- DateFormat constant ----------

func TestDateFormat(t *testing.T) {
	now := time.Now()
	formatted := now.Format(DateFormat)
	if len(formatted) != 10 { // YYYY-MM-DD
		t.Fatalf("expected 10-char date, got %q", formatted)
	}
}
