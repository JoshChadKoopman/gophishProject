package context

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSetAndGet verifies that a value stored via Set can be retrieved via Get.
func TestSetAndGet(t *testing.T) {
	type ctxKey string
	key := ctxKey("user_id")
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Before setting, Get should return nil
	val := Get(req, key)
	if val != nil {
		t.Fatalf("expected nil before Set, got %v", val)
	}

	// Set a value and verify Get returns it
	req = Set(req, key, int64(42))
	val = Get(req, key)
	if val == nil {
		t.Fatal("expected non-nil after Set, got nil")
	}
	if val.(int64) != 42 {
		t.Fatalf("expected 42, got %v", val)
	}
}

// TestSetNilValueReturnsOriginalRequest verifies that passing nil value
// returns the original request without modification.
func TestSetNilValueReturnsOriginalRequest(t *testing.T) {
	type ctxKey string
	key := ctxKey("token")
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	reqAfter := Set(req, key, nil)
	if reqAfter != req {
		t.Fatal("expected same request pointer when value is nil")
	}
}

// TestSetOverwrite verifies that setting a new value for the same key
// overwrites the old one.
func TestSetOverwrite(t *testing.T) {
	type ctxKey string
	key := ctxKey("role")
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	req = Set(req, key, "admin")
	req = Set(req, key, "user")

	val := Get(req, key)
	if val == nil || val.(string) != "user" {
		t.Fatalf("expected 'user' after overwrite, got %v", val)
	}
}

// TestMultipleKeys verifies that multiple different keys coexist.
func TestMultipleKeys(t *testing.T) {
	type ctxKey string
	keyA := ctxKey("a")
	keyB := ctxKey("b")
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	req = Set(req, keyA, "alpha")
	req = Set(req, keyB, "beta")

	if Get(req, keyA).(string) != "alpha" {
		t.Fatal("key A mismatch")
	}
	if Get(req, keyB).(string) != "beta" {
		t.Fatal("key B mismatch")
	}
}

// TestClearIsNoop verifies that Clear does not panic or corrupt state.
func TestClearIsNoop(t *testing.T) {
	type ctxKey string
	key := ctxKey("data")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = Set(req, key, "value")

	// Clear should not panic
	Clear(req)

	// In Go >= 1.7, Clear is a noop, so the value should still be there.
	val := Get(req, key)
	if val == nil || val.(string) != "value" {
		t.Fatalf("expected 'value' after Clear (noop), got %v", val)
	}
}

// TestGetMissingKeyReturnsNil verifies that retrieving a never-set key
// returns nil.
func TestGetMissingKeyReturnsNil(t *testing.T) {
	type ctxKey string
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	val := Get(req, ctxKey("nonexistent"))
	if val != nil {
		t.Fatalf("expected nil for missing key, got %v", val)
	}
}

// TestSetWithDifferentValueTypes verifies that various Go types can be
// stored and retrieved.
func TestSetWithDifferentValueTypes(t *testing.T) {
	type ctxKey string
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	req = Set(req, ctxKey("int"), 123)
	req = Set(req, ctxKey("bool"), true)
	req = Set(req, ctxKey("slice"), []string{"a", "b"})

	if Get(req, ctxKey("int")).(int) != 123 {
		t.Fatal("int mismatch")
	}
	if Get(req, ctxKey("bool")).(bool) != true {
		t.Fatal("bool mismatch")
	}
	s := Get(req, ctxKey("slice")).([]string)
	if len(s) != 2 || s[0] != "a" || s[1] != "b" {
		t.Fatal("slice mismatch")
	}
}
