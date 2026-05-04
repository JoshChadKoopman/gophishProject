package config

import (
	"encoding/json"
	"os"

	log "github.com/gophish/gophish/logger"
)

// OIDCConfig holds Keycloak (or any OIDC-compatible) provider settings.
// All fields can be overridden via environment variables at startup.
type OIDCConfig struct {
	Enabled      bool   `json:"enabled"`
	ProviderURL  string `json:"provider_url"` // e.g. http://keycloak:8080/realms/nivoxis
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"` // e.g. https://app.example.com/auth/oidc/callback
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
	Enabled                bool   `json:"enabled"`                  // Whether AI features are enabled
	Provider               string `json:"provider"`                 // "claude" or "openai"
	APIKey                 string `json:"api_key"`                  // API key for the provider
	Model                  string `json:"model"`                    // e.g. "claude-sonnet-4-20250514", "gpt-4o"
	MonthlyTokenBudget     int    `json:"monthly_token_budget"`     // 0 = unlimited
	TelephonyWebhookSecret string `json:"telephony_webhook_secret"` // Shared secret for vishing telephony callbacks
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

// SAMLConfig holds settings for SAML 2.0 SSO with separate admin/user paths.
type SAMLConfig struct {
	Enabled         bool   `json:"enabled"`
	IDPURL          string `json:"idp_url"`           // IdP SSO endpoint URL
	IDPMetadataURL  string `json:"idp_metadata_url"`  // IdP Metadata URL
	SPEntityID      string `json:"sp_entity_id"`      // e.g. https://app.example.com/saml
	AdminGroupClaim string `json:"admin_group_claim"` // SAML attribute for admin group
	AdminGroupValue string `json:"admin_group_value"` // Value that grants admin access
	DefaultRoleSlug string `json:"default_role_slug"` // Default role for SSO-provisioned users
	SplitAdminUser  bool   `json:"split_admin_user"`  // Enable separate admin/user SSO paths
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
	SAML           SAMLConfig  `json:"saml"`
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
	configFile, err := os.ReadFile(filepath)
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

	applyOIDCEnvOverrides(config)
	applySAMLEnvOverrides(config)
	applyMFAEnvOverrides(config)
	applyAIEnvOverrides(config)

	return config, nil
}

// applyOIDCEnvOverrides applies OIDC/Keycloak environment variable overrides.
func applyOIDCEnvOverrides(config *Config) {
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
}

// applyMFAEnvOverrides applies MFA environment variable overrides.
func applyMFAEnvOverrides(config *Config) {
	if encKey := os.Getenv("MFA_TOTP_ENCRYPTION_KEY"); encKey != "" {
		config.MFA.TOTPEncryptionKey = encKey
	}
	if config.MFA.BackupCodeCount <= 0 {
		config.MFA.BackupCodeCount = 8
	}
}

// applySAMLEnvOverrides applies SAML environment variable overrides.
func applySAMLEnvOverrides(config *Config) {
	if idpURL := os.Getenv("SAML_IDP_URL"); idpURL != "" {
		config.SAML.Enabled = true
		config.SAML.IDPURL = idpURL
	}
	if entityID := os.Getenv("SAML_SP_ENTITY_ID"); entityID != "" {
		config.SAML.SPEntityID = entityID
	}
	if adminGroup := os.Getenv("SAML_ADMIN_GROUP"); adminGroup != "" {
		config.SAML.AdminGroupValue = adminGroup
	}
	if os.Getenv("SAML_SPLIT_ADMIN_USER") == "true" {
		config.SAML.SplitAdminUser = true
	}
}

// applyAIEnvOverrides applies AI provider environment variable overrides.
func applyAIEnvOverrides(config *Config) {
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
}
