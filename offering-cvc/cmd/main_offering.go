package main

import (
	"context"
	"log"
	"taxi/internal/app"
)

func main() {
	// Создание контекста
	ctx := context.Background()

	// Создание управляющего приложения
	newApp := app.NewApp()
	err := newApp.Start(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}
}
