package trip_svc

import (
	"context"
	"flag"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"log"
	"net"
	"os/signal"
	"strconv"
	"syscall"
	"taxi/internal/trip_svc"
	"taxi/internal/trip_svc/app/config"
)

func main() {
	// Logger
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Info("Starting reading config")
	cfg, err := ReadConfig()
	if err != nil {
		sugar.Error("Config init error")
		log.Fatal(err)
	}
	sugar.Info("Read config")

	sugar.Info("Initializing TripSvc")
	application := trip_svc.NewApp()
	application.Init(cfg, sugar)
	application.Start()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	application.Stop(ctx)
}

func ReadConfig() (*config.Config, error) {
	var configPath string
	flag.StringVar(&configPath, "c", "task3/configs/config.yaml", "set config path")
	flag.Parse()
	return config.NewConfig(configPath)
}

func InitController() {
	var (
		brokers    = flag.String("b", "127.0.0.1:29092", "Brokers list")
		partitions = flag.Int("p", 1, "paritions numbers")
		replicas   = flag.Int("r", 1, "replicas numbers")
	)
	flag.Parse()
	conn, err := kafka.DialContext(context.Background(), "tcp", *brokers)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		panic(err.Error())
	}
	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		panic(err.Error())
	}
	defer controllerConn.Close()

	topicNames := []string{
		"upd_trip",
		"info_to_driver",
		"info_to_client",
	}
	for _, value := range topicNames {
		err = controllerConn.CreateTopics(kafka.TopicConfig{
			Topic:             value,
			NumPartitions:     *partitions,
			ReplicationFactor: *replicas,
			ConfigEntries: []kafka.ConfigEntry{
				{ConfigName: "segment.bytes", ConfigValue: "2097152"},
			},
		})
	}
}
