package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
)

// Shared WHERE-clause fragments for satisfaction queries.
const (
	whereJoinedOrgID      = "p.org_id = ?"
	wherePresentationIDEq = "presentation_id = ?"
)

// TrainingSatisfactionRating stores a user's rating for a completed training session.
// This mirrors Phished's user satisfaction tracking (4.7/5 average).
type TrainingSatisfactionRating struct {
	Id             int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId         int64     `json:"user_id" gorm:"column:user_id"`
	PresentationId int64     `json:"presentation_id" gorm:"column:presentation_id"`
	Rating         int       `json:"rating" gorm:"column:rating"`     // 1-5 stars
	Feedback       string    `json:"feedback" gorm:"column:feedback"` // Optional text feedback
	CreatedDate    time.Time `json:"created_date" gorm:"column:created_date"`
}

// TableName overrides the default table name.
func (TrainingSatisfactionRating) TableName() string {
	return "training_satisfaction_ratings"
}

// PostTrainingSatisfactionRating saves a user's rating for a training session.
// One rating per user per presentation; updates if already exists.
func PostTrainingSatisfactionRating(r *TrainingSatisfactionRating) error {
	if r.Rating < 1 {
		r.Rating = 1
	}
	if r.Rating > 5 {
		r.Rating = 5
	}
	r.CreatedDate = time.Now().UTC()

	// Upsert: update if exists
	existing := TrainingSatisfactionRating{}
	err := db.Where("user_id = ? AND presentation_id = ?", r.UserId, r.PresentationId).First(&existing).Error
	if err == nil {
		existing.Rating = r.Rating
		existing.Feedback = r.Feedback
		existing.CreatedDate = r.CreatedDate
		return db.Save(&existing).Error
	}
	return db.Save(r).Error
}

// GetSatisfactionRating returns the user's rating for a specific presentation.
func GetSatisfactionRating(userId, presentationId int64) (*TrainingSatisfactionRating, error) {
	r := TrainingSatisfactionRating{}
	err := db.Where("user_id = ? AND presentation_id = ?", userId, presentationId).First(&r).Error
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// SatisfactionStats holds aggregate satisfaction metrics.
type SatisfactionStats struct {
	TotalRatings int     `json:"total_ratings"`
	AverageScore float64 `json:"average_score"`
	Star5Count   int     `json:"star_5_count"`
	Star4Count   int     `json:"star_4_count"`
	Star3Count   int     `json:"star_3_count"`
	Star2Count   int     `json:"star_2_count"`
	Star1Count   int     `json:"star_1_count"`
}

// GetOrgSatisfactionStats returns aggregate satisfaction stats for an organization.
func GetOrgSatisfactionStats(orgId int64) SatisfactionStats {
	stats := SatisfactionStats{}

	// Get average and total
	row := db.Table("training_satisfaction_ratings r").
		Joins("JOIN training_presentations p ON p.id = r.presentation_id").
		Where(whereJoinedOrgID, orgId).
		Select("COUNT(*) as total, COALESCE(AVG(r.rating), 0) as avg_score").Row()
	if row != nil {
		var total int
		var avg float64
		if err := row.Scan(&total, &avg); err == nil {
			stats.TotalRatings = total
			stats.AverageScore = avg
		}
	}

	// Get distribution
	for rating := 1; rating <= 5; rating++ {
		var count int
		db.Table("training_satisfaction_ratings r").
			Joins("JOIN training_presentations p ON p.id = r.presentation_id").
			Where("p.org_id = ? AND r.rating = ?", orgId, rating).
			Count(&count)
		switch rating {
		case 1:
			stats.Star1Count = count
		case 2:
			stats.Star2Count = count
		case 3:
			stats.Star3Count = count
		case 4:
			stats.Star4Count = count
		case 5:
			stats.Star5Count = count
		}
	}

	return stats
}

// GetPresentationSatisfactionStats returns satisfaction stats for a single presentation.
func GetPresentationSatisfactionStats(presentationId int64) SatisfactionStats {
	stats := SatisfactionStats{}

	row := db.Table("training_satisfaction_ratings").
		Where(wherePresentationIDEq, presentationId).
		Select("COUNT(*) as total, COALESCE(AVG(rating), 0) as avg_score").Row()
	if row != nil {
		var total int
		var avg float64
		if err := row.Scan(&total, &avg); err == nil {
			stats.TotalRatings = total
			stats.AverageScore = avg
		}
	}

	// Get star distribution
	for rating := 1; rating <= 5; rating++ {
		var count int
		db.Table("training_satisfaction_ratings").
			Where("presentation_id = ? AND rating = ?", presentationId, rating).
			Count(&count)
		switch rating {
		case 1:
			stats.Star1Count = count
		case 2:
			stats.Star2Count = count
		case 3:
			stats.Star3Count = count
		case 4:
			stats.Star4Count = count
		case 5:
			stats.Star5Count = count
		}
	}

	return stats
}

// TrainingAnalyticsSummary is a comprehensive training analytics response.
type TrainingAnalyticsSummary struct {
	TotalCourses         int                  `json:"total_courses"`
	TotalEnrollments     int                  `json:"total_enrollments"`
	CompletionRate       float64              `json:"completion_rate"`
	AvgCompletionMinutes float64              `json:"avg_completion_minutes"`
	QuizPassRate         float64              `json:"quiz_pass_rate"`
	Satisfaction         SatisfactionStats    `json:"satisfaction"`
	TopCourses           []CourseStats        `json:"top_courses"`
	CompletionTrend      []TrainingTrendPoint `json:"completion_trend"`
}

// CourseStats holds per-course metrics.
type CourseStats struct {
	PresentationId int64   `json:"presentation_id"`
	Name           string  `json:"name"`
	Enrollments    int     `json:"enrollments"`
	Completions    int     `json:"completions"`
	CompletionRate float64 `json:"completion_rate"`
	AvgRating      float64 `json:"avg_rating"`
}

// TrainingTrendPoint is a date-value pair for training trend charts.
type TrainingTrendPoint struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
}

