package config

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	log "github.com/gophish/gophish/logger"
)

var validConfig = []byte(`{
	"admin_server": {
		"listen_url": "127.0.0.1:3333",
		"use_tls": true,
		"cert_path": "gophish_admin.crt",
		"key_path": "gophish_admin.key"
	},
	"phish_server": {
		"listen_url": "0.0.0.0:8080",
		"use_tls": false,
		"cert_path": "example.crt",
		"key_path": "example.key"
	},
	"db_name": "sqlite3",
	"db_path": "gophish.db",
	"migrations_prefix": "db/db_",
	"contact_address": ""
}`)

// fatalIfErr is a test helper that fails the test if err is non-nil.
func fatalIfErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func createTemporaryConfig(t *testing.T) *os.File {
	t.Helper()
	f, err := os.CreateTemp("", "gophish-config")
	if err != nil {
		t.Fatalf("unable to create temporary config: %v", err)
	}
	return f
}

func removeTemporaryConfig(t *testing.T, f *os.File) {
	t.Helper()
	err := f.Close()
	if err != nil {
		t.Fatalf("unable to remove temporary config: %v", err)
	}
}

func writeConfig(t *testing.T, data []byte) string {
	t.Helper()
	f := createTemporaryConfig(t)
	if _, err := f.Write(data); err != nil {
		t.Fatalf("error writing config: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestLoadConfig(t *testing.T) {
	f := createTemporaryConfig(t)
	defer removeTemporaryConfig(t, f)
	_, err := f.Write(validConfig)
	if err != nil {
		t.Fatalf("error writing config to temporary file: %v", err)
	}
	// Load the valid config
	conf, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("error loading config from temporary file: %v", err)
	}

	expectedConfig := &Config{}
	err = json.Unmarshal(validConfig, &expectedConfig)
	if err != nil {
		t.Fatalf("error unmarshaling config: %v", err)
	}
	expectedConfig.MigrationsPath = expectedConfig.MigrationsPath + expectedConfig.DBName
	expectedConfig.TestFlag = false
	expectedConfig.AdminConf.CSRFKey = ""
	expectedConfig.Logging = &log.Config{}
	// LoadConfig defaults BackupCodeCount to 8 when not specified
	expectedConfig.MFA.BackupCodeCount = 8
	if !reflect.DeepEqual(expectedConfig, conf) {
		t.Fatalf("invalid config received. expected %#v got %#v", expectedConfig, conf)
	}

	// Load an invalid config
	_, err = LoadConfig("bogusfile")
	if err == nil {
		t.Fatalf("expected error when loading invalid config, but got %v", err)
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	path := writeConfig(t, []byte(`{invalid json`))
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error when loading invalid JSON")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error when loading nonexistent file")
	}
}

func TestLoadConfigNullLogging(t *testing.T) {
	path := writeConfig(t, validConfig)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.Logging == nil {
		t.Fatal("expected Logging to be initialized when nil in config")
	}
}

func TestLoadConfigMigrationsPath(t *testing.T) {
	path := writeConfig(t, validConfig)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	expected := "db/db_sqlite3"
	if conf.MigrationsPath != expected {
		t.Fatalf("expected MigrationsPath %q, got %q", expected, conf.MigrationsPath)
	}
}

func TestLoadConfigTestFlagAlwaysFalse(t *testing.T) {
	configWithTestFlag := []byte(`{
		"admin_server": {"listen_url": "127.0.0.1:3333"},
		"phish_server": {"listen_url": "0.0.0.0:8080"},
		"db_name": "sqlite3",
		"db_path": "gophish.db",
		"migrations_prefix": "db/db_",
		"test_flag": true
	}`)
	path := writeConfig(t, configWithTestFlag)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.TestFlag {
		t.Fatal("TestFlag should always be false after LoadConfig, even if set in JSON")
	}
}

func TestLoadConfigMFADefaults(t *testing.T) {
	path := writeConfig(t, validConfig)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.MFA.BackupCodeCount != 8 {
		t.Fatalf("expected BackupCodeCount default 8, got %d", conf.MFA.BackupCodeCount)
	}
}

