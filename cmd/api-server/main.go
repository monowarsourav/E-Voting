package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/covertvote/e-voting/api/handlers"
	"github.com/covertvote/e-voting/api/middleware"
	"github.com/covertvote/e-voting/api/routes"
	_ "github.com/covertvote/e-voting/docs" // Swagger docs
	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/database"
	"github.com/covertvote/e-voting/internal/tally"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/internal/voting"
	"github.com/covertvote/e-voting/pkg/config"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const VERSION = "1.0.0"

// @title           CovertVote E-Voting API
// @version         1.0.0
// @description     Blockchain-based secure e-voting system with homomorphic encryption, SMDC coercion resistance, and anonymous voting
// @description     Features: Paillier encryption, Pedersen commitments, Ring signatures, Zero-knowledge proofs, SA² aggregation
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.covertvote.io/support
// @contact.email  support@covertvote.io

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1
// @schemes   http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @securityDefinitions.apikey AdminAuth
// @in header
// @name Authorization
// @description Admin bearer token for administrative operations

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	log.Println("Initializing database...")
	dbPath := cfg.Database.Path
	if dbPath == "" {
		dbPath = "./data/covertvote.db"
	}

	// Ensure database directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// Connect to database
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	log.Println("Running database migrations...")
	migrationsDir := "./migrations"
	if err := db.RunMigrations(migrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database initialized successfully")

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create router
	router := gin.Default()

	// Add CORS middleware
	router.Use(middleware.CORSMiddleware())

	// Initialize cryptographic components
	log.Println("Initializing cryptographic components...")

	// Generate Paillier keys
	paillierSK, err := crypto.GeneratePaillierKeyPair(cfg.Crypto.PaillierKeySize)
	if err != nil {
		log.Fatalf("Failed to generate Paillier keys: %v", err)
	}
	paillierPK := paillierSK.PublicKey

	// Generate Pedersen parameters
	pedersenParams, err := crypto.GeneratePedersenParams(cfg.Crypto.PedersenKeySize)
	if err != nil {
		log.Fatalf("Failed to generate Pedersen parameters: %v", err)
	}

	// Generate Ring parameters
	ringParams, err := crypto.GenerateRingParams(cfg.Crypto.RingKeySize)
	if err != nil {
		log.Fatalf("Failed to generate Ring parameters: %v", err)
	}

	// Initialize biometric components
	log.Println("Initializing biometric components...")
	fingerprintProcessor := biometric.NewFingerprintProcessor()
	livenessDetector := biometric.NewLivenessDetector(0.5) // 0.5 entropy threshold

	// Initialize voter registration system
	log.Println("Initializing voter registration system...")
	// Development: Sample eligible voters (voter001-voter100)
	eligibleVoters := []string{}
	for i := 1; i <= 100; i++ {
		eligibleVoters = append(eligibleVoters, fmt.Sprintf("voter%03d", i))
	}
	// Add some common test voter IDs
	eligibleVoters = append(eligibleVoters, "test_voter", "admin", "alice", "bob", "charlie")
	log.Printf("Loaded %d eligible voters for development", len(eligibleVoters))

	registrationSystem := voter.NewRegistrationSystem(
		pedersenParams,
		ringParams,
		cfg.Crypto.SMDCSlots,
		eligibleVoters,
		"election001",
	)

	// Create sample election
	log.Println("Creating sample election...")
	election := &voting.Election{
		ElectionID:  "election001",
		Title:       "Presidential Election 2026",
		Description: "Annual presidential election",
		Candidates: []*voting.Candidate{
			{ID: 1, Name: "Candidate A", Description: "Experienced leader", Party: "Party 1"},
			{ID: 2, Name: "Candidate B", Description: "Reform candidate", Party: "Party 2"},
		},
		StartTime: time.Now().Unix(),
		EndTime:   time.Now().Add(24 * time.Hour).Unix(),
		IsActive:  true,
	}

	// Initialize vote caster
	log.Println("Initializing vote casting system...")
	voteCaster := voting.NewVoteCaster(paillierPK, ringParams, registrationSystem, election)

	// Initialize tally counter
	log.Println("Initializing tallying system...")
	counter := tally.NewCounter(paillierPK, paillierSK)

	// Initialize handlers
	log.Println("Initializing API handlers...")
	registrationHandler := handlers.NewRegistrationHandler(
		registrationSystem,
		fingerprintProcessor,
		livenessDetector,
	)
	votingHandler := handlers.NewVotingHandler(
		voteCaster,
		registrationSystem,
		fingerprintProcessor,
		livenessDetector,
	)
	votingHandler.Elections[election.ElectionID] = election // Add sample election
	tallyHandler := handlers.NewTallyHandler(counter, voteCaster, votingHandler.Elections)
	healthHandler := handlers.NewHealthHandler(VERSION)

	// Setup routes
	log.Println("Setting up routes...")
	routes.SetupRoutes(router, registrationHandler, votingHandler, tallyHandler, healthHandler)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start session cleanup routine with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in session cleanup goroutine: %v", r)
			}
		}()
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			middleware.CleanupExpiredSessions()
		}
	}()

	// Start server with graceful shutdown
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting CovertVote API Server v%s on %s", VERSION, addr)
	log.Printf("Swagger UI: http://%s/swagger/index.html", addr)
	log.Printf("API Documentation: http://%s/api/v1", addr)
	log.Printf("Health Check: http://%s/health", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
