package config

import (
	"encoding/json"
	"io/ioutil"
	"os"

	log "github.com/gophish/gophish/logger"
)

// OIDCConfig holds Keycloak (or any OIDC-compatible) provider settings.
// All fields can be overridden via environment variables at startup.
type OIDCConfig struct {
	Enabled      bool   `json:"enabled"`
	ProviderURL  string `json:"provider_url"`   // e.g. http://keycloak:8080/realms/nivoxis
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`   // e.g. https://app.example.com/auth/oidc/callback
}

// MFAConfig holds settings for the native TOTP / backup-code subsystem.
// TOTPEncryptionKey must be a 32-byte value encoded as base64. If absent,
// MFA enrollment will be refused to prevent storing unencrypted secrets.
type MFAConfig struct {
	TOTPEncryptionKey string `json:"totp_encryption_key"`
	BackupCodeCount   int    `json:"backup_code_count"` // defaults to 8
}

// AIConfig holds settings for AI-powered template generation.
// Provider must be "claude" or "openai". API keys can be set via env vars.
type AIConfig struct {
	Provider          string `json:"provider"`            // "claude" or "openai"
	APIKey            string `json:"api_key"`             // API key for the provider
	Model             string `json:"model"`               // e.g. "claude-sonnet-4-20250514", "gpt-4o"
	MonthlyTokenBudget int   `json:"monthly_token_budget"` // 0 = unlimited
}

// AdminServer represents the Admin server configuration details
type AdminServer struct {
	ListenURL            string   `json:"listen_url"`
	UseTLS               bool     `json:"use_tls"`
	CertPath             string   `json:"cert_path"`
	KeyPath              string   `json:"key_path"`
	CSRFKey              string   `json:"csrf_key"`
	AllowedInternalHosts []string `json:"allowed_internal_hosts"`
	TrustedOrigins       []string `json:"trusted_origins"`
}

// PhishServer represents the Phish server configuration details
type PhishServer struct {
	ListenURL string `json:"listen_url"`
	UseTLS    bool   `json:"use_tls"`
	CertPath  string `json:"cert_path"`
	KeyPath   string `json:"key_path"`
}

// Config represents the configuration information.
type Config struct {
	AdminConf      AdminServer `json:"admin_server"`
	PhishConf      PhishServer `json:"phish_server"`
	DBName         string      `json:"db_name"`
	DBPath         string      `json:"db_path"`
	DBSSLCaPath    string      `json:"db_sslca_path"`
	MigrationsPath string      `json:"migrations_prefix"`
	TestFlag       bool        `json:"test_flag"`
	ContactAddress string      `json:"contact_address"`
	Logging        *log.Config `json:"logging"`
	OIDC           OIDCConfig  `json:"oidc"`
	MFA            MFAConfig   `json:"mfa"`
	AI             AIConfig    `json:"ai"`
}

// Version contains the current gophish version
var Version = ""

// ServerName is the server type that is returned in the transparency response.
const ServerName = "gophish"

// LoadConfig loads the configuration from the specified filepath
func LoadConfig(filepath string) (*Config, error) {
	// Get the config file
	configFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = json.Unmarshal(configFile, config)
	if err != nil {
		return nil, err
	}
	if config.Logging == nil {
		config.Logging = &log.Config{}
	}
	// Choosing the migrations directory based on the database used.
	config.MigrationsPath = config.MigrationsPath + config.DBName
	// Explicitly set the TestFlag to false to prevent config.json overrides
	config.TestFlag = false

	// Apply OIDC environment variable overrides. Setting KEYCLOAK_URL enables
	// OIDC regardless of what the config.json says.
	if keycloakURL := os.Getenv("KEYCLOAK_URL"); keycloakURL != "" {
		realm := os.Getenv("KEYCLOAK_REALM")
		if realm == "" {
			realm = "master"
		}
		config.OIDC.Enabled = true
		config.OIDC.ProviderURL = keycloakURL + "/realms/" + realm
	}
	if clientID := os.Getenv("KEYCLOAK_CLIENT_ID"); clientID != "" {
		config.OIDC.ClientID = clientID
	}
	if clientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET"); clientSecret != "" {
		config.OIDC.ClientSecret = clientSecret
	}
	if redirectURL := os.Getenv("OIDC_REDIRECT_URL"); redirectURL != "" {
		config.OIDC.RedirectURL = redirectURL
	}

	// Apply MFA environment variable overrides.
	if encKey := os.Getenv("MFA_TOTP_ENCRYPTION_KEY"); encKey != "" {
		config.MFA.TOTPEncryptionKey = encKey
	}
	if config.MFA.BackupCodeCount <= 0 {
		config.MFA.BackupCodeCount = 8
	}

	// Apply AI environment variable overrides.
	if provider := os.Getenv("NIVOXIS_AI_PROVIDER"); provider != "" {
		config.AI.Provider = provider
	}
	if apiKey := os.Getenv("CLAUDE_API_KEY"); apiKey != "" {
		config.AI.APIKey = apiKey
		if config.AI.Provider == "" {
			config.AI.Provider = "claude"
		}
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" && config.AI.APIKey == "" {
		config.AI.APIKey = apiKey
		if config.AI.Provider == "" {
			config.AI.Provider = "openai"
		}
	}
	if model := os.Getenv("NIVOXIS_AI_MODEL"); model != "" {
		config.AI.Model = model
	}
	// Default models per provider
	if config.AI.Model == "" && config.AI.Provider == "claude" {
		config.AI.Model = "claude-sonnet-4-20250514"
	}
	if config.AI.Model == "" && config.AI.Provider == "openai" {
		config.AI.Model = "gpt-4o"
	}

	return config, nil
}