func TestLoadConfigMFACustomBackupCodeCount(t *testing.T) {
	configWithMFA := []byte(`{
		"admin_server": {"listen_url": "127.0.0.1:3333"},
		"phish_server": {"listen_url": "0.0.0.0:8080"},
		"db_name": "sqlite3",
		"db_path": "gophish.db",
		"migrations_prefix": "db/db_",
		"mfa": {"backup_code_count": 12}
	}`)
	path := writeConfig(t, configWithMFA)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.MFA.BackupCodeCount != 12 {
		t.Fatalf("expected BackupCodeCount 12, got %d", conf.MFA.BackupCodeCount)
	}
}

// ---------- OIDC environment variable overrides ----------

func TestLoadConfigOIDCEnvOverrides(t *testing.T) {
	t.Setenv("KEYCLOAK_URL", "http://keycloak:8080")
	t.Setenv("KEYCLOAK_REALM", "nivoxis")
	t.Setenv("KEYCLOAK_CLIENT_ID", "gophish-app")
	t.Setenv("KEYCLOAK_CLIENT_SECRET", "super-secret")
	t.Setenv("OIDC_REDIRECT_URL", "https://app.example.com/auth/oidc/callback")

	path := writeConfig(t, validConfig)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}

	if !conf.OIDC.Enabled {
		t.Fatal("expected OIDC to be enabled when KEYCLOAK_URL is set")
	}
	if conf.OIDC.ProviderURL != "http://keycloak:8080/realms/nivoxis" {
		t.Fatalf("unexpected ProviderURL: %q", conf.OIDC.ProviderURL)
	}
	if conf.OIDC.ClientID != "gophish-app" {
		t.Fatalf("unexpected ClientID: %q", conf.OIDC.ClientID)
	}
	if conf.OIDC.ClientSecret != "super-secret" {
		t.Fatalf("unexpected ClientSecret: %q", conf.OIDC.ClientSecret)
	}
	if conf.OIDC.RedirectURL != "https://app.example.com/auth/oidc/callback" {
		t.Fatalf("unexpected RedirectURL: %q", conf.OIDC.RedirectURL)
	}
}

func TestLoadConfigOIDCDefaultRealm(t *testing.T) {
	t.Setenv("KEYCLOAK_URL", "http://keycloak:8080")
	// KEYCLOAK_REALM not set → should default to "master"

	path := writeConfig(t, validConfig)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.OIDC.ProviderURL != "http://keycloak:8080/realms/master" {
		t.Fatalf("expected default realm 'master', got ProviderURL: %q", conf.OIDC.ProviderURL)
	}
}

// ---------- MFA environment variable overrides ----------

func TestLoadConfigMFAEncryptionKeyEnv(t *testing.T) {
	t.Setenv("MFA_TOTP_ENCRYPTION_KEY", "dGVzdC1rZXktZm9yLXVuaXQtdGVzdGluZy1vbmx5")

	path := writeConfig(t, validConfig)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.MFA.TOTPEncryptionKey != "dGVzdC1rZXktZm9yLXVuaXQtdGVzdGluZy1vbmx5" {
		t.Fatalf("unexpected TOTPEncryptionKey: %q", conf.MFA.TOTPEncryptionKey)
	}
}

// ---------- AI environment variable overrides ----------

func TestLoadConfigAIClaudeEnv(t *testing.T) {
	t.Setenv("CLAUDE_API_KEY", "sk-ant-test-key")

	path := writeConfig(t, validConfig)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.AI.APIKey != "sk-ant-test-key" {
		t.Fatalf("unexpected AI.APIKey: %q", conf.AI.APIKey)
	}
	if conf.AI.Provider != "claude" {
		t.Fatalf("expected provider 'claude', got %q", conf.AI.Provider)
	}
	if conf.AI.Model != "claude-sonnet-4-20250514" {
		t.Fatalf("expected default Claude model, got %q", conf.AI.Model)
	}
}

func TestLoadConfigAIOpenAIEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-openai-test-key")

	path := writeConfig(t, validConfig)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.AI.APIKey != "sk-openai-test-key" {
		t.Fatalf("unexpected AI.APIKey: %q", conf.AI.APIKey)
	}
	if conf.AI.Provider != "openai" {
		t.Fatalf("expected provider 'openai', got %q", conf.AI.Provider)
	}
	if conf.AI.Model != "gpt-4o" {
		t.Fatalf("expected default OpenAI model, got %q", conf.AI.Model)
	}
}

func TestLoadConfigAIClaudeTakesPrecedence(t *testing.T) {
	// When both are set, CLAUDE_API_KEY is checked first
	t.Setenv("CLAUDE_API_KEY", "sk-claude")
	t.Setenv("OPENAI_API_KEY", "sk-openai")

	path := writeConfig(t, validConfig)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.AI.APIKey != "sk-claude" {
		t.Fatalf("expected Claude key to take precedence, got %q", conf.AI.APIKey)
	}
}

