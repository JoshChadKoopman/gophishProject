package models

import (
	"crypto/rand"
	"math/big"
	"time"

	log "github.com/gophish/gophish/logger"
)

// Certificate represents a training course completion certificate.
type Certificate struct {
	Id               int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId           int64     `json:"user_id" gorm:"column:user_id"`
	PresentationId   int64     `json:"presentation_id" gorm:"column:presentation_id"`
	QuizAttemptId    int64     `json:"quiz_attempt_id,omitempty" gorm:"column:quiz_attempt_id"`
	VerificationCode string    `json:"verification_code" gorm:"column:verification_code"`
	IssuedDate       time.Time `json:"issued_date" gorm:"column:issued_date"`
}

const verificationCodeLen = 16
const verificationCodeChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// generateVerificationCode produces a 16-character alphanumeric string using crypto/rand.
func generateVerificationCode() (string, error) {
	result := make([]byte, verificationCodeLen)
	for i := range result {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(verificationCodeChars))))
		if err != nil {
			return "", err
		}
		result[i] = verificationCodeChars[idx.Int64()]
	}
	return string(result), nil
}

// IssueCertificate creates a new certificate for a user's course completion.
// quizAttemptId should be 0 if the course has no quiz.
func IssueCertificate(userId, presentationId, quizAttemptId int64) (*Certificate, error) {
	code, err := generateVerificationCode()
	if err != nil {
		return nil, err
	}
	cert := &Certificate{
		UserId:           userId,
		PresentationId:   presentationId,
		QuizAttemptId:    quizAttemptId,
		VerificationCode: code,
		IssuedDate:       time.Now().UTC(),
	}
	err = db.Save(cert).Error
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return cert, nil
}

// GetCertificate looks up a certificate by its verification code.
func GetCertificate(verificationCode string) (Certificate, error) {
	c := Certificate{}
	err := db.Where("verification_code=?", verificationCode).First(&c).Error
	return c, err
}

// GetCertificatesForUser returns all certificates for a given user.
func GetCertificatesForUser(userId int64) ([]Certificate, error) {
	certs := []Certificate{}
	err := db.Where("user_id=?", userId).Order("issued_date desc").Find(&certs).Error
	return certs, err
}

// GetCertificateForCourse returns the most recent certificate for a user on a specific course.
func GetCertificateForCourse(userId, presentationId int64) (Certificate, error) {
	c := Certificate{}
	err := db.Where("user_id=? AND presentation_id=?", userId, presentationId).
		Order("issued_date desc").First(&c).Error
	return c, err
}
