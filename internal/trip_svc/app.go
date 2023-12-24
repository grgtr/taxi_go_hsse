package trip_svc

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"log"
	"os"
	"taxi/internal/trip_svc/app/config"
	"taxi/internal/trip_svc/app/repository"
	"taxi/internal/trip_svc/app/repository/trip"
	"taxi/internal/trip_svc/app/service"
)

type App struct {
	config         *config.Config
	storage        service.TripRepository
	logger         *zap.SugaredLogger
	reader         *kafka.Reader
	writerToClient kafka.Writer
	writerToDriver kafka.Writer
}

func NewApp() *App {
	return &App{}
}

func (app *App) Init(cfg *config.Config, logger *zap.SugaredLogger) {
	app.config = cfg
	//"host=localhost user=taxi_admin password=pswd dbname=trip_db port=9920 sslmode=disable TimeZone=Europe/Moscow"
	app.storage = trip.NewRepository(repository.NewGorm(app.config.ConfigDB.DSN))
	app.logger = logger

	app.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{os.Getenv("kafkaURL")},
		Topic:   "upd_trip",
		GroupID: os.Getenv("GroupID"),
	})
	app.writerToClient = kafka.Writer{
		Addr:  kafka.TCP(os.Getenv("kafkaURL")),
		Topic: "to-client",
	}
	app.writerToDriver = kafka.Writer{
		Addr:  kafka.TCP(os.Getenv("kafkaURL")),
		Topic: "to-driver",
	}
}

func (app *App) Start() {
	m, err := app.reader.ReadMessage(context.Background())
	if err != nil {
		fmt.Println("There is an error." + err.Error())
	}
}

func (app *App) Stop(ctx context.Context) error {
	<-ctx.Done()

	done := make(chan bool)
	log.Printf("Server is shutting down...")

	// остановка приложения, gracefully shutdown
	go func() {

		log.Printf("Server stopped")
		close(done)
	}()

	<-done
	return nil
}