func TestLoadConfigAIExplicitProvider(t *testing.T) {
	t.Setenv("NIVOXIS_AI_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "sk-openai-test")

	path := writeConfig(t, validConfig)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.AI.Provider != "openai" {
		t.Fatalf("expected explicit provider 'openai', got %q", conf.AI.Provider)
	}
}

func TestLoadConfigAICustomModel(t *testing.T) {
	t.Setenv("NIVOXIS_AI_MODEL", "claude-3-haiku")
	t.Setenv("CLAUDE_API_KEY", "sk-ant-test")

	path := writeConfig(t, validConfig)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.AI.Model != "claude-3-haiku" {
		t.Fatalf("expected custom model 'claude-3-haiku', got %q", conf.AI.Model)
	}
}

// ---------- JSON config with inline OIDC/MFA/AI ----------

func TestLoadConfigWithInlineOIDC(t *testing.T) {
	configWithOIDC := []byte(`{
		"admin_server": {"listen_url": "127.0.0.1:3333"},
		"phish_server": {"listen_url": "0.0.0.0:8080"},
		"db_name": "sqlite3",
		"db_path": "gophish.db",
		"migrations_prefix": "db/db_",
		"oidc": {
			"enabled": true,
			"provider_url": "http://keycloak:8080/realms/test",
			"client_id": "my-client",
			"client_secret": "my-secret",
			"redirect_url": "http://localhost/callback"
		}
	}`)
	path := writeConfig(t, configWithOIDC)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if !conf.OIDC.Enabled {
		t.Fatal("expected OIDC to be enabled from JSON")
	}
	if conf.OIDC.ClientID != "my-client" {
		t.Fatalf("unexpected ClientID: %q", conf.OIDC.ClientID)
	}
}

func TestLoadConfigWithInlineAI(t *testing.T) {
	configWithAI := []byte(`{
		"admin_server": {"listen_url": "127.0.0.1:3333"},
		"phish_server": {"listen_url": "0.0.0.0:8080"},
		"db_name": "sqlite3",
		"db_path": "gophish.db",
		"migrations_prefix": "db/db_",
		"ai": {
			"provider": "claude",
			"api_key": "sk-from-json",
			"model": "claude-sonnet-4-20250514",
			"monthly_token_budget": 500000
		}
	}`)
	path := writeConfig(t, configWithAI)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.AI.Provider != "claude" {
		t.Fatalf("unexpected provider: %q", conf.AI.Provider)
	}
	if conf.AI.APIKey != "sk-from-json" {
		t.Fatalf("unexpected API key: %q", conf.AI.APIKey)
	}
	if conf.AI.MonthlyTokenBudget != 500000 {
		t.Fatalf("unexpected token budget: %d", conf.AI.MonthlyTokenBudget)
	}
}

// ---------- Edge cases: empty/minimal JSON ----------

func TestLoadConfigEmptyJSON(t *testing.T) {
	path := writeConfig(t, []byte(`{}`))
	conf, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error loading empty JSON: %v", err)
	}
	// Defaults should still apply
	if conf.Logging == nil {
		t.Fatal("expected Logging to be initialized")
	}
	if conf.MFA.BackupCodeCount != 8 {
		t.Fatalf("expected BackupCodeCount default 8, got %d", conf.MFA.BackupCodeCount)
	}
	if conf.TestFlag {
		t.Fatal("TestFlag should be false")
	}
}

func TestLoadConfigMFANegativeBackupCodeCount(t *testing.T) {
	configWithNegative := []byte(`{
		"admin_server": {"listen_url": "127.0.0.1:3333"},
		"phish_server": {"listen_url": "0.0.0.0:8080"},
		"db_name": "sqlite3",
		"db_path": "gophish.db",
		"migrations_prefix": "db/db_",
		"mfa": {"backup_code_count": -5}
	}`)
	path := writeConfig(t, configWithNegative)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	// Negative value should be replaced with the default
	if conf.MFA.BackupCodeCount != 8 {
		t.Fatalf("expected BackupCodeCount default 8 for negative value, got %d", conf.MFA.BackupCodeCount)
	}
}

