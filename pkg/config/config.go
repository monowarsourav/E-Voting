package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration.
type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	Crypto      CryptoConfig
	Election    ElectionConfig
	Blockchain  BlockchainConfig
	SA2         SA2Config
	Auth        AuthConfig
	Logging     LoggingConfig
	TLS         TLSConfig
	CORS        CORSConfig
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
	Path string
}

// CryptoConfig holds cryptographic parameters.
type CryptoConfig struct {
	PaillierKeySize int
	PedersenKeySize int
	RingKeySize     int
	SMDCSlots       int
	// DuressHMACKey is the server-side secret for HMAC-SHA256 of behavioral
	// duress signals. Loaded from DURESS_HMAC_KEY env var. When empty a
	// hard-coded dev fallback is used and a warning should be logged.
	DuressHMACKey string
}

// ElectionConfig holds election parameters.
type ElectionConfig struct {
	MaxVotesPerSecond int
	VotingPeriodHours int
}

// BlockchainConfig holds blockchain configuration.
type BlockchainConfig struct {
	Network       string
	ChannelName   string
	ChaincodeName string
	Enabled       bool
}

// SA2Config holds SA² server configuration.
type SA2Config struct {
	ServerAURL     string
	ServerBURL     string
	Threshold      int
	LeaderAPIKey   string
	LeaderAdminKey string
	HelperAPIKey   string
	HelperAdminKey string
}

// AuthConfig holds authentication-related configuration. Values here MUST be
// supplied via environment variables in production; defaults are empty to
// force explicit configuration.
type AuthConfig struct {
	AdminTokens []string // loaded from ADMIN_TOKEN (comma-separated)
	JWTSecret   string
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// TLSConfig holds TLS configuration. When Enabled=true both CertFile and
// KeyFile must be present and readable.
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

// CORSConfig holds CORS allowlist. An empty AllowedOrigins disables CORS (no
// wildcard in production).
type CORSConfig struct {
	AllowedOrigins []string
}

// DefaultConfig returns the default configuration. Values are safe for local
// development only; production callers MUST populate Auth.AdminTokens and
// JWTSecret via environment variables.
func DefaultConfig() *Config {
	return &Config{
		Environment: "development",
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Database: DatabaseConfig{
			Path: "./data/covertvote.db",
		},
		Crypto: CryptoConfig{
			PaillierKeySize: 2048,
			PedersenKeySize: 1024,
			RingKeySize:     1024,
			SMDCSlots:       5,
		},
		Election: ElectionConfig{
			MaxVotesPerSecond: 1000,
			VotingPeriodHours: 24,
		},
		Blockchain: BlockchainConfig{
			Network:       "test",
			ChannelName:   "covertvote-channel",
			ChaincodeName: "covertvote-chaincode",
			Enabled:       false,
		},
		SA2: SA2Config{
			ServerAURL: "http://localhost:8081",
			ServerBURL: "http://localhost:8082",
			Threshold:  2,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// LoadConfig loads configuration from environment variables on top of defaults.
// Returns an error on invalid values (e.g. unparseable integers).
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Environment = env
	}

	// Server
	if host := getenvFirst("SERVER_HOST", "COVERTVOTE_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if port, ok, err := getenvInt("SERVER_PORT", "COVERTVOTE_PORT"); err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
	} else if ok {
		cfg.Server.Port = port
	}

	// Database
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}

	// Crypto
	if v, ok, err := getenvInt("PAILLIER_KEY_SIZE", "COVERTVOTE_KEY_SIZE"); err != nil {
		return nil, fmt.Errorf("invalid PAILLIER_KEY_SIZE: %w", err)
	} else if ok {
		cfg.Crypto.PaillierKeySize = v
	}
	if v, ok, err := getenvInt("PEDERSEN_KEY_SIZE"); err != nil {
		return nil, fmt.Errorf("invalid PEDERSEN_KEY_SIZE: %w", err)
	} else if ok {
		cfg.Crypto.PedersenKeySize = v
	}
	if v, ok, err := getenvInt("RING_KEY_SIZE"); err != nil {
		return nil, fmt.Errorf("invalid RING_KEY_SIZE: %w", err)
	} else if ok {
		cfg.Crypto.RingKeySize = v
	}
	if v, ok, err := getenvInt("SMDC_SLOTS"); err != nil {
		return nil, fmt.Errorf("invalid SMDC_SLOTS: %w", err)
	} else if ok {
		cfg.Crypto.SMDCSlots = v
	}
	cfg.Crypto.DuressHMACKey = os.Getenv("DURESS_HMAC_KEY")

	// SA² URLs
	if v := os.Getenv("SA2_SERVER_A_URL"); v != "" {
		cfg.SA2.ServerAURL = v
	}
	if v := os.Getenv("SA2_SERVER_B_URL"); v != "" {
		cfg.SA2.ServerBURL = v
	}
	cfg.SA2.LeaderAPIKey = os.Getenv("SA2_LEADER_API_KEY")
	cfg.SA2.LeaderAdminKey = os.Getenv("SA2_LEADER_ADMIN_KEY")
	cfg.SA2.HelperAPIKey = os.Getenv("SA2_HELPER_API_KEY")
	cfg.SA2.HelperAdminKey = os.Getenv("SA2_HELPER_ADMIN_KEY")

	// Blockchain
	if v := os.Getenv("FABRIC_NETWORK"); v != "" {
		cfg.Blockchain.Network = v
	}
	if v := os.Getenv("FABRIC_CHANNEL"); v != "" {
		cfg.Blockchain.ChannelName = v
	}
	if v := os.Getenv("FABRIC_CHAINCODE"); v != "" {
		cfg.Blockchain.ChaincodeName = v
	}
	if v := os.Getenv("FABRIC_ENABLED"); v != "" {
		cfg.Blockchain.Enabled = isTruthy(v)
	}

	// Auth — NO defaults. Admin access disabled unless explicitly configured.
	if v := os.Getenv("ADMIN_TOKEN"); v != "" {
		cfg.Auth.AdminTokens = splitAndTrim(v, ",")
	}
	cfg.Auth.JWTSecret = os.Getenv("JWT_SECRET")

	// Logging
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Logging.Level = strings.ToLower(v)
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.Logging.Format = strings.ToLower(v)
	}

	// TLS
	if v := os.Getenv("TLS_ENABLED"); v != "" {
		cfg.TLS.Enabled = isTruthy(v)
	}
	cfg.TLS.CertFile = os.Getenv("TLS_CERT_FILE")
	cfg.TLS.KeyFile = os.Getenv("TLS_KEY_FILE")

	// CORS
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		cfg.CORS.AllowedOrigins = splitAndTrim(v, ",")
	}

	return cfg, nil
}

