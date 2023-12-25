package main

import (
	"context"
	"fmt"
	"time"
	"trip/internal/app"
	"trip/pkg/kafka"
)

func main() {
	ctx := context.Background()

	newApp := app.NewApp(ctx)

	fmt.Println(newApp)

	//topic_trip_to_client, err := kafka.ConnectKafka(ctx, "kafka:9092", "trip-driver-topic", 0)
	//if err != nil {
	//	fmt.Println(err)
	//}
	err := kafka.SendToTopic(newApp.FromClientDriverTopic, []byte("hellofrommain"))
	if err != nil {
		fmt.Println(err)
	}
	go func() {
		//for {
		//err := kafka.SendToTopic(newApp.FromClientDriverTopic, []byte("hello"))
		//if err != nil {
		//	fmt.Println(err)
		//}
		//}
	}()
	go func() {
		for {
			msg, err := kafka.ReadFromTopic(newApp.FromClientDriverTopic)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(string(msg))
		}
	}()
	time.Sleep(time.Minute * 5)
}
