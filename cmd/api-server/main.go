// Package main launches the CovertVote API server. The program is kept thin:
// it reads configuration, wires dependencies, and drives the HTTP server
// lifecycle. Business logic lives under internal/, reusable libraries under
// pkg/, and HTTP adapters under api/.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/covertvote/e-voting/api/handlers"
	"github.com/covertvote/e-voting/api/middleware"
	"github.com/covertvote/e-voting/api/models"
	"github.com/covertvote/e-voting/api/routes"
	_ "github.com/covertvote/e-voting/docs"
	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/database"
	"github.com/covertvote/e-voting/internal/repository/keyimage"
	"github.com/covertvote/e-voting/internal/repository/session"
	"github.com/covertvote/e-voting/internal/tally"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/internal/voting"
	"github.com/covertvote/e-voting/pkg/audit"
	"github.com/covertvote/e-voting/pkg/config"
	"github.com/covertvote/e-voting/pkg/logger"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// version is stamped at build time via -ldflags "-X main.version=...".
var version = "1.0.0"

// @title           CovertVote E-Voting API
// @version         1.0.0
// @description     Blockchain-based secure e-voting system with homomorphic encryption, SMDC coercion resistance, and anonymous voting
// @BasePath        /api/v1
// @schemes         http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @securityDefinitions.apikey AdminAuth
// @in header
// @name Authorization
func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	log := logger.Init(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})
	log.Info("starting CovertVote API",
		"version", version,
		"environment", cfg.Environment)

	// --- Admin tokens ---
	middleware.SetAdminTokens(cfg.Auth.AdminTokens)
	if len(cfg.Auth.AdminTokens) == 0 {
		log.Warn("ADMIN_TOKEN not configured — admin endpoints will reject all requests")
	}

	// --- Database ---
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.RunMigrations("./migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	if err := audit.InitAuditTable(db.DB); err != nil {
		return fmt.Errorf("init audit table: %w", err)
	}
	log.Info("database initialised", "path", cfg.Database.Path)

	// --- Repositories ---
	sessionStore := session.New(db.DB)
	middleware.VoterSessions.Backend = sessionStore
	keyImageStore := keyimage.New(db.DB)
	auditLogger := audit.NewAuditLogger(db.DB)

	// --- Gin ---
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(log))
	router.Use(middleware.CORSMiddleware(cfg.CORS.AllowedOrigins))

	if err := models.RegisterValidators(); err != nil {
		return fmt.Errorf("register validators: %w", err)
	}

	// --- Rate limiters (tied to app lifecycle) ---
	rlCtx, rlCancel := context.WithCancel(context.Background())
	defer rlCancel()
	standardLimiter := middleware.NewRateLimiter(rlCtx, 100, 200)
	strictLimiter := middleware.NewRateLimiter(rlCtx, 10, 20)

	// --- Crypto parameters ---
	log.Info("generating crypto parameters",
		"paillier_bits", cfg.Crypto.PaillierKeySize,
		"pedersen_bits", cfg.Crypto.PedersenKeySize,
		"ring_bits", cfg.Crypto.RingKeySize)

	paillierSK, err := crypto.GeneratePaillierKeyPair(cfg.Crypto.PaillierKeySize)
	if err != nil {
		return fmt.Errorf("generate paillier keys: %w", err)
	}
	pedersenParams, err := crypto.GeneratePedersenParams(cfg.Crypto.PedersenKeySize)
	if err != nil {
		return fmt.Errorf("generate pedersen params: %w", err)
	}
	ringParams, err := crypto.GenerateRingParams(cfg.Crypto.RingKeySize)
	if err != nil {
		return fmt.Errorf("generate ring params: %w", err)
	}

	// --- Domain objects ---
	fingerprintProcessor := biometric.NewFingerprintProcessor()
	livenessDetector := biometric.NewLivenessDetector(0.5)

	eligibleVoters := make([]string, 0, 105)
	for i := 1; i <= 100; i++ {
		eligibleVoters = append(eligibleVoters, fmt.Sprintf("voter%03d", i))
	}
	eligibleVoters = append(eligibleVoters, "test_voter", "admin", "alice", "bob", "charlie")

	registrationSystem := voter.NewRegistrationSystem(
		pedersenParams, ringParams, cfg.Crypto.SMDCSlots, eligibleVoters, "election001",
	)

	election := &voting.Election{
		ElectionID:  "election001",
		Title:       "Presidential Election 2026",
		Description: "Annual presidential election",
		Candidates: []*voting.Candidate{
			{ID: 1, Name: "Candidate A", Description: "Experienced leader", Party: "Party 1"},
			{ID: 2, Name: "Candidate B", Description: "Reform candidate", Party: "Party 2"},
		},
		StartTime: time.Now().Unix(),
		EndTime:   time.Now().Add(time.Duration(cfg.Election.VotingPeriodHours) * time.Hour).Unix(),
		IsActive:  true,
	}

	voteCaster := voting.NewVoteCaster(
		paillierSK.PublicKey,
		ringParams,
		registrationSystem,
		election,
		voting.WithKeyImageStore(keyImageStore),
	)

	counter := tally.NewCounter(paillierSK.PublicKey, paillierSK)

	// --- Handlers ---
	registrationHandler := handlers.NewRegistrationHandler(
		registrationSystem, fingerprintProcessor, livenessDetector,
	)
	registrationHandler.Auditor = auditLogger
	votingHandler := handlers.NewVotingHandler(
		voteCaster, registrationSystem, fingerprintProcessor, livenessDetector,
	)
	votingHandler.Auditor = auditLogger
	votingHandler.Elections[election.ElectionID] = election
	tallyHandler := handlers.NewTallyHandler(counter, voteCaster, votingHandler.Elections)
	healthHandler := handlers.NewHealthHandler(version, db.DB)

	routes.SetupRoutes(router, routes.Dependencies{
		Registration:    registrationHandler,
		Voting:          votingHandler,
		Tally:           tallyHandler,
		Health:          healthHandler,
		StandardLimiter: standardLimiter,
		StrictLimiter:   strictLimiter,
	})
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// --- Background workers ---
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go sessionJanitor(workerCtx, log)

	// --- HTTP server ---
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
	if cfg.TLS.Enabled {
		srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("HTTP server listening",
			"addr", addr,
			"tls", cfg.TLS.Enabled)
		var err error
		if cfg.TLS.Enabled {
			err = srv.ListenAndServeTLS(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info("shutdown signal received", "signal", sig.String())
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	workerCancel()
	standardLimiter.Stop()
	strictLimiter.Stop()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	log.Info("server exited gracefully")
	return nil
}

// sessionJanitor periodically evicts expired sessions.
func sessionJanitor(ctx context.Context, log *slog.Logger) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			defer func() {
				if r := recover(); r != nil {
					log.Error("session janitor panic", "recover", r)
				}
			}()
			middleware.CleanupExpiredSessions()
		}
	}
}