// Validate validates the configuration. Production mode enforces stricter
// requirements than development.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.New("invalid server port")
	}
	if c.Crypto.PaillierKeySize < 2048 {
		return errors.New("paillier key size must be at least 2048 bits")
	}
	if c.Crypto.SMDCSlots < 2 {
		return errors.New("smdc slots must be at least 2")
	}
	if c.SA2.Threshold < 2 {
		return errors.New("sa² threshold must be at least 2")
	}

	if c.IsProduction() {
		// Predictable HMAC key would let an attacker pre-compute valid duress
		// hashes and bypass coercion-resistance for all voters.
		if c.Crypto.DuressHMACKey == "" {
			return errors.New("DURESS_HMAC_KEY is required when ENVIRONMENT=production")
		}
		if len(c.Auth.AdminTokens) == 0 {
			return errors.New("admin_token must be set in production")
		}
		for _, tok := range c.Auth.AdminTokens {
			if len(tok) < 32 {
				return errors.New("admin_token entries must be at least 32 characters in production")
			}
		}
		if c.Auth.JWTSecret == "" || len(c.Auth.JWTSecret) < 32 {
			return errors.New("jwt_secret must be set (>= 32 chars) in production")
		}
		if c.TLS.Enabled && (c.TLS.CertFile == "" || c.TLS.KeyFile == "") {
			return errors.New("tls_cert_file and tls_key_file required when tls_enabled=true")
		}
		if len(c.CORS.AllowedOrigins) == 0 {
			return errors.New("cors_allowed_origins must be set in production")
		}
	}

	return nil
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	env := c.Environment
	return env == "" || env == "development" || env == "dev"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production" || c.Environment == "prod"
}

// --- env helpers ---

func getenvFirst(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// getenvInt looks up the first non-empty key among those provided and parses
// it as an int. Returns (value, found, error).
func getenvInt(keys ...string) (int, bool, error) {
	raw := getenvFirst(keys...)
	if raw == "" {
		return 0, false, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false, err
	}
	return v, true, nil
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "y", "on":
		return true
	}
	return false
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
