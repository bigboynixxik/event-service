package main

import (
	"context"
	"log"

	"eventify-events/internal/app"
)

func main() {
	ctx := context.Background()
	a, err := app.NewApp(ctx)
	if err != nil {
		panic(err)
	}
	if err := a.Run(); err != nil {
		log.Fatalf("main.main, failed to run app %v", err)
	}
}