// GetTrainingAnalytics returns comprehensive training analytics for an org.
func GetTrainingAnalytics(orgId int64) TrainingAnalyticsSummary {
	summary := TrainingAnalyticsSummary{}

	// Total courses
	db.Table("training_presentations").Where("org_id = ?", orgId).Count(&summary.TotalCourses)

	// Total enrollments (course_progress records)
	db.Table("course_progress cp").
		Joins("JOIN training_presentations p ON p.id = cp.presentation_id").
		Where(whereJoinedOrgID, orgId).
		Count(&summary.TotalEnrollments)

	// Completions
	var completions int
	db.Table("course_progress cp").
		Joins("JOIN training_presentations p ON p.id = cp.presentation_id").
		Where("p.org_id = ? AND cp.status = 'complete'", orgId).
		Count(&completions)
	if summary.TotalEnrollments > 0 {
		summary.CompletionRate = float64(completions) / float64(summary.TotalEnrollments) * 100.0
	}

	// Quiz pass rate
	var totalAttempts, passedAttempts int
	db.Table("quiz_attempts qa").
		Joins("JOIN quizzes q ON q.id = qa.quiz_id").
		Joins("JOIN training_presentations p ON p.id = q.presentation_id").
		Where(whereJoinedOrgID, orgId).
		Count(&totalAttempts)
	db.Table("quiz_attempts qa").
		Joins("JOIN quizzes q ON q.id = qa.quiz_id").
		Joins("JOIN training_presentations p ON p.id = q.presentation_id").
		Where("p.org_id = ? AND qa.passed = 1", orgId).
		Count(&passedAttempts)
	if totalAttempts > 0 {
		summary.QuizPassRate = float64(passedAttempts) / float64(totalAttempts) * 100.0
	}

	// Satisfaction
	summary.Satisfaction = GetOrgSatisfactionStats(orgId)

	// Top courses by completion
	presentations := []TrainingPresentation{}
	db.Where("org_id = ?", orgId).Order("name asc").Find(&presentations)
	for _, p := range presentations {
		var enrollments, completions int
		db.Table("course_progress").Where(wherePresentationIDEq, p.Id).Count(&enrollments)
		db.Table("course_progress").Where("presentation_id = ? AND status = 'complete'", p.Id).Count(&completions)

		var avgRating float64
		row := db.Table("training_satisfaction_ratings").
			Where(wherePresentationIDEq, p.Id).
			Select("COALESCE(AVG(rating), 0)").Row()
		if row != nil {
			row.Scan(&avgRating)
		}

		rate := 0.0
		if enrollments > 0 {
			rate = float64(completions) / float64(enrollments) * 100.0
		}
		summary.TopCourses = append(summary.TopCourses, CourseStats{
			PresentationId: p.Id,
			Name:           p.Name,
			Enrollments:    enrollments,
			Completions:    completions,
			CompletionRate: rate,
			AvgRating:      avgRating,
		})
	}

	log.Infof("Training analytics computed for org %d: %d courses, %d enrollments, %.1f%% completion",
		orgId, summary.TotalCourses, summary.TotalEnrollments, summary.CompletionRate)

	return summary
}
