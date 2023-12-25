package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	"trip/internal/models"
	kfk "trip/pkg/kafka"
)

const configPath = "./config/config.json"

type App struct {
	ToClientTopic         *kafka.Conn
	ToDriverTopic         *kafka.Conn
	FromClientDriverTopic *kafka.Conn
	Config                *models.Config
	Logger                *zap.Logger
	Tracer                trace.Tracer
	Postgres              *sql.DB
}

func NewApp(ctx context.Context) *App {
	// Создание логгера
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Logger init error. %v", err)
		return nil
	}
	sugLog := logger.Sugar()
	sugLog.Info("Logger initialized")

	// Создание трейсера
	tracer := otel.Tracer("final")
	sugLog.Info("Tracer created")

	// Инициализация конфига
	sugLog.Info("Initializing config")
	config, err := initConfig()
	if err != nil {
		sugLog.Fatalf("Config init error. %v", err)
		return nil
	}

	// Инициализация postgres
	sugLog.Info("Initializing postgres")
	postgres, err := initPostgres(config.PostgresHost, config.PostgresPort, config.PostgresUser, config.PostgresPass)
	if err != nil {
		sugLog.Fatalf("Postgres init error. %v", err)
		return nil
	}
	sugLog.Info("Postgres connected")

	// Подключение к Kafka
	connClient, err := kfk.ConnectKafka(ctx, config.KafkaAddress, "trip-client-topic", 0)
	if err != nil {
		sugLog.Fatalf("Kafka connect error. %v", err)
		return nil
	}
	connDriver, err := kfk.ConnectKafka(ctx, config.KafkaAddress, "trip-driver-topic", 0)
	if err != nil {
		sugLog.Fatalf("Kafka connect error. %v", err)
		return nil
	}
	connClDrv, err := kfk.ConnectKafka(ctx, config.KafkaAddress, "driver-client-trip-topic", 0)
	if err != nil {
		sugLog.Fatalf("Kafka connect error. %v", err)
		return nil
	}

	// Создание объекта App
	sugLog.Info("Creating app")
	app := App{
		ToClientTopic:         connClient,
		ToDriverTopic:         connDriver,
		FromClientDriverTopic: connClDrv,
		Config:                config,
		Logger:                logger,
		Tracer:                tracer,
		Postgres:              postgres,
	}
	sugLog.Info("App created")

	return &app
}

// initConfig инициализирует конфиг
func initConfig() (*models.Config, error) {
	// Получение информации о файле
	stat, err := os.Stat(configPath)
	if err != nil {
		return nil, err
	}

	// Открытие файла
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}

	// Считывание bytes
	data := make([]byte, stat.Size())
	_, err = file.Read(data)
	if err != nil {
		return nil, err
	}

	// Десериализация в конфиг
	var config models.Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// sendPostgres сохраняет запись в postgres
