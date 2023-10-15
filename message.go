package seatalkbot

import "encoding/json"

// Message is used as a parameter for sending message
type Message interface {
	Message() json.RawMessage
}

var _ Message = TextMessage("")

// TextMessage wraps string and implements Message
type TextMessage string

func (t TextMessage) Message() json.RawMessage {
	b, err := json.Marshal(textMessage{
		Tag: "text",
		Text: struct {
			Content string `json:"content"`
		}{Content: string(t)},
	})

	if err != nil {
		panic(err)
	}

	return b
}

type textMessage struct {
	Tag  string `json:"tag"`
	Text struct {
		Content string `json:"content"`
	} `json:"text"`
}
