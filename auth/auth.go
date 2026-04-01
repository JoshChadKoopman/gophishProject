package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// MaxFailedLogins is the number of consecutive failed login attempts before
// the account is temporarily locked out.
const MaxFailedLogins = 5

// LoginLockoutDuration is the window during which the account remains locked
// after exceeding MaxFailedLogins.
const LoginLockoutDuration = 15 * time.Minute

// MinPasswordLength is the minimum number of characters required in a password
const MinPasswordLength = 10

// APIKeyLength is the length of Gophish API keys
const APIKeyLength = 32

// ErrInvalidPassword is thrown when a user provides an incorrect password.
var ErrInvalidPassword = errors.New("Invalid Password")

// ErrPasswordMismatch is thrown when a user provides a mismatching password
// and confirmation password.
var ErrPasswordMismatch = errors.New("Passwords do not match")

// ErrReusedPassword is thrown when a user attempts to change their password to
// the existing password
var ErrReusedPassword = errors.New("Cannot reuse existing password")

// ErrEmptyPassword is thrown when a user provides a blank password to the register
// or change password functions
var ErrEmptyPassword = errors.New("No password provided")

// ErrPasswordTooShort is thrown when a user provides a password that is less
// than MinPasswordLength
var ErrPasswordTooShort = fmt.Errorf("Password must be at least %d characters", MinPasswordLength)

// ErrPasswordNoUpper is thrown when a password lacks an uppercase letter.
var ErrPasswordNoUpper = errors.New("Password must contain at least one uppercase letter")

// ErrPasswordNoLower is thrown when a password lacks a lowercase letter.
var ErrPasswordNoLower = errors.New("Password must contain at least one lowercase letter")

// ErrPasswordNoDigit is thrown when a password lacks a digit.
var ErrPasswordNoDigit = errors.New("Password must contain at least one digit")

// GenerateSecureKey returns the hex representation of key generated from n
// random bytes
func GenerateSecureKey(n int) string {
	k := make([]byte, n)
	io.ReadFull(rand.Reader, k)
	return fmt.Sprintf("%x", k)
}

// GeneratePasswordHash returns the bcrypt hash for the provided password using
// the default bcrypt cost.
func GeneratePasswordHash(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPasswordPolicy ensures the provided password is valid according to our
// password policy: minimum length plus at least one uppercase letter, one
// lowercase letter, and one digit.
func CheckPasswordPolicy(password string) error {
	switch {
	case password == "":
		return ErrEmptyPassword
	case len(password) < MinPasswordLength:
		return ErrPasswordTooShort
	}
	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}
	if !hasUpper {
		return ErrPasswordNoUpper
	}
	if !hasLower {
		return ErrPasswordNoLower
	}
	if !hasDigit {
		return ErrPasswordNoDigit
	}
	return nil
}

// ValidatePassword validates that the provided password matches the provided
// bcrypt hash.
func ValidatePassword(password string, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ValidatePasswordChange validates that the new password matches the
// configured password policy, that the new password and confirmation
// password match.
//
// Note that this assumes the current password has been confirmed by the
// caller.
//
// If all of the provided data is valid, then the hash of the new password is
// returned.
func ValidatePasswordChange(currentHash, newPassword, confirmPassword string) (string, error) {
	// Ensure the new password passes our password policy
	if err := CheckPasswordPolicy(newPassword); err != nil {
		return "", err
	}
	// Check that new passwords match
	if newPassword != confirmPassword {
		return "", ErrPasswordMismatch
	}
	// Make sure that the new password isn't the same as the old one
	err := ValidatePassword(newPassword, currentHash)
	if err == nil {
		return "", ErrReusedPassword
	}
	// Generate the new hash
	return GeneratePasswordHash(newPassword)
}
