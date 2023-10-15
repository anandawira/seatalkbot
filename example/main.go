package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
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

	fmt.Println("Access token:", client.AccessToken())

	err = client.UpdateAccessToken(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println("Access token:", client.AccessToken())

	err = client.SendPrivateMessage(context.Background(), "150001", seatalkbot.TextMessage("test message", ""))
	if err != nil {
		panic(err)
	}

	fmt.Println("Message sent successfully")

	groupIDs, err := client.GetGroupIDs(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println("Group IDs:", groupIDs)

	for _, groupID := range groupIDs {
		messageID, err := client.SendGroupMessage(context.Background(), groupID, seatalkbot.TextMessage("test message", ""))
		if err != nil {
			panic(err)
		}
		fmt.Printf("Message sent successfully. group_id: %s, message_id: %s", groupID, messageID)
	}

	fmt.Println("Done")
}
