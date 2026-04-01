package auth

import (
	"testing"
)

func TestPasswordPolicy(t *testing.T) {
	candidate := "short"
	got := CheckPasswordPolicy(candidate)
	if got != ErrPasswordTooShort {
		t.Fatalf("unexpected error received. expected %v got %v", ErrPasswordTooShort, got)
	}

	// Must also fail for missing uppercase
	candidate = "alllowercase1"
	got = CheckPasswordPolicy(candidate)
	if got != ErrPasswordNoUpper {
		t.Fatalf("unexpected error received. expected %v got %v", ErrPasswordNoUpper, got)
	}

	// Must fail for missing lowercase
	candidate = "ALLUPPERCASE1"
	got = CheckPasswordPolicy(candidate)
	if got != ErrPasswordNoLower {
		t.Fatalf("unexpected error received. expected %v got %v", ErrPasswordNoLower, got)
	}

	// Must fail for missing digit
	candidate = "NoDigitsHere"
	got = CheckPasswordPolicy(candidate)
	if got != ErrPasswordNoDigit {
		t.Fatalf("unexpected error received. expected %v got %v", ErrPasswordNoDigit, got)
	}

	// Valid password: meets all requirements
	candidate = "ValidPass1ok"
	got = CheckPasswordPolicy(candidate)
	if got != nil {
		t.Fatalf("unexpected error received. expected %v got %v", nil, got)
	}
}

func TestValidatePasswordChange(t *testing.T) {
	newPassword := "ValidPass1ok"
	confirmPassword := "Mismatch1no"
	currentPassword := "CurrentPass1"
	currentHash, err := GeneratePasswordHash(currentPassword)
	if err != nil {
		t.Fatalf("unexpected error generating password hash: %v", err)
	}

	_, got := ValidatePasswordChange(currentHash, newPassword, confirmPassword)
	if got != ErrPasswordMismatch {
		t.Fatalf("unexpected error received. expected %v got %v", ErrPasswordMismatch, got)
	}

	newPassword = currentPassword
	confirmPassword = newPassword
	_, got = ValidatePasswordChange(currentHash, newPassword, confirmPassword)
	if got != ErrReusedPassword {
		t.Fatalf("unexpected error received. expected %v got %v", ErrReusedPassword, got)
	}
}
