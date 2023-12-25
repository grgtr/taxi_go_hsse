// internal/app/app_client.go
package app

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"taxi/internal/config"
	"taxi/internal/handlers"
	"taxi/internal/mongodb"
	"time"
)

// App represents the client service application.
type App struct {
	Config   *config.Config
	Server   *http.Server
	Database *mongodb.Database
	Logger   *zap.SugaredLogger
	Tracer   trace.Tracer
}

// NewApp initializes and returns a new instance of the client service application.
func NewApp(cfg *config.Config) *App {
	// Создание логгера
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Logger init error. %v", err)
	}
	sugLog := logger.Sugar()
	sugLog.Info("Logger initialized!")

	// Создание трейсера
	tracer := otel.Tracer("final")
	sugLog.Info("Tracer created")

	// Установка подключения к БД
	db, err := mongodb.NewDatabase(cfg.Database.URI, cfg.Database.Name, "root", "example") // Pass the database name from config
	if err != nil {
		log.Fatal("Error initializing database:", err)
	}
	sugLog.Info("Connection established successfully!")

	sugLog.Info("Creating server")
	server := &http.Server{
		Addr:    cfg.HTTP,
		Handler: nil, // Use nil handler initially
	}
	sugLog.Info("Server created successfully!")
	return &App{
		Config:   cfg,
		Server:   server,
		Database: db,
		Logger:   sugLog,
		Tracer:   tracer,
	}
}

// Start runs the client service.
func (app *App) Start() {
	// Register handlers when needed
	app.Server.Handler = handlers.Router(app.Database)

	go func() {
		log.Println("Server is starting on", app.Config.HTTP)
		if err := app.Server.ListenAndServe(); err != nil {
			app.Logger.Fatalf("Server error. %v", err)
			log.Fatal("Error starting server:", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit
	app.Logger.Info("Shutting down the server...")
	log.Println("Shutting down the server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	app.Database.Close() // Close MongoDB connection before shutting down

	if err := app.Server.Shutdown(ctx); err != nil {
		log.Fatal("Error shutting down server:", err)
	}
	app.Logger.Info("Server has stopped successfully.")
}
