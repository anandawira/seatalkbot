package seatalkbot

import "encoding/json"

type accessTokenReqBody struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

type sendPrivateMessageReqBody struct {
	EmployeeCode string          `json:"employee_code"`
	Message      json.RawMessage `json:"message"`
}
