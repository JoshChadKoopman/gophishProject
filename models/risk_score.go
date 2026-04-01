package models

import (
	"math"
	"sort"
	"time"

	log "github.com/gophish/gophish/logger"
)

// BRS component weights (must sum to 1.0)
const (
	WeightSimulation  = 0.40
	WeightAcademy     = 0.20
	WeightQuiz        = 0.15
	WeightTrend       = 0.15
	WeightConsistency = 0.10

	// DateFormat is the standard date layout used for BRS history and trend queries.
	DateFormat = "2006-01-02"
)

// UserRiskScoreRecord is the materialized BRS stored in the database.
type UserRiskScoreRecord struct {
	Id              int64     `json:"id" gorm:"primary_key"`
	UserId          int64     `json:"user_id" gorm:"unique"`
	OrgId           int64     `json:"org_id"`
	SimulationScore float64   `json:"simulation_score"`
	AcademyScore    float64   `json:"academy_score"`
	QuizScore       float64   `json:"quiz_score"`
	TrendScore      float64   `json:"trend_score"`
	ConsistencyScore float64  `json:"consistency_score"`
	CompositeScore  float64   `json:"composite_score"`
	Percentile      float64   `json:"percentile"`
	LastCalculated  time.Time `json:"last_calculated"`
}

// TableName overrides GORM's default table name.
func (UserRiskScoreRecord) TableName() string {
	return "user_risk_scores"
}

// DepartmentRiskScore is the aggregated BRS per department.
type DepartmentRiskScore struct {
	Id              int64     `json:"id" gorm:"primary_key"`
	OrgId           int64     `json:"org_id"`
	Department      string    `json:"department"`
	CompositeScore  float64   `json:"composite_score"`
	UserCount       int       `json:"user_count"`
	LastCalculated  time.Time `json:"last_calculated"`
}

// BRSHistoryPoint stores a historical composite score for trend analysis.
type BRSHistoryPoint struct {
	Id              int64     `json:"id" gorm:"primary_key"`
	UserId          int64     `json:"user_id"`
	CompositeScore  float64   `json:"composite_score"`
	CalculatedDate  time.Time `json:"calculated_date"`
}

// TableName overrides GORM's default table name.
func (BRSHistoryPoint) TableName() string {
	return "brs_history"
}

// BRSUserDetail is the API response for a single user's BRS breakdown.
type BRSUserDetail struct {
	UserId           int64   `json:"user_id"`
	Email            string  `json:"email"`
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	Department       string  `json:"department"`
	SimulationScore  float64 `json:"simulation_score"`
	AcademyScore     float64 `json:"academy_score"`
	QuizScore        float64 `json:"quiz_score"`
	TrendScore       float64 `json:"trend_score"`
	ConsistencyScore float64 `json:"consistency_score"`
	CompositeScore   float64 `json:"composite_score"`
	Percentile       float64 `json:"percentile"`
	LastCalculated   string  `json:"last_calculated"`
}

// BRSTrendPoint is a point on the BRS trend chart.
type BRSTrendPoint struct {
	Date           string  `json:"date"`
	CompositeScore float64 `json:"composite_score"`
}

// BRSBenchmark holds org-level and global benchmark data.
type BRSBenchmark struct {
	OrgAvgScore    float64 `json:"org_avg_score"`
	OrgMedian      float64 `json:"org_median_score"`
	OrgUserCount   int     `json:"org_user_count"`
	GlobalAvgScore float64 `json:"global_avg_score"`
	GlobalMedian   float64 `json:"global_median_score"`
}

