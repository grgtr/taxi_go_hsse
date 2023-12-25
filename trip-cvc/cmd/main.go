package main

import (
	"context"
	"time"
	"trip/internal/app"
)

func main() {
	ctx := context.Background()

	newApp := app.NewApp(ctx)

	newApp.Start(ctx)
	time.Sleep(time.Minute * 5)
}
