package trip_svc

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"os"
	"sync"
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
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			m, err := app.reader.ReadMessage(context.Background())
			if err != nil {
				fmt.Println("There is an error." + err.Error())
			}
			var msg trip.Event
			err = kafka.Unmarshal(m.Value, &msg)
		}
	}()

	go func() {
		defer wg.Done()
		for {
			//
		}
	}()

	wg.Wait()
}

func (app *App) Stop(ctx context.Context) error {
	<-ctx.Done()

	defer app.reader.Close()
	defer app.writerToClient.Close()
	defer app.writerToDriver.Close()

	done := make(chan bool)
	app.logger.Info("Server is shutting down...")

	// остановка приложения, gracefully shutdown
	go func() {
		app.logger.Info("Server stopped")
		close(done)
	}()

	<-done
	return nil
}
