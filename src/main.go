package main

import (
	"log"
	"net/http"
	"os"
	"ricehub/src/errs"
	"ricehub/src/handlers"
	"ricehub/src/repository"
	"ricehub/src/utils"
	"time"

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
	utils.InitJWT(keysDir)

	utils.InitCache(utils.Config.RedisUrl)
	defer utils.CloseCache()

	repository.Init(utils.Config.DatabaseUrl)
	defer repository.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(gin.Recovery(), utils.LoggerMiddleware(logger), errs.ErrorHandler(logger), utils.RateLimitMiddleware(100, time.Minute))

	r.MaxMultipartMemory = utils.Config.MultipartLimit
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

	auth := r.Group("/auth")
	{
		auth.POST("/register", handlers.Register)
		auth.POST("/login", handlers.Login)
		auth.POST("/refresh", utils.PathRateLimitMiddleware(5, utils.Config.JWT.AccessExpiration), handlers.RefreshToken)
	}

	users := r.Group("/users")
	{
		users.GET("/:id/rices/:slug", handlers.GetUserRiceBySlug)

		authedOnly := users.Use(utils.AuthMiddleware)
		// authedOnly.GET("/me", handlers.GetMe)
		authedOnly.GET("/:id", handlers.GetUser)
		authedOnly.DELETE("/:id", handlers.DeleteUser)
		authedOnly.PATCH("/:id/displayName", utils.PathRateLimitMiddleware(5, 24*time.Hour), handlers.UpdateDisplayName)
		authedOnly.PATCH("/:id/password", utils.PathRateLimitMiddleware(5, 24*time.Hour), handlers.UpdatePassword)
		authedOnly.POST("/:id/avatar", utils.PathRateLimitMiddleware(5, 24*time.Hour), handlers.UploadAvatar)
		authedOnly.DELETE("/:id/avatar", utils.PathRateLimitMiddleware(10, 24*time.Hour), handlers.DeleteAvatar)
	}

	tags := r.Group("/tags")
	{
		tags.GET("", handlers.GetAllTags)

		adminOnly := tags.Use(utils.AuthMiddleware, utils.AdminMiddleware)
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

		auth := rices.Use(utils.AuthMiddleware)
		auth.POST("", utils.PathRateLimitMiddleware(10, 24*time.Hour), handlers.CreateRice)
		auth.PATCH("/:id", utils.PathRateLimitMiddleware(5, time.Hour), handlers.UpdateRiceMetadata)
		auth.POST("/:id/dotfiles", utils.PathRateLimitMiddleware(5, time.Hour), handlers.UpdateDotfiles)
		auth.POST("/:id/previews", utils.PathRateLimitMiddleware(25, time.Hour), handlers.AddPreview)
		auth.POST("/:id/star", handlers.AddRiceStar)
		auth.DELETE("/:id/star", handlers.DeleteRiceStar)
		auth.DELETE("/:id/previews/:previewId", handlers.DeletePreview)
		auth.DELETE("/:id", handlers.DeleteRice)
	}

	comments := r.Group("/comments").Use(utils.AuthMiddleware)
	{
		comments.POST("", utils.PathRateLimitMiddleware(10, time.Hour), handlers.AddComment)
		comments.PATCH("/:id", utils.PathRateLimitMiddleware(10, time.Hour), handlers.UpdateComment)
		comments.DELETE("/:id", handlers.DeleteComment)
	}

	reports := r.Group("/reports").Use(utils.AuthMiddleware)
	{
		reports.POST("", utils.PathRateLimitMiddleware(50, 24*time.Hour), handlers.CreateReport)

		adminOnly := reports.Use(utils.AdminMiddleware)
		adminOnly.GET("", handlers.GetAllReports)
		adminOnly.GET("/:reportId", handlers.GetReport)
		adminOnly.POST("/:reportId/close", handlers.CloseReport)
	}
}