func sendPostgres(db *sql.DB, trip *models.Trip) error {
	// SQL-запрос
	query := `INSERT INTO trips_history
  	(tripid, source, type, datacontenttype, time, driverid, reason, offerid, price, status, locfrom, locto)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	// Сериализация объектов в string
	bytes, err := json.Marshal(trip.Price)
	if err != nil {
		return err
	}
	price := string(bytes)

	bytes, err = json.Marshal(trip.From)
	if err != nil {
		return err
	}
	from := string(bytes)

	bytes, err = json.Marshal(trip.To)
	if err != nil {
		return err
	}
	to := string(bytes)

	// Выполнение запроса
	_, err = db.Exec(query, trip.Id, trip.Source, trip.Type, trip.DataContentType, trip.Time, trip.DriverId, trip.Reason, trip.OfferId, price, trip.Status, from, to)
	if err != nil {
		return err
	}

	return nil
}

// initPostgres инициализирует базу данных PostgreSQL
func initPostgres(host string, port string, user string, password string) (*sql.DB, error) {
	// Строка подключения к базе данных PostgreSQL
	connStr := fmt.Sprintf("host=%v port=%v user=%v dbname=postgres sslmode=disable password=%v", host, port, user, password)

	// Открываем соединение с базой данных
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Проверяем соединение с базой данных
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	// Создание таблицы в postgres
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS trips_history (
			"id" serial PRIMARY KEY,
			"tripid" VARCHAR(255),
    		"source" VARCHAR(255),
    		"type" VARCHAR(255),
    		"datacontenttype" VARCHAR(255),
    		"time" VARCHAR(255),
    		"driverid" VARCHAR(255),
    		"reason" VARCHAR(255),
    		"offerid" VARCHAR(255),
    		"price" VARCHAR(255),
    		"status" VARCHAR(255),
    		"locfrom" VARCHAR(255),
    		"locto" VARCHAR(255)
	)`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// iteration слушает kafka и обрабатывает сообщения
func (a *App) Start(ctx context.Context) {
	for {
		a.iteration(ctx)
		select {
		case <-ctx.Done():
			break
		default:
		}
	}
}

func (a *App) iteration(ctx context.Context) {
	ctx, span := a.Tracer.Start(ctx, "Iteration")
	defer span.End()

	// Чтение из Kafka
	bytes, err := kfk.ReadFromTopic(a.FromClientDriverTopic)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Kafka read error")
		a.Logger.Sugar().Errorf("Kafka read error. %v", err)
		return
	}

	// Десериализация запроса
	var request models.Request
	err = json.Unmarshal(bytes, &request)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Unmarshal error")
		a.Logger.Sugar().Errorf("Unmarshal error. %v", err)
		return
	}

	// Проверка на тип Data
	if request.DataContentType != "application/jsonapplication/json" {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Data type error")
		a.Logger.Sugar().Errorf("Data type error. %v", err)
		return
	}
	fmt.Println("from client ", request)
	time.Sleep(time.Second * 5)

	response := models.Request{
		Id:              request.Id,
		Source:          "/trip",
		Type:            "", // будет заполнено далее
		DataContentType: "application/json",
		Time:            request.Time,
		Data:            nil, // будет заполнено далее
	}
	var conn []*kafka.Conn
	switch request.Type {
	case "trip.command.accept":
		response.Type = "trip.event.accepted"
		conn = make([]*kafka.Conn, 1)
		conn[0] = a.ToClientTopic

		// Десериализация commandData
		var commandData models.CommandAcceptData
		err := json.Unmarshal(request.Data, &commandData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data unmarshal error")
			a.Logger.Sugar().Errorf("Data unmarshal error. %v", err)
			return
		}

		// Сохранение в Postgres
		a.Logger.Info("Writing to postgres")
		err = sendPostgres(a.Postgres, &models.Trip{
			Id:              response.Id,
			Source:          response.Source,
			Type:            response.Type,
			DataContentType: response.DataContentType,
			Time:            response.Time,
			DriverId:        commandData.DriverId,
			Status:          "DRIVER_FOUND",
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Postgres write error")
			a.Logger.Sugar().Errorf("Postgres write error. %v", err)
			return
		}
		a.Logger.Info("Written correctly")

		// Создание ответной data
		eventData := models.EventAcceptData{
			TripId: commandData.TripId,
		}

		// Сериализация eventData
		response.Data, err = json.Marshal(eventData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data marshal error")
			a.Logger.Sugar().Errorf("Data marshal error. %v", err)
			return
		}
	case "trip.command.cancel":
		response.Type = "trip.event.canceled"
		conn = make([]*kafka.Conn, 1)
		conn[0] = a.ToDriverTopic

		// Десериализация commandData
		var commandData models.CommandCancelData
		err := json.Unmarshal(request.Data, &commandData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data unmarshal error")
			a.Logger.Sugar().Errorf("Data unmarshal error. %v", err)
			return
		}

		// Сохранение в Postgres
		a.Logger.Info("Writing to postgres")
		err = sendPostgres(a.Postgres, &models.Trip{
			Id:              response.Id,
			Source:          response.Source,
			Type:            response.Type,
			DataContentType: response.DataContentType,
			Time:            response.Time,
			Reason:          commandData.Reason,
			Status:          "CANCELED",
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Postgres write error")
			a.Logger.Sugar().Errorf("Postgres write error. %v", err)
			return
		}
		a.Logger.Info("Written correctly")

		// Создание ответной data
		eventData := models.EventCancelData{
			TripId: commandData.TripId,
		}

		// Сериализация eventData
		response.Data, err = json.Marshal(eventData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data marshal error")
			a.Logger.Sugar().Errorf("Data marshal error. %v", err)
			return
		}
	case "trip.command.create":
		response.Type = "trip.event.created"
		conn = make([]*kafka.Conn, 2)
		conn[0] = a.ToDriverTopic
		conn[1] = a.ToClientTopic

		// Десериализация commandData
		var commandData models.CommandCreateData
		err := json.Unmarshal(request.Data, &commandData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data unmarshal error")
			a.Logger.Sugar().Errorf("Data unmarshal error. %v", err)
			return
		}

		// Получение информации из OfferingService
		order, err := a.getOffer(commandData.OfferId)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Offer get error")
			a.Logger.Sugar().Errorf("Offer get error. %v", err)
			return
		}

		// Создание ответной data
		eventData := models.EventCreateData{
			TripId:  request.Id,
			OfferId: commandData.OfferId,
			Price:   order.Price,
			Status:  "DRIVER_SEARCH",
			From:    order.From,
			To:      order.To,
		}

		// Сохранение в Postgres
		a.Logger.Info("Writing to postgres")
		err = sendPostgres(a.Postgres, &models.Trip{
			Id:              response.Id,
			Source:          response.Source,
			Type:            response.Type,
			DataContentType: response.DataContentType,
			Time:            response.Time,
			OfferId:         eventData.OfferId,
			Price:           eventData.Price,
			From:            eventData.From,
			To:              eventData.To,
			Status:          "DRIVER_SEARCH",
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Postgres write error")
			a.Logger.Sugar().Errorf("Postgres write error. %v", err)
			return
		}
		a.Logger.Info("Written correctly")

		// Сериализация eventData
		response.Data, err = json.Marshal(eventData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data marshal error")
			a.Logger.Sugar().Errorf("Data marshal error. %v", err)
			return
		}
	case "trip.command.end":
		response.Type = "trip.event.ended"
		conn = make([]*kafka.Conn, 1)
		conn[0] = a.ToClientTopic

		// Десериализация commandData
		var commandData models.CommandEndData
		err := json.Unmarshal(request.Data, &commandData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data unmarshal error")
			a.Logger.Sugar().Errorf("Data unmarshal error. %v", err)
			return
		}

		// Сохранение в Postgres
		a.Logger.Info("Writing to postgres")
		err = sendPostgres(a.Postgres, &models.Trip{
			Id:              response.Id,
			Source:          response.Source,
			Type:            response.Type,
			DataContentType: response.DataContentType,
			Time:            response.Time,
			Status:          "ENDED",
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Postgres write error")
			a.Logger.Sugar().Errorf("Postgres write error. %v", err)
			return
		}
		a.Logger.Info("Written correctly")

		// Создание ответной data
		eventData := models.EventEndData{
			TripId: commandData.TripId,
		}

		// Сериализация eventData
		response.Data, err = json.Marshal(eventData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data marshal error")
			a.Logger.Sugar().Errorf("Data marshal error. %v", err)
			return
		}
	case "trip.command.start":
		response.Type = "trip.event.started"
		conn = make([]*kafka.Conn, 1)
		conn[0] = a.ToClientTopic

		// Десериализация commandData
		var commandData models.CommandCancelData
		err := json.Unmarshal(request.Data, &commandData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data unmarshal error")
			a.Logger.Sugar().Errorf("Data unmarshal error. %v", err)
			return
		}

		// Сохранение в Postgres
		a.Logger.Info("Writing to postgres")
		err = sendPostgres(a.Postgres, &models.Trip{
			Id:              response.Id,
			Source:          response.Source,
			Type:            response.Type,
			DataContentType: response.DataContentType,
			Time:            response.Time,
			Reason:          commandData.Reason,
			Status:          "STARTED",
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Postgres write error")
			a.Logger.Sugar().Errorf("Postgres write error. %v", err)
			return
		}
		a.Logger.Info("Written correctly")

		// Создание ответной data
		eventData := models.EventCancelData{
			TripId: commandData.TripId,
		}

		// Сериализация eventData
		response.Data, err = json.Marshal(eventData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Data marshal error")
			a.Logger.Sugar().Errorf("Data marshal error. %v", err)
			return
		}
	}

	// Сериализация response
	bytes, err = json.Marshal(response)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Data marshal error")
		a.Logger.Sugar().Errorf("Data marshal error. %v", err)
		return
	}

	// Запись в Kafka
	for _, top := range conn {
		err = kfk.SendToTopic(top, bytes)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Kafka write error")
			a.Logger.Sugar().Errorf("Kafka write error. %v", err)
			return
		}
	}

	a.Logger.Info("Message sent")
}

// getOffer получает информацию о заказе из OfferingService
func (a *App) getOffer(offerID string) (*models.Order, error) {
	// Запрос к OfferingService
	resp, err := http.Get("http://" + a.Config.OfferingAddress + "/offers/" + offerID)
	if err != nil {
		return nil, err
	}

	// Чтение Body
	fmt.Println(resp)
	bytes, err := io.ReadAll(resp.Body)
	fmt.Println(bytes)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Десериализация
	var order models.Order
	err = json.Unmarshal(bytes, &order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}
