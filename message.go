package seatalkbot

import "encoding/json"

// Message is used as a parameter for sending message
type Message interface {
	Message() json.RawMessage
}

func TextMessage(content, quotedMessageID string) Message {
	return textMessage{
		Tag: "text",
		Text: struct {
			Content string `json:"content"`
		}{Content: content},
		QuotedMessageID: quotedMessageID,
	}
}

type textMessage struct {
	Tag  string `json:"tag"`
	Text struct {
		Content string `json:"content"`
	} `json:"text"`
	QuotedMessageID string `json:"quoted_message_id,omitempty"`
}

func (t textMessage) Message() json.RawMessage {
	b, err := json.Marshal(t)

	if err != nil {
		panic(err)
	}

	return b
}
