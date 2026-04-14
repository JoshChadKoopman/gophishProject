package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

// Test constants to avoid duplicate literal warnings (SonarLint S1192).
const (
	errGetAuditLogs         = "GetAuditLogs failed: %v"
	errGetAuditLogsFiltered = "GetAuditLogsFiltered failed: %v"
)

// setupAuditTest initialises an in-memory DB for audit tests.
func setupAuditTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM audit_logs")
	return func() {
		db.Exec("DELETE FROM audit_logs")
	}
}

// ---------- CreateAuditLog ----------

func TestCreateAuditLog(t *testing.T) {
	teardown := setupAuditTest(t)
	defer teardown()

	entry := &AuditLog{
		OrgId:         1,
		ActorID:       1,
		ActorUsername: "admin",
		Action:        AuditActionLoginSuccess,
		IPAddress:     "127.0.0.1",
	}
	if err := CreateAuditLog(entry); err != nil {
		t.Fatalf("CreateAuditLog failed: %v", err)
	}
	if entry.Timestamp.IsZero() {
		t.Fatal("expected Timestamp to be set")
	}
	if entry.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestCreateAuditLogWithTarget(t *testing.T) {
	teardown := setupAuditTest(t)
	defer teardown()

	entry := &AuditLog{
		OrgId:          1,
		ActorID:        1,
		ActorUsername:  "admin",
		Action:         AuditActionUserCreated,
		TargetType:     "user",
		TargetID:       2,
		TargetUsername: "newuser",
		IPAddress:      "10.0.0.1",
	}
	if err := CreateAuditLog(entry); err != nil {
		t.Fatalf("CreateAuditLog failed: %v", err)
	}
	if entry.TargetUsername != "newuser" {
		t.Fatalf("expected TargetUsername 'newuser', got %q", entry.TargetUsername)
	}
}

// ---------- GetAuditLogs ----------

func TestGetAuditLogs(t *testing.T) {
	teardown := setupAuditTest(t)
	defer teardown()

	for i := 0; i < 5; i++ {
		CreateAuditLog(&AuditLog{
			OrgId:         1,
			ActorID:       1,
			ActorUsername: "admin",
			Action:        AuditActionLoginSuccess,
		})
	}

	scope := OrgScope{OrgId: 1}
	logs, err := GetAuditLogs(scope, 0, 0)
	if err != nil {
		t.Fatalf(errGetAuditLogs, err)
	}
	if len(logs) != 5 {
		t.Fatalf("expected 5 logs, got %d", len(logs))
	}
}

func TestGetAuditLogsDefaultLimit(t *testing.T) {
	teardown := setupAuditTest(t)
	defer teardown()

	// The default limit is 100, so inserting 3 should return 3
	for i := 0; i < 3; i++ {
		CreateAuditLog(&AuditLog{OrgId: 1, ActorID: 1, ActorUsername: "admin", Action: AuditActionLoginFailed})
	}
	scope := OrgScope{OrgId: 1}
	logs, err := GetAuditLogs(scope, -1, 0)
	if err != nil {
		t.Fatalf(errGetAuditLogs, err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(logs))
	}
}

func TestGetAuditLogsOrgIsolation(t *testing.T) {
	teardown := setupAuditTest(t)
	defer teardown()

	CreateAuditLog(&AuditLog{OrgId: 1, ActorID: 1, ActorUsername: "admin", Action: AuditActionLoginSuccess})
	CreateAuditLog(&AuditLog{OrgId: 2, ActorID: 2, ActorUsername: "other", Action: AuditActionLoginSuccess})

	scope := OrgScope{OrgId: 1}
	logs, err := GetAuditLogs(scope, 0, 0)
	if err != nil {
		t.Fatalf(errGetAuditLogs, err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log for org 1, got %d", len(logs))
	}
}

// ---------- GetAuditLogsFiltered ----------

func TestGetAuditLogsFilteredByAction(t *testing.T) {
	teardown := setupAuditTest(t)
	defer teardown()

	CreateAuditLog(&AuditLog{OrgId: 1, ActorID: 1, ActorUsername: "admin", Action: AuditActionLoginSuccess})
	CreateAuditLog(&AuditLog{OrgId: 1, ActorID: 1, ActorUsername: "admin", Action: AuditActionLoginFailed})
	CreateAuditLog(&AuditLog{OrgId: 1, ActorID: 1, ActorUsername: "admin", Action: AuditActionLoginFailed})

	scope := OrgScope{OrgId: 1}
	resp, err := GetAuditLogsFiltered(scope, 0, 0, AuditActionLoginFailed, "", "", "")
	if err != nil {
		t.Fatalf(errGetAuditLogsFiltered, err)
	}
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
	if len(resp.Logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(resp.Logs))
	}
}

func TestGetAuditLogsFilteredByActor(t *testing.T) {
	teardown := setupAuditTest(t)
	defer teardown()

	CreateAuditLog(&AuditLog{OrgId: 1, ActorID: 1, ActorUsername: "admin", Action: AuditActionUserCreated})
	CreateAuditLog(&AuditLog{OrgId: 1, ActorID: 2, ActorUsername: "manager", Action: AuditActionUserCreated})

	scope := OrgScope{OrgId: 1}
	resp, err := GetAuditLogsFiltered(scope, 0, 0, "", "manager", "", "")
	if err != nil {
		t.Fatalf(errGetAuditLogsFiltered, err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 log for actor 'manager', got %d", resp.Total)
	}
}

func TestGetAuditLogsFilteredPagination(t *testing.T) {
	teardown := setupAuditTest(t)
	defer teardown()

	for i := 0; i < 10; i++ {
		CreateAuditLog(&AuditLog{OrgId: 1, ActorID: 1, ActorUsername: "admin", Action: AuditActionLoginSuccess})
	}

	scope := OrgScope{OrgId: 1}
	resp, err := GetAuditLogsFiltered(scope, 3, 0, "", "", "", "")
	if err != nil {
		t.Fatalf(errGetAuditLogsFiltered, err)
	}
	if resp.Total != 10 {
		t.Fatalf("expected total 10, got %d", resp.Total)
	}
	if len(resp.Logs) != 3 {
		t.Fatalf("expected 3 logs (limit), got %d", len(resp.Logs))
	}
}

// ---------- LogRoleChange ----------

func TestLogRoleChange(t *testing.T) {
	teardown := setupAuditTest(t)
	defer teardown()

	actor := User{Id: 1, OrgId: 1, Username: "admin"}
	target := User{Id: 2, Username: "user1"}
	err := LogRoleChange(actor, target, "user", "admin", "192.168.1.1")
	if err != nil {
		t.Fatalf("LogRoleChange failed: %v", err)
	}

	scope := OrgScope{OrgId: 1}
	logs, _ := GetAuditLogs(scope, 0, 0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Action != AuditActionRoleChange {
		t.Fatalf("expected action %q, got %q", AuditActionRoleChange, logs[0].Action)
	}
	if logs[0].IPAddress != "192.168.1.1" {
		t.Fatalf("expected IP 192.168.1.1, got %q", logs[0].IPAddress)
	}
}

// ---------- LogUserEvent ----------

func TestLogUserEvent(t *testing.T) {
	teardown := setupAuditTest(t)
	defer teardown()

	actor := User{Id: 1, OrgId: 1, Username: "admin"}
	target := User{Id: 2, Username: "user1"}
	err := LogUserEvent(actor, target, AuditActionUserLocked, "10.0.0.5")
	if err != nil {
		t.Fatalf("LogUserEvent failed: %v", err)
	}

	scope := OrgScope{OrgId: 1}
	logs, _ := GetAuditLogs(scope, 0, 0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Action != AuditActionUserLocked {
		t.Fatalf("expected action %q, got %q", AuditActionUserLocked, logs[0].Action)
	}
	if logs[0].TargetType != "user" {
		t.Fatalf("expected target_type 'user', got %q", logs[0].TargetType)
	}
}

// ---------- Audit Action Constants ----------

func TestAuditActionConstants(t *testing.T) {
	actions := []string{
		AuditActionRoleChange,
		AuditActionUserCreated,
		AuditActionUserDeleted,
		AuditActionUserLocked,
		AuditActionUserUnlocked,
		AuditActionLoginSuccess,
		AuditActionLoginFailed,
		AuditActionMFAEnrolled,
		AuditActionMFAVerified,
		AuditActionMFAFailed,
		AuditActionMFALockout,
		AuditActionTrainingAssigned,
		AuditActionTrainingAutoAssigned,
		AuditActionTrainingCompleted,
		AuditActionCertificateIssued,
	}
	for _, a := range actions {
		if a == "" {
			t.Fatal("audit action constant must not be empty")
		}
	}
}
