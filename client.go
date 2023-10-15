package seatalkbot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/tidwall/gjson"

	"github.com/anandawira/seatalkbot/helper"
)

const (
	// defaultHost is the default host to use on API calls when not set in the config.
	defaultHost = "https://openapi.seatalk.io"
	// pageSize is the page size for each API call that uses pagination
	pageSize = 50
)

// Client is a Seatalkbot API caller. Client must initialize access token and update it with a new one before expired.
// You MUST call Close() on a client to avoid leaks, it will not be garbage-collected automatically when it passes out of scope.
// It is safe to share a client amongst many users.
type Client interface {
	// SendPrivateMessage send a private message to a user by employeeCode.
	SendPrivateMessage(ctx context.Context, employeeCode string, message Message) error

	// GetGroupIDs get list of group ids joined by the bot.
	GetGroupIDs(ctx context.Context) ([]string, error)
	// SendGroupMessage send a message to a group by groupID.
	SendGroupMessage(ctx context.Context, groupID string, message Message) (messageID string, err error)

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

// SendPrivateMessage implements Client
func (c *client) SendPrivateMessage(ctx context.Context, employeeCode string, message Message) error {
	reqBody, err := json.Marshal(sendPrivateMessageReqBody{
		EmployeeCode: employeeCode,
		Message:      message.Message(),
	})

	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/messaging/v2/single_chat", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http response code not 200, got: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if code := gjson.Get(string(respBody), "code"); !code.Exists() || code.Int() != 0 {
		return fmt.Errorf("code in response body is not exist or not 0, resp_body: %s", respBody)
	}

	return nil
}

// GetGroupIDs implements Client
func (c *client) GetGroupIDs(ctx context.Context) ([]string, error) {
	var groupIDs []string
	var cursor string

	for {
		ids, nextCursor, err := c.getGroupIDs(ctx, cursor)
		if err != nil {
			return nil, err
		}

		groupIDs = append(groupIDs, ids...)

		if nextCursor == "" {
			break
		}
	}

	return groupIDs, nil
}

// SendGroupMessage implements Client
func (c *client) SendGroupMessage(ctx context.Context, groupID string, message Message) (messageID string, err error) {
	reqBody, err := json.Marshal(sendGroupMessageReqBody{
		GroupID: groupID,
		Message: message.Message(),
	})

	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/messaging/v2/group_chat", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http response code not 200, got: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if code := gjson.Get(string(respBody), "code"); !code.Exists() || code.Int() != 0 {
		return "", fmt.Errorf("code in response body is not exist or not 0, resp_body: %s", respBody)
	}

	return gjson.Get(string(respBody), "message_id").String(), nil
}

// UpdateAccessToken implements Client
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
		return fmt.Errorf("status code is not 200, got: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	accessToken := gjson.Get(string(respBody), "app_access_token")
	if !accessToken.Exists() {
		return fmt.Errorf("access token not exist. resp_body: %s", respBody)
	}

	c.accessToken = accessToken.String()

	return nil
}

// AccessToken implements Client
func (c *client) AccessToken() string {
	return c.accessToken
}

// Close implements Client
func (c *client) Close() error {
	if c.stop != nil {
		c.stop()
	}
	return nil
}

func (c *client) getGroupIDs(ctx context.Context, cursor string) (groupIDs []string, nextCursor string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+"/messaging/v2/group_chat/joined", http.NoBody)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	q := url.Values{}
	q.Set("page_size", strconv.Itoa(pageSize))
	if cursor != "" {
		q.Set("cursor", cursor)
	}

	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("status code not 200, got: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	response := getGroupIDsRespBody{
		Code: -1, // To know if the code is not found in the response body.
	}
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return nil, "", err
	}

	if response.Code != 0 {
		return nil, "", fmt.Errorf("error code is not 0, got: %d", response.Code)
	}

	return response.JoinedGroupChats.GroupIDs, response.NextCursor, nil
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