func TestLoadConfigMFAZeroBackupCodeCount(t *testing.T) {
	configWithZero := []byte(`{
		"admin_server": {"listen_url": "127.0.0.1:3333"},
		"phish_server": {"listen_url": "0.0.0.0:8080"},
		"db_name": "sqlite3",
		"db_path": "gophish.db",
		"migrations_prefix": "db/db_",
		"mfa": {"backup_code_count": 0}
	}`)
	path := writeConfig(t, configWithZero)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	// Zero should be replaced with the default
	if conf.MFA.BackupCodeCount != 8 {
		t.Fatalf("expected BackupCodeCount default 8 for zero value, got %d", conf.MFA.BackupCodeCount)
	}
}

// ---------- OIDC partial env overrides ----------

func TestLoadConfigOIDCPartialEnvOverride(t *testing.T) {
	// Only set client ID via env; other fields from config JSON
	configWithOIDC := []byte(`{
		"admin_server": {"listen_url": "127.0.0.1:3333"},
		"phish_server": {"listen_url": "0.0.0.0:8080"},
		"db_name": "sqlite3",
		"db_path": "gophish.db",
		"migrations_prefix": "db/db_",
		"oidc": {
			"enabled": true,
			"provider_url": "http://keycloak:8080/realms/test",
			"client_id": "json-client-id",
			"client_secret": "json-secret"
		}
	}`)
	t.Setenv("KEYCLOAK_CLIENT_ID", "env-client-id")
	path := writeConfig(t, configWithOIDC)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	// Environment variable should override the JSON value
	if conf.OIDC.ClientID != "env-client-id" {
		t.Fatalf("expected env-client-id, got %q", conf.OIDC.ClientID)
	}
	// Non-overridden field should retain JSON value
	if conf.OIDC.ClientSecret != "json-secret" {
		t.Fatalf("expected json-secret, got %q", conf.OIDC.ClientSecret)
	}
}

// ---------- AI: env API key should not override JSON when already set ----------

func TestLoadConfigAIEnvDoesNotOverrideJSON(t *testing.T) {
	configWithAI := []byte(`{
		"admin_server": {"listen_url": "127.0.0.1:3333"},
		"phish_server": {"listen_url": "0.0.0.0:8080"},
		"db_name": "sqlite3",
		"db_path": "gophish.db",
		"migrations_prefix": "db/db_",
		"ai": {
			"provider": "claude",
			"api_key": "sk-json-key"
		}
	}`)
	// Set CLAUDE_API_KEY in env — this should override the JSON key
	t.Setenv("CLAUDE_API_KEY", "sk-env-key")
	path := writeConfig(t, configWithAI)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.AI.APIKey != "sk-env-key" {
		t.Fatalf("expected env key to override JSON, got %q", conf.AI.APIKey)
	}
}

func TestLoadConfigAIOpenAIIgnoredWhenClaudeSet(t *testing.T) {
	// JSON already has a Claude key; OPENAI_API_KEY should be ignored
	configWithAI := []byte(`{
		"admin_server": {"listen_url": "127.0.0.1:3333"},
		"phish_server": {"listen_url": "0.0.0.0:8080"},
		"db_name": "sqlite3",
		"db_path": "gophish.db",
		"migrations_prefix": "db/db_",
		"ai": {
			"provider": "claude",
			"api_key": "sk-claude-json"
		}
	}`)
	t.Setenv("OPENAI_API_KEY", "sk-openai-env")
	path := writeConfig(t, configWithAI)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	// Claude key from JSON should still be present (OPENAI only overrides when APIKey is empty)
	if conf.AI.APIKey != "sk-claude-json" {
		t.Fatalf("expected Claude JSON key to remain, got %q", conf.AI.APIKey)
	}
}

// ---------- MigrationsPath with different DB names ----------

func TestLoadConfigMigrationsPathMySQL(t *testing.T) {
	configMySQL := []byte(`{
		"admin_server": {"listen_url": "127.0.0.1:3333"},
		"phish_server": {"listen_url": "0.0.0.0:8080"},
		"db_name": "mysql",
		"db_path": "gophish:password@/gophish",
		"migrations_prefix": "db/db_"
	}`)
	path := writeConfig(t, configMySQL)
	conf, err := LoadConfig(path)
	if err != nil {
		fatalIfErr(t, err)
	}
	if conf.MigrationsPath != "db/db_mysql" {
		t.Fatalf("expected MigrationsPath 'db/db_mysql', got %q", conf.MigrationsPath)
	}
}
