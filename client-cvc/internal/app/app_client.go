// internal/app/app_client.go
package app

import (
	"context"
	"encoding/json"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"taxi/internal/config"
	"taxi/internal/handlers"
	"taxi/internal/models"
	"taxi/internal/mongodb"
	kfk "taxi/pkg/kafka"
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

	go func() {
		for {
			app.listner()
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

func (app *App) listner() {
	ctx, span := app.Tracer.Start(context.Background(), "Iteration")
	defer span.End()
	connClient, err := kfk.ConnectKafka(ctx, "kafka:9092", "trip-client-topic", 0)

	if err != nil {
		app.Logger.Fatalf("Kafka connect error. %v", err)
		return
	}
	bytes, err := kfk.ReadFromTopic(connClient)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Kafka read error")
		app.Logger.Errorf("Kafka read error. %v", err)
		return
	}
	var response models.Request
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Unmarshal error")
		app.Logger.Errorf("Unmarshal error. %v", err)
		return
	}
	if response.DataContentType != "application/json" {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Data type error")
		app.Logger.Errorf("Data type error. %v", err)
		return
	}

	switch response.Type {
	case "trip.event.accepted":
		var eventData models.EventAcceptData
		err := json.Unmarshal(response.Data, &eventData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data unmarshal error")
			app.Logger.Errorf("Data unmarshal error. %v", err)
			return
		}
		err = app.Database.UpdateTripStatus(eventData.TripId, "DRIVER_FOUND") // Implement this function in your mongodb package
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Get trip error")
			app.Logger.Errorf("Get trip error. %v", err)
			return
		}
	case "trip.event.started":
		var eventData models.EventStartData
		err := json.Unmarshal(response.Data, &eventData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data unmarshal error")
			app.Logger.Errorf("Data unmarshal error. %v", err)
			return
		}
		err = app.Database.UpdateTripStatus(eventData.TripId, "STARTED") // Implement this function in your mongodb package
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Get trip error")
			app.Logger.Errorf("Get trip error. %v", err)
			return
		}
	case "trip.event.ended":
		var eventData models.EventEndData
		err := json.Unmarshal(response.Data, &eventData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data unmarshal error")
			app.Logger.Errorf("Data unmarshal error. %v", err)
			return
		}
		err = app.Database.UpdateTripStatus(eventData.TripId, "ENDED") // Implement this function in your mongodb package
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Get trip error")
			app.Logger.Errorf("Get trip error. %v", err)
			return
		}
	}
}