// CalculateUserBRS computes the 5-factor BRS for a single user and persists it.
func CalculateUserBRS(userId int64) (*UserRiskScoreRecord, error) {
	user, err := GetUser(userId)
	if err != nil {
		return nil, err
	}

	sim := calcSimulationScore(userId)
	acad := calcAcademyScore(userId)
	quiz := calcQuizScore(userId)
	trend := calcTrendScore(userId)
	cons := calcConsistencyScore(userId)

	composite := sim*WeightSimulation + acad*WeightAcademy + quiz*WeightQuiz + trend*WeightTrend + cons*WeightConsistency
	composite = math.Round(composite*100) / 100

	now := time.Now().UTC()
	record := UserRiskScoreRecord{
		UserId:           userId,
		OrgId:            user.OrgId,
		SimulationScore:  math.Round(sim*100) / 100,
		AcademyScore:     math.Round(acad*100) / 100,
		QuizScore:        math.Round(quiz*100) / 100,
		TrendScore:       math.Round(trend*100) / 100,
		ConsistencyScore: math.Round(cons*100) / 100,
		CompositeScore:   composite,
		LastCalculated:   now,
	}

	// Upsert: try update first, then create
	existing := UserRiskScoreRecord{}
	if db.Where("user_id = ?", userId).First(&existing).RecordNotFound() {
		err = db.Create(&record).Error
	} else {
		record.Id = existing.Id
		err = db.Save(&record).Error
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Record history point (one per day max)
	today := now.Format(DateFormat)
	var histCount int
	db.Model(&BRSHistoryPoint{}).Where("user_id = ? AND calculated_date = ?", userId, today).Count(&histCount)
	if histCount == 0 {
		if err := db.Create(&BRSHistoryPoint{
			UserId:         userId,
			CompositeScore: composite,
			CalculatedDate: now,
		}).Error; err != nil {
			log.Errorf("BRS: failed to create history point for user %d: %v", userId, err)
		}
	} else {
		db.Model(&BRSHistoryPoint{}).Where("user_id = ? AND calculated_date = ?", userId, today).
			Update("composite_score", composite)
	}

	return &record, nil
}

// RecalculateOrgBRS recalculates BRS for all users in an org and updates
// department scores and percentiles.
func RecalculateOrgBRS(orgId int64) error {
	var users []User
	err := db.Where("org_id = ?", orgId).Find(&users).Error
	if err != nil {
		return err
	}

	scores := make([]float64, 0, len(users))
	for _, u := range users {
		record, calcErr := CalculateUserBRS(u.Id)
		if calcErr != nil {
			log.Errorf("BRS calc failed for user %d: %v", u.Id, calcErr)
			continue
		}
		scores = append(scores, record.CompositeScore)
	}

	// Update percentiles
	sort.Float64s(scores)
	for _, u := range users {
		var rec UserRiskScoreRecord
		if db.Where("user_id = ?", u.Id).First(&rec).RecordNotFound() {
			continue
		}
		pct := percentileRank(scores, rec.CompositeScore)
		db.Model(&rec).Update("percentile", pct)
	}

	// Update department scores
	updateDepartmentScores(orgId)

	return nil
}

// RecalculateAllBRS recalculates BRS for all orgs.
func RecalculateAllBRS() error {
	var orgIds []int64
	db.Model(&Organization{}).Pluck("id", &orgIds)
	for _, oid := range orgIds {
		if err := RecalculateOrgBRS(oid); err != nil {
			log.Errorf("BRS recalc failed for org %d: %v", oid, err)
		}
	}
	return nil
}

// GetUserBRS returns the materialized BRS detail for a user.
func GetUserBRS(userId int64) (BRSUserDetail, error) {
	var rec UserRiskScoreRecord
	err := db.Where("user_id = ?", userId).First(&rec).Error
	if err != nil {
		return BRSUserDetail{}, err
	}
	user, userErr := GetUser(userId)
	if userErr != nil {
		log.Errorf("BRS: failed to load user %d: %v", userId, userErr)
	}
	return BRSUserDetail{
		UserId:           user.Id,
		Email:            user.Email,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
		Department:       user.Department,
		SimulationScore:  rec.SimulationScore,
		AcademyScore:     rec.AcademyScore,
		QuizScore:        rec.QuizScore,
		TrendScore:       rec.TrendScore,
		ConsistencyScore: rec.ConsistencyScore,
		CompositeScore:   rec.CompositeScore,
		Percentile:       rec.Percentile,
		LastCalculated:   rec.LastCalculated.Format(time.RFC3339),
	}, nil
}

// GetDepartmentBRS returns department-level BRS for an org.
func GetDepartmentBRS(scope OrgScope) ([]DepartmentRiskScore, error) {
	var scores []DepartmentRiskScore
	q := db.Table("department_risk_scores")
	q = scopeQuery(q, scope)
	err := q.Order("composite_score DESC").Find(&scores).Error
	return scores, err
}

// GetBRSBenchmark returns the org and global benchmark.
func GetBRSBenchmark(orgId int64) (BRSBenchmark, error) {
	bench := BRSBenchmark{}

	// Org stats
	var orgScores []float64
	db.Model(&UserRiskScoreRecord{}).Where("org_id = ?", orgId).Pluck("composite_score", &orgScores)
	bench.OrgUserCount = len(orgScores)
	if len(orgScores) > 0 {
		bench.OrgAvgScore = math.Round(avg(orgScores)*100) / 100
		bench.OrgMedian = math.Round(median(orgScores)*100) / 100
	}

	// Global stats
	var globalScores []float64
	db.Model(&UserRiskScoreRecord{}).Pluck("composite_score", &globalScores)
	if len(globalScores) > 0 {
		bench.GlobalAvgScore = math.Round(avg(globalScores)*100) / 100
		bench.GlobalMedian = math.Round(median(globalScores)*100) / 100
	}

	return bench, nil
}

// GetBRSTrend returns historical BRS data points for a user over the given number of days.
func GetBRSTrend(userId int64, days int) ([]BRSTrendPoint, error) {
	if days <= 0 {
		days = 90
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	var history []BRSHistoryPoint
	err := db.Where("user_id = ? AND calculated_date >= ?", userId, cutoff).
		Order("calculated_date ASC").Find(&history).Error
	if err != nil {
		return nil, err
	}
	points := make([]BRSTrendPoint, 0, len(history))
	for _, h := range history {
		points = append(points, BRSTrendPoint{
			Date:           h.CalculatedDate.Format(DateFormat),
			CompositeScore: h.CompositeScore,
		})
	}
	return points, nil
}

// GetBRSLeaderboard returns the top N users by composite BRS in an org.
func GetBRSLeaderboard(scope OrgScope, limit int) ([]BRSUserDetail, error) {
	if limit <= 0 {
		limit = 25
	}
	var records []UserRiskScoreRecord
	q := db.Table("user_risk_scores")
	q = scopeQuery(q, scope)
	err := q.Order("composite_score ASC").Limit(limit).Find(&records).Error
	if err != nil {
		return nil, err
	}

	details := make([]BRSUserDetail, 0, len(records))
	for _, rec := range records {
		user, uErr := GetUser(rec.UserId)
		if uErr != nil {
			continue
		}
		details = append(details, BRSUserDetail{
			UserId:           user.Id,
			Email:            user.Email,
			FirstName:        user.FirstName,
			LastName:         user.LastName,
			Department:       user.Department,
			SimulationScore:  rec.SimulationScore,
			AcademyScore:     rec.AcademyScore,
			QuizScore:        rec.QuizScore,
			TrendScore:       rec.TrendScore,
			ConsistencyScore: rec.ConsistencyScore,
			CompositeScore:   rec.CompositeScore,
			Percentile:       rec.Percentile,
			LastCalculated:   rec.LastCalculated.Format(time.RFC3339),
		})
	}
	return details, nil
}

// --- Component score calculators ---

// calcSimulationScore: 0 = highest risk (always clicks), 100 = no clicks/submissions.
// Based on click rate, submit rate, and report rate from campaign results.
func calcSimulationScore(userId int64) float64 {
	user, err := GetUser(userId)
	if err != nil {
		return 50
	}

	type simRow struct {
		Total     int64
		Clicked   int64
		Submitted int64
		Reported  int64
	}
	var row simRow
	err = db.Raw(`
		SELECT COUNT(*) as total,
			SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as clicked,
			SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END) as submitted,
			SUM(CASE WHEN r.reported = 1 THEN 1 ELSE 0 END) as reported
		FROM results r
		JOIN campaigns c ON r.campaign_id = c.id
		WHERE c.org_id = ? AND r.email = ?
	`, EventClicked, EventDataSubmit, EventDataSubmit, user.OrgId, user.Email).Scan(&row).Error
	if err != nil || row.Total == 0 {
		return 50 // Neutral if no data
	}

	clickRate := float64(row.Clicked) / float64(row.Total)
	submitRate := float64(row.Submitted) / float64(row.Total)
	reportRate := float64(row.Reported) / float64(row.Total)

	// Lower click/submit = better score; higher report = better score
	score := (1 - clickRate*0.4 - submitRate*0.4 + reportRate*0.2) * 100
	return clamp(score, 0, 100)
}

// calcAcademyScore: percentage of completed training assignments.
func calcAcademyScore(userId int64) float64 {
	var total, completed int
	db.Model(&CourseAssignment{}).Where("user_id = ?", userId).Count(&total)
	if total == 0 {
		return 50 // Neutral
	}
	db.Model(&CourseAssignment{}).Where("user_id = ? AND status = ?", userId, AssignmentStatusCompleted).Count(&completed)
	return clamp(float64(completed)/float64(total)*100, 0, 100)
}

// calcQuizScore: average quiz pass percentage.
func calcQuizScore(userId int64) float64 {
	type avgRow struct {
		AvgPct float64
	}
	var row avgRow
	err := db.Raw(`
		SELECT COALESCE(AVG(CASE WHEN total_questions > 0 THEN score * 100.0 / total_questions ELSE 0 END), 0) as avg_pct
		FROM quiz_attempts
		WHERE user_id = ?
	`, userId).Scan(&row).Error
	if err != nil || row.AvgPct == 0 {
		return 50
	}
	return clamp(row.AvgPct, 0, 100)
}

// calcTrendScore: compares recent simulation performance (last 30 days) vs older (30-90 days).
// Improving = higher score, declining = lower score, no change = 50.
func calcTrendScore(userId int64) float64 {
	user, err := GetUser(userId)
	if err != nil {
		return 50
	}

	type periodRow struct {
		Total   int64
		Clicked int64
	}

	now := time.Now()
	recent30 := now.AddDate(0, 0, -30).Format(DateFormat)
	older90 := now.AddDate(0, 0, -90).Format(DateFormat)

	var recent, older periodRow
	db.Raw(`
		SELECT COUNT(*) as total,
			SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as clicked
		FROM results r JOIN campaigns c ON r.campaign_id = c.id
		WHERE c.org_id = ? AND r.email = ? AND DATE(c.created_date) >= ?
	`, EventClicked, EventDataSubmit, user.OrgId, user.Email, recent30).Scan(&recent)

	db.Raw(`
		SELECT COUNT(*) as total,
			SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as clicked
		FROM results r JOIN campaigns c ON r.campaign_id = c.id
		WHERE c.org_id = ? AND r.email = ? AND DATE(c.created_date) >= ? AND DATE(c.created_date) < ?
	`, EventClicked, EventDataSubmit, user.OrgId, user.Email, older90, recent30).Scan(&older)

	if recent.Total == 0 || older.Total == 0 {
		return 50
	}

	recentRate := float64(recent.Clicked) / float64(recent.Total)
	olderRate := float64(older.Clicked) / float64(older.Total)
	improvement := olderRate - recentRate // positive = improving

	// Map -1..+1 improvement to 0..100
	return clamp(50+improvement*50, 0, 100)
}

// calcConsistencyScore: regularity of training engagement over the last 90 days.
// Measures how many of the last 12 weeks had at least one training activity.
func calcConsistencyScore(userId int64) float64 {
	cutoff := time.Now().AddDate(0, 0, -84).Format(DateFormat) // 12 weeks
	type weekRow struct {
		Week string
	}
	var weeks []weekRow

	// Count distinct weeks with quiz attempts or assignment completions
	db.Raw(`
		SELECT DISTINCT strftime('%%Y-%%W', created_date) as week
		FROM quiz_attempts WHERE user_id = ? AND DATE(created_date) >= ?
		UNION
		SELECT DISTINCT strftime('%%Y-%%W', modified_date) as week
		FROM course_assignments WHERE user_id = ? AND status = ? AND DATE(modified_date) >= ?
	`, userId, cutoff, userId, AssignmentStatusCompleted, cutoff).Scan(&weeks)

	activeWeeks := len(weeks)
	if activeWeeks > 12 {
		activeWeeks = 12
	}
	return clamp(float64(activeWeeks)/12.0*100, 0, 100)
}

// --- Helpers ---

func updateDepartmentScores(orgId int64) {
	type deptRow struct {
		Department string
		AvgScore   float64
		UserCount  int
	}
	var rows []deptRow
	db.Raw(`
		SELECT u.department, AVG(urs.composite_score) as avg_score, COUNT(*) as user_count
		FROM user_risk_scores urs
		JOIN users u ON urs.user_id = u.id
		WHERE urs.org_id = ? AND u.department != ''
		GROUP BY u.department
	`, orgId).Scan(&rows)

	now := time.Now().UTC()
	for _, r := range rows {
		var existing DepartmentRiskScore
		if db.Where("org_id = ? AND department = ?", orgId, r.Department).First(&existing).RecordNotFound() {
			if err := db.Create(&DepartmentRiskScore{
				OrgId:          orgId,
				Department:     r.Department,
				CompositeScore: math.Round(r.AvgScore*100) / 100,
				UserCount:      r.UserCount,
				LastCalculated: now,
			}).Error; err != nil {
				log.Errorf("BRS: failed to create department score for %s: %v", r.Department, err)
			}
		} else {
			db.Model(&existing).Updates(map[string]interface{}{
				"composite_score": math.Round(r.AvgScore*100) / 100,
				"user_count":      r.UserCount,
				"last_calculated": now,
			})
		}
	}
}

func percentileRank(sorted []float64, value float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	count := 0
	for _, v := range sorted {
		if v < value {
			count++
		}
	}
	return math.Round(float64(count)/float64(len(sorted))*10000) / 100
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func median(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := make([]float64, n)
	copy(sorted, values)
	sort.Float64s(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
