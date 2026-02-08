package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	_ "middelmatigheid/docs"
	"middelmatigheid/internal/database"
	. "middelmatigheid/internal/handlers"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

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

// @title Subscriptions API
// @version 1.0
// @description It is a simple API to manage subscriptions
// @host localhost:8080
// @BasePath /subscriptions/
func main() {
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
	db, err := database.Connect(logger)
	if err != nil {
		logger.Error("Error while connecting to the database", slog.String("function", "main"), slog.String("error", err.Error()))
		return
	}
	defer db.Close()

	// if err := db.MigrateDown(); err != nil {
	// 	logger.Error("Failed to run migrations", slog.String("error", err.Error()))
	// 	return
	// }
	if err := db.Migrate(); err != nil {
		logger.Error("Failed to run migrations", slog.String("error", err.Error()))
		return
	}
	logger.Info("Migrated successfully", slog.String("function", "main"))

	handler := &Handler{Db: db}
	server := gin.Default()
	server.Use(Logger(logger))
	subscriptions := server.Group("/subscriptions")
	subscriptions.POST("/create", handler.Create)
	subscriptions.GET("/read", handler.Read)
	subscriptions.PUT("/update", handler.Update)
	subscriptions.DELETE("/delete", handler.Delete)
	subscriptions.GET("/list", handler.List)
	subscriptions.GET("/summary", handler.Summary)
	subscriptions.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	logger.Info("Server starting", slog.String("port", "8080"), slog.String("swagger", "http://localhost:8080/subscriptions/swagger/index.html"))
	if err := server.Run(":8080"); err != nil {
		logger.Error("Server failed to start", slog.String("function", "main"), slog.String("error", err.Error()))
	}
}
