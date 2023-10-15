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

type sendGroupMessageReqBody struct {
	GroupID string          `json:"group_id"`
	Message json.RawMessage `json:"message"`
}

type getGroupIDsRespBody struct {
	Code             int    `json:"code"`
	NextCursor       string `json:"next_cursor"`
	JoinedGroupChats struct {
		GroupIDs []string `json:"group_id"`
	} `json:"joined_group_chats"`
}
