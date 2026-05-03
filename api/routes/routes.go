package routes

import (
	"github.com/covertvote/e-voting/api/handlers"
	"github.com/covertvote/e-voting/api/middleware"
	"github.com/gin-gonic/gin"
)

// Dependencies bundles the objects required to configure the API router.
// Injecting limiters (rather than constructing new ones per middleware call)
// ensures their janitor goroutines can be shut down cleanly on exit.
type Dependencies struct {
	Registration *handlers.RegistrationHandler
	Voting       *handlers.VotingHandler
	Tally        *handlers.TallyHandler
	Health       *handlers.HealthHandler
	Duress       *handlers.DuressHandler // optional; nil disables the duress-signal endpoint

	// Rate limiters — standard for public reads, strict for sensitive ops.
	StandardLimiter *middleware.RateLimiter
	StrictLimiter   *middleware.RateLimiter
}

// SetupRoutes configures all API routes using the provided dependencies.
func SetupRoutes(router *gin.Engine, deps Dependencies) {
	// Health checks (no auth required, not rate-limited so probes always work).
	router.GET("/health", deps.Health.HealthCheck)
	router.GET("/ready", deps.Health.ReadinessCheck)
	router.GET("/live", deps.Health.LivenessCheck)

	v1 := router.Group("/api/v1")
	{
		public := v1.Group("")
		public.Use(middleware.RateLimitMiddleware(deps.StandardLimiter))
		{
			public.POST("/register", deps.Registration.Register)
			public.POST("/login", deps.Registration.Login)
			public.POST("/verify-eligibility", deps.Registration.VerifyEligibility)

			public.GET("/elections", deps.Voting.GetElections)
			public.GET("/elections/:id", deps.Voting.GetElection)

			public.GET("/vote-count", deps.Tally.GetVoteCount)
		}

		authenticated := v1.Group("")
		authenticated.Use(middleware.AuthMiddleware())
		authenticated.Use(middleware.RateLimitMiddleware(deps.StrictLimiter))
		{
			authenticated.GET("/voter/:id", deps.Registration.GetVoterInfo)
			authenticated.POST("/vote", deps.Voting.CastVote)
			authenticated.POST("/verify-vote", deps.Voting.VerifyVote)
			authenticated.GET("/results/:electionId", deps.Tally.GetResults)

			// Behavioral duress signal — coercion-resistance feature.
			if deps.Duress != nil {
				authenticated.POST("/voters/:voterID/duress-signal", deps.Duress.SetSignal)
				authenticated.DELETE("/voters/:voterID/duress-signal", deps.Duress.RemoveSignal)
			}
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.AdminAuthMiddleware())
		admin.Use(middleware.RateLimitMiddleware(deps.StrictLimiter))
		{
			admin.POST("/elections", deps.Voting.CreateElection)
			admin.POST("/tally", deps.Tally.TallyVotes)
			admin.GET("/voters", deps.Registration.GetAllVoters)
		}
	}
}
