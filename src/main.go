package main

import (
	"log"
	"net/http"
	"os"
	"ricehub/src/errs"
	"ricehub/src/handlers"
	"ricehub/src/repository"
	"ricehub/src/security"
	"ricehub/src/utils"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const configPath = "config.toml"
const keysDir = "keys"

func main() {
	logger := setupLogger()
	defer logger.Sync()

	utils.InitConfig(configPath)
	utils.InitValidator()
	security.InitJWT(keysDir)

	if utils.Config.DisableRateLimits {
		logger.Warn("Rate limits disabled! Is it intentional?")
	}
	if utils.Config.Maintenance {
		logger.Warn("Maintenance mode toggled! Is it intentional?")
	}

	utils.InitCache(utils.Config.RedisUrl)
	defer utils.CloseCache()

	repository.Init(utils.Config.DatabaseUrl)
	defer repository.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	corsConfig := cors.Config{
		AllowOrigins:     []string{utils.Config.CorsOrigin},
		AllowMethods:     []string{"GET", "POST", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "Set-Cookie"},
		AllowCredentials: true,
	}

	r.Use(gin.Recovery(), cors.New(corsConfig), security.LoggerMiddleware(logger), errs.ErrorHandler(logger), security.RateLimitMiddleware(100, time.Minute))

	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("Failed to set trusted proxies: %v", err)
	}

	setupRoutes(r)

	logger.Info("API is available on port :3000")
	if err := r.Run(":3000"); err != nil {
		log.Fatalf("Failed to start the API: %v\n", err)
	}
}

func setupLogger() *zap.Logger {
	// json file logger with rotation
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "./logs/gin.json",
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     7, // in days
	})
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// console logger
	// https://last9.io/blog/zap-logger/
	encodeCfg := zap.NewDevelopmentEncoderConfig()
	encodeCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encodeCfg.EncodeTime = func(t time.Time, pae zapcore.PrimitiveArrayEncoder) {
		pae.AppendString(t.Format("2006/01/02 15:04:05"))
	}

	consoleEncoder := zapcore.NewConsoleEncoder(encodeCfg)
	consoleWriter := zapcore.AddSync(os.Stdout)

	// levels
	level := zap.InfoLevel
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleWriter, level),
		zapcore.NewCore(fileEncoder, fileWriter, level),
	)

	logger := zap.New(core)
	zap.ReplaceGlobals(logger)
	return logger
}

