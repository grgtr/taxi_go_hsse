// cmd/main_client.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"taxi/internal/app"
	"taxi/internal/config"
)

func main() {
	var filePath string
	flag.StringVar(&filePath, "path", "./cmd/.config.json", "set config path")
	flag.Parse()

	cfg, err := config.Parse(filePath)
	if err != nil {
		fmt.Println(err)
		log.Fatal()
	}
	fmt.Println(cfg)
	clientApp := app.NewApp(cfg)

	// Run the client service in a goroutine
	go clientApp.Start()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Shutdown the client service gracefully
	log.Println("Shutting down the client service...")
	clientApp.Database.Close()
	log.Println("Client service has stopped.")
}
