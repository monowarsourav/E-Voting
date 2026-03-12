package config

import (
	"errors"
	"os"
	"time"
)

// Config holds the application configuration
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Crypto      CryptoConfig
	Election    ElectionConfig
	Blockchain  BlockchainConfig
	SA2         SA2Config
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path string
}

// CryptoConfig holds cryptographic parameters
type CryptoConfig struct {
	PaillierKeySize int  // Key size in bits (e.g., 2048)
	PedersenKeySize int  // Key size for Pedersen (e.g., 1024)
	RingKeySize     int  // Key size for ring signatures
	SMDCSlots       int  // Number of SMDC slots (k)
}

// ElectionConfig holds election parameters
type ElectionConfig struct {
	MaxVotesPerSecond int
	VotingPeriodHours int
}

// BlockchainConfig holds blockchain configuration
type BlockchainConfig struct {
	Network     string
	ChannelName string
	ChaincodeName string
	Enabled     bool
}

// SA2Config holds SA² server configuration
type SA2Config struct {
	ServerAURL string
	ServerBURL string
	Threshold  int // Number of servers required for decryption
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
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
			Enabled:       false, // Disabled by default until Hyperledger is integrated
		},
		SA2: SA2Config{
			ServerAURL: "http://localhost:8081",
			ServerBURL: "http://localhost:8082",
			Threshold:  2,
		},
	}
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	// Override with environment variables if present
	if host := os.Getenv("COVERTVOTE_HOST"); host != "" {
		cfg.Server.Host = host
	}

	if port := os.Getenv("COVERTVOTE_PORT"); port != "" {
		// Parse port (simplified)
		cfg.Server.Port = 8080 // Use default for now
	}

	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}

	if keySizeStr := os.Getenv("COVERTVOTE_KEY_SIZE"); keySizeStr != "" {
		// Parse key size (simplified)
		cfg.Crypto.PaillierKeySize = 2048
	}

	if sa2ServerA := os.Getenv("SA2_SERVER_A_URL"); sa2ServerA != "" {
		cfg.SA2.ServerAURL = sa2ServerA
	}

	if sa2ServerB := os.Getenv("SA2_SERVER_B_URL"); sa2ServerB != "" {
		cfg.SA2.ServerBURL = sa2ServerB
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.New("invalid server port")
	}

	if c.Crypto.PaillierKeySize < 2048 {
		return errors.New("Paillier key size must be at least 2048 bits")
	}

	if c.Crypto.SMDCSlots < 2 {
		return errors.New("SMDC slots must be at least 2")
	}

	if c.SA2.Threshold < 2 {
		return errors.New("SA² threshold must be at least 2")
	}

	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	env := os.Getenv("ENVIRONMENT")
	return env == "" || env == "development" || env == "dev"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	env := os.Getenv("ENVIRONMENT")
	return env == "production" || env == "prod"
}
