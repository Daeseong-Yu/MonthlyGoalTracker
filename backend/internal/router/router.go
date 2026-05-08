package router

import (
	"fmt"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/handler"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/repository"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(database *gorm.DB, cfg config.Config) *gin.Engine {
	r := gin.Default()
	configureTrustedProxies(r, cfg.TrustedProxies)

	sessionConfig := cfg.Session.WithDefaults()
	authFlowConfig := cfg.AuthFlow.WithDefaults()
	userRepo := repository.NewUserRepository(database)
	sessionRepo := repository.NewSessionRepository(database)
	goalRepo := repository.NewGoalRepository(database)
	memoRepo := repository.NewDailyMemoRepository(database)
	checkRepo := repository.NewGoalCheckRepository(database)

	authService := service.NewAuthService(
		userRepo,
		sessionRepo,
		time.Duration(sessionConfig.TTLHours)*time.Hour,
		authFlowConfig.LegacyClaimToken,
	)
	emailConfig := cfg.Email.WithDefaults()
	if emailConfig.Enabled() {
		emailSender := service.NewSMTPVerificationEmailSender(emailConfig)
		authService.EnableEmailVerification(
			repository.NewEmailVerificationRepository(database),
			emailSender,
			time.Duration(authFlowConfig.EmailVerificationTTLHours)*time.Hour,
		)
		authService.EnablePasswordReset(
			repository.NewPasswordResetRepository(database),
			emailSender,
			time.Duration(authFlowConfig.PasswordResetTTLHours)*time.Hour,
		)
	}
	goalService := service.NewGoalService(goalRepo)
	memoService := service.NewMemoService(memoRepo)
	checkService := service.NewCheckService(goalRepo, checkRepo)
	monthService := service.NewMonthService(goalRepo, memoRepo, checkRepo)

	authHandler := handler.NewAuthHandler(authService, sessionConfig)
	goalHandler := handler.NewGoalHandler(goalService)
	memoHandler := handler.NewMemoHandler(memoService)
	checkHandler := handler.NewCheckHandler(checkService)
	monthHandler := handler.NewMonthHandler(monthService)

	r.GET("/api/health", handler.Health)

	apiRoot := r.Group("/api")
	if cfg.Auth.Enabled() {
		apiRoot.Use(basicAuthMiddleware(cfg.Auth))
	}
	apiRoot.GET("/bootstrap", authHandler.Bootstrap)

	signupLimiter := newAuthRateLimiterWithMaxKeys(
		authFlowConfig.SignupRateLimitPerMinute,
		time.Minute,
		authFlowConfig.RateLimitMaxBuckets,
	)
	loginLimiter := newAuthRateLimiterWithMaxKeys(
		authFlowConfig.LoginRateLimitPerMinute,
		time.Minute,
		authFlowConfig.RateLimitMaxBuckets,
	)

	authRoutes := apiRoot.Group("/auth")
	authRoutes.POST("/signup", signupLimiter.Middleware("signup"), authHandler.SignUp)
	authRoutes.POST("/login", loginLimiter.Middleware("login"), authHandler.Login)
	authRoutes.POST("/verify-email", loginLimiter.Middleware("verify-email"), authHandler.VerifyEmail)
	authRoutes.POST("/password-reset/request", signupLimiter.Middleware("password-reset-request"), authHandler.RequestPasswordReset)
	authRoutes.POST("/password-reset/confirm", loginLimiter.Middleware("password-reset-confirm"), authHandler.ResetPassword)

	protectedAuthRoutes := authRoutes.Group("")
	protectedAuthRoutes.Use(sessionMiddleware(authService, sessionConfig.CookieName))
	protectedAuthRoutes.GET("/me", authHandler.Me)
	protectedAuthRoutes.Use(csrfMiddleware(sessionConfig.CSRFCookieName))
	protectedAuthRoutes.POST("/logout", authHandler.Logout)
	protectedAuthRoutes.POST("/logout/others", authHandler.LogoutOtherSessions)
	protectedAuthRoutes.PATCH("/me/locale", authHandler.UpdateLocale)
	protectedAuthRoutes.POST("/password/change", loginLimiter.AuthenticatedUserMiddleware("password-change"), authHandler.ChangePassword)

	api := apiRoot.Group("")
	api.Use(sessionMiddleware(authService, sessionConfig.CookieName))
	api.Use(csrfMiddleware(sessionConfig.CSRFCookieName))
	api.POST("/months/:month/ensure", monthHandler.Ensure)
	api.GET("/months/:month", monthHandler.Get)
	api.POST("/months/:month/goals", goalHandler.Create)
	api.PATCH("/goals/:id", goalHandler.Update)
	api.POST("/goals/:id/deactivate", goalHandler.Deactivate)
	api.PUT("/memos/:date", memoHandler.Save)
	api.PUT("/checks", checkHandler.Set)

	return r
}

func configureTrustedProxies(engine *gin.Engine, trustedProxies []string) {
	if err := engine.SetTrustedProxies(trustedProxies); err != nil {
		panic(fmt.Sprintf("invalid trusted proxy config: %v", err))
	}
}
