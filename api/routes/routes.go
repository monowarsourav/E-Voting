package routes

import (
	"github.com/covertvote/e-voting/api/handlers"
	"github.com/covertvote/e-voting/api/middleware"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(
	router *gin.Engine,
	registrationHandler *handlers.RegistrationHandler,
	votingHandler *handlers.VotingHandler,
	tallyHandler *handlers.TallyHandler,
	healthHandler *handlers.HealthHandler,
) {
	// Health checks (no auth required)
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/ready", healthHandler.ReadinessCheck)
	router.GET("/live", healthHandler.LivenessCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes (with rate limiting)
		public := v1.Group("")
		public.Use(middleware.StandardRateLimitMiddleware())
		{
			// Registration and Login
			public.POST("/register", registrationHandler.Register)
			public.POST("/login", registrationHandler.Login)
			public.POST("/verify-eligibility", registrationHandler.VerifyEligibility)

			// Elections (read-only)
			public.GET("/elections", votingHandler.GetElections)
			public.GET("/elections/:id", votingHandler.GetElection)

			// Vote count summary (read-only, no sensitive data)
			public.GET("/vote-count", tallyHandler.GetVoteCount)
		}

		// Authenticated routes (voter auth required)
		authenticated := v1.Group("")
		authenticated.Use(middleware.AuthMiddleware())
		authenticated.Use(middleware.StrictRateLimitMiddleware())
		{
			// Voter info
			authenticated.GET("/voter/:id", registrationHandler.GetVoterInfo)

			// Vote casting
			authenticated.POST("/vote", votingHandler.CastVote)
			authenticated.POST("/verify-vote", votingHandler.VerifyVote)

			// Results (requires authentication)
			authenticated.GET("/results/:electionId", tallyHandler.GetResults)
		}

		// Admin routes (admin auth required)
		admin := v1.Group("/admin")
		admin.Use(middleware.AdminAuthMiddleware())
		admin.Use(middleware.StrictRateLimitMiddleware())
		{
			// Election management
			admin.POST("/elections", votingHandler.CreateElection)

			// Tallying
			admin.POST("/tally", tallyHandler.TallyVotes)

			// System info
			admin.GET("/voters", registrationHandler.GetAllVoters)
		}
	}
}
