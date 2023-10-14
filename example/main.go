package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anandawira/seatalkbot"
)

func main() {
	client, err := seatalkbot.NewClient(seatalkbot.Config{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Host:       "https://openapi.seatalk.io",
		AppID:      os.Getenv("APP_ID"),
		AppSecret:  os.Getenv("APP_SECRET"),
	})

	if err != nil {
		panic(err)
	}

	defer client.Close()

	fmt.Println(client.AccessToken())

	err = client.UpdateAccessToken(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println(client.AccessToken())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, os.Interrupt)

	<-quit
}