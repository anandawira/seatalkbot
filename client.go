package seatalkbot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tidwall/gjson"

	"github.com/anandawira/seatalkbot/helper"
)

// defaultHost is the default host to use on API calls when not set in the config.
const defaultHost = "https://openapi.seatalk.io"

// Client is a Seatalkbot API caller. Client must initialize access token and update it with a new one before expired.
// You MUST call Close() on a client to avoid leaks, it will not be garbage-collected automatically when it passes out of scope.
// It is safe to share a client amongst many users.
type Client interface {
	// UpdateAccessToken gets new access token by using the credentials and store it in the client.
	UpdateAccessToken(ctx context.Context) error

	// AccessToken gets the underlying access token inside the client.
	// It might be used to implement your own API caller that's not yet supported by this library.
	AccessToken() string

	// Close stops the goroutines that auto refresh the access token. It is required to call
	// this function before the object passes out of scope, as it will otherwise leak memory.
	Close() error
}

type client struct {
	httpClient *http.Client
	host       string
	appID      string
	appSecret  string

	accessToken string
	stop        context.CancelFunc
}

type Config struct {
	// HTTPClient will be used for every HTTP calls made by the seatalkbot client.
	HTTPClient *http.Client
	// Host is the url of the bot api. It's https://openapi.seatalk.io by default.
	Host string
	// AppID of the seatalk bot. It can be found in the app setting at the seatalk dashboard.
	AppID string
	// AppSecret of the seatalk bot. It can be found in the app setting at the seatalk dashboard.
	AppSecret string
}

// NewClient returns a Client with the provided *http.Client and bot credentials. It will initialize access token using
// the credentials and automatically refresh the access token every 7000 seconds (expiration is 7200 seconds).
// It is required to call Close() before the object passes out of scope, as it will otherwise leak memory.
func NewClient(config Config) (Client, error) {
	if config.HTTPClient == nil {
		return nil, errors.New("http client should not be nil")
	}
	if config.Host == "" {
		config.Host = defaultHost
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &client{
		httpClient:  config.HTTPClient,
		host:        config.Host,
		appID:       config.AppID,
		appSecret:   config.AppSecret,
		accessToken: "",
		stop:        cancel,
	}

	err := helper.RunWithRetry(
		func() error {
			return c.UpdateAccessToken(ctx)
		},
		3,
		1*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("can't initialize access token, %w", err)
	}

	c.runAccessTokenScheduler(ctx)

	return c, nil
}

func (c *client) UpdateAccessToken(ctx context.Context) error {
	reqBody, err := json.Marshal(accessTokenReqBody{
		AppID:     c.appID,
		AppSecret: c.appSecret,
	})

	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/auth/app_access_token", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("status code not 200")
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	accessToken := gjson.Get(string(respBody), "app_access_token")
	if !accessToken.Exists() {
		return fmt.Errorf("access token not exist. resp_body = %s", respBody)
	}

	c.accessToken = accessToken.String()

	return nil
}

func (c *client) AccessToken() string {
	return c.accessToken
}

func (c *client) Close() error {
	if c.stop != nil {
		c.stop()
	}
	return nil
}

func (c *client) runAccessTokenScheduler(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(7000 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = helper.RunWithRetry(
					func() error {
						return c.UpdateAccessToken(ctx)
					},
					-1,
					10*time.Second,
				)
			}
		}
	}()
}