func setupRoutes(r *gin.Engine) {
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "The requested resource could not be found on this server!"})
	})

	r.Static("/public", "./public")

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"maintenance": utils.Config.Maintenance})
	})
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "I'm working and responding!"})
	})

	auth := r.Group("/auth")
	{
		auth.POST("/register", security.MaintenanceMiddleware(), handlers.Register)
		auth.POST("/login", handlers.Login)
		auth.POST("/refresh", security.PathRateLimitMiddleware(5, 1*time.Minute), handlers.RefreshToken)
		auth.POST("/logout", handlers.LogOut)
	}

	users := r.Group("/users")
	{
		defaultRL := security.PathRateLimitMiddleware(5, 1*time.Minute)
		users.GET("", handlers.FetchUsers)
		users.GET("/:id/rices", defaultRL, handlers.FetchUserRices)
		users.GET("/:id/rices/:slug", defaultRL, handlers.GetUserRiceBySlug)

		authedOnly := users.Use(security.AuthMiddleware)
		authedOnly.GET("/:id", defaultRL, handlers.GetUserById)
		authedOnly.DELETE("/:id", security.MaintenanceMiddleware(), defaultRL, handlers.DeleteUser) // should this be affected by maintenance mode?
		authedOnly.PATCH("/:id/displayName", security.MaintenanceMiddleware(), security.PathRateLimitMiddleware(5, 24*time.Hour), handlers.UpdateDisplayName)
		authedOnly.PATCH("/:id/password", security.MaintenanceMiddleware(), security.PathRateLimitMiddleware(5, 24*time.Hour), handlers.UpdatePassword)
		authedOnly.POST("/:id/avatar", security.MaintenanceMiddleware(), security.FileSizeLimitMiddleware(utils.Config.Limits.UserAvatarSizeLimit), security.PathRateLimitMiddleware(5, 24*time.Hour), handlers.UploadAvatar)
		authedOnly.DELETE("/:id/avatar", security.MaintenanceMiddleware(), security.PathRateLimitMiddleware(10, 24*time.Hour), handlers.DeleteAvatar)

		adminOnly := users.Use(security.AdminMiddleware)
		adminOnly.POST("/:id/ban", handlers.BanUser)
		adminOnly.DELETE("/:id/ban", handlers.UnbanUser)
	}

	profiles := r.Group("/profiles")
	{
		profiles.GET("/:username", handlers.GetUserProfile)
	}

	tags := r.Group("/tags")
	{
		tags.GET("", handlers.GetAllTags)

		adminOnly := tags.Use(security.AuthMiddleware, security.AdminMiddleware)
		adminOnly.POST("", handlers.CreateTag)
		adminOnly.PATCH("/:id", handlers.UpdateTag)
		adminOnly.DELETE("/:id", handlers.DeleteTag)
	}

	rices := r.Group("/rices")
	{
		rices.GET("", handlers.FetchRices)
		rices.GET("/:id", handlers.GetRiceById)
		rices.GET("/:id/comments", handlers.GetRiceComments)
		rices.GET("/:id/dotfiles", handlers.DownloadDotfiles)

		auth := rices.Use(security.AuthMiddleware)
		// This is actually unreadable, I feel like Im gonna have a seizure trying to compherend this line
		auth.POST("", security.MaintenanceMiddleware(), security.FileSizeLimitMiddleware(utils.Config.Limits.DotfilesSizeLimit+int64(utils.Config.Limits.MaxPreviewsPerRice)*utils.Config.Limits.PreviewSizeLimit), security.PathRateLimitMiddleware(5, 24*time.Hour), handlers.CreateRice)
		auth.PATCH("/:id", security.MaintenanceMiddleware(), security.PathRateLimitMiddleware(5, time.Hour), handlers.UpdateRiceMetadata)
		auth.POST("/:id/dotfiles", security.MaintenanceMiddleware(), security.FileSizeLimitMiddleware(utils.Config.Limits.DotfilesSizeLimit), security.PathRateLimitMiddleware(5, time.Hour), handlers.UpdateDotfiles)
		auth.POST("/:id/previews", security.MaintenanceMiddleware(), security.FileSizeLimitMiddleware(utils.Config.Limits.PreviewSizeLimit), security.PathRateLimitMiddleware(25, time.Hour), handlers.AddPreview)
		auth.POST("/:id/star", security.MaintenanceMiddleware(), handlers.AddRiceStar)
		auth.DELETE("/:id/star", security.MaintenanceMiddleware(), handlers.DeleteRiceStar)
		auth.DELETE("/:id/previews/:previewId", security.MaintenanceMiddleware(), handlers.DeletePreview)
		auth.DELETE("/:id", security.MaintenanceMiddleware(), handlers.DeleteRice)
	}

	comments := r.Group("/comments").Use(security.AuthMiddleware)
	{
		comments.GET("", security.AdminMiddleware, handlers.GetRecentComments)

		comments.POST("", security.MaintenanceMiddleware(), security.PathRateLimitMiddleware(10, time.Hour), handlers.AddComment)
		comments.GET("/:id", security.PathRateLimitMiddleware(10, time.Minute), handlers.GetCommentById)
		comments.PATCH("/:id", security.MaintenanceMiddleware(), security.PathRateLimitMiddleware(10, time.Hour), handlers.UpdateComment)
		comments.DELETE("/:id", security.MaintenanceMiddleware(), handlers.DeleteComment)
	}

	reports := r.Group("/reports").Use(security.AuthMiddleware)
	{
		reports.POST("", security.PathRateLimitMiddleware(50, 24*time.Hour), handlers.CreateReport)

		adminOnly := reports.Use(security.AdminMiddleware)
		adminOnly.GET("", handlers.FetchReports)
		adminOnly.GET("/:reportId", handlers.GetReportById)
		adminOnly.POST("/:reportId/close", handlers.CloseReport)
	}

	admin := r.Group("/admin").Use(security.AuthMiddleware, security.AdminMiddleware)
	{
		admin.GET("/stats", handlers.ServiceStatistics)
	}

	webVars := r.Group("/vars")
	{
		webVars.GET("/:key", security.PathRateLimitMiddleware(5, 1*time.Minute), handlers.GetWebsiteVariable)
	}

	links := r.Group("/links")
	{
		links.GET("/:name", security.PathRateLimitMiddleware(5, 1*time.Minute), handlers.GetLinkByName)
	}
}
