package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	_ "github.com/middelmatigheid/subscriptions-api/docs"
	"github.com/middelmatigheid/subscriptions-api/internal/config"
	"github.com/middelmatigheid/subscriptions-api/internal/database"
	"github.com/middelmatigheid/subscriptions-api/internal/handlers"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Custom response writer for catching logs
type Writer struct {
	gin.ResponseWriter
	body []byte
}

func (r *Writer) Write(b []byte) (int, error) {
	r.body = b
	return r.ResponseWriter.Write(b)
}

func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		writer := &Writer{
			ResponseWriter: c.Writer,
			body:           []byte{},
		}
		c.Writer = writer

		c.Next()
		if writer.Status() >= 200 && writer.Status() < 300 {
			if len(writer.body) > 0 {
				logger.Info(strconv.Itoa(writer.Status()), slog.String("url", c.Request.URL.Path), slog.String("method", c.Request.Method), slog.String("info", string(writer.body)))
			} else {
				logger.Info(strconv.Itoa(writer.Status()), slog.String("url", c.Request.URL.Path), slog.String("method", c.Request.Method))
			}
		} else {
			if len(writer.body) > 0 {
				logger.Error(strconv.Itoa(writer.Status()), slog.String("url", c.Request.URL.Path), slog.String("method", c.Request.Method), slog.String("info", string(writer.body)))
			} else {
				logger.Error(strconv.Itoa(writer.Status()), slog.String("url", c.Request.URL.Path), slog.String("method", c.Request.Method))
			}
		}
	}
}

// Server graceful shutdown
func gracefulShutdown(server *http.Server, db *database.Database, logger *slog.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("Received shutdown signal, starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	}

	if err := db.Close(); err != nil {
		logger.Error("Database close error", slog.String("error", err.Error()))
	}
	logger.Info("Server gracefully stopped")
}

// @title Subscriptions API
// @version 1.0
// @description It is just a simple API to manage subscriptions
// @host localhost:8080
// @BasePath /subscriptions/
func main() {
	// Configuring logger
	logDir := "/logs"
	os.MkdirAll(logDir, 0755)
	file, err := os.OpenFile(
		filepath.Join(logDir, "app.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return
	}
	logger := slog.New(slog.NewJSONHandler(file, nil))

	// Getting config
	config, err := config.GetConfig()
	if err != nil {
		logger.Error("Error while getting config", slog.String("error", err.Error()))
		return
	}

	// Connecting to the database
	db, err := database.Connect(config, logger)
	if err != nil {
		logger.Error("Error while connecting to the database", slog.String("error", err.Error()))
		return
	}

	// Setting up the handler
	handler, err := handlers.NewHandler(config, db)
	if err != nil {
		logger.Error("Error while creating the handler", slog.String("error", err.Error()))
		return
	}
	// Setting up the endpoints
	server := gin.Default()
	server.Use(Logger(logger))
	subscriptions := server.Group("/subscriptions")
	subscriptions.POST("/create", handler.Create)
	subscriptions.GET("/read", handler.Read)
	subscriptions.PUT("/update", handler.Update)
	subscriptions.PUT("/patch", handler.Patch)
	subscriptions.DELETE("/delete", handler.Delete)
	subscriptions.GET("/list", handler.List)
	subscriptions.GET("/summary", handler.Summary)
	subscriptions.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Starting up the server
	go func() {
		logger.Info("Server starting", slog.String("port", config.Port), slog.String("swagger", "http://localhost:"+config.Port+"/subscriptions/swagger/index.html"))
		if err := server.Run(":" + config.Port); err != nil {
			logger.Error("Server failed to start", slog.String("error", err.Error()))
		}
	}()

	// Graceful shutdown
	gracefulShutdown(&http.Server{
		Addr:    ":" + config.Port,
		Handler: server,
	}, db, logger)
}
