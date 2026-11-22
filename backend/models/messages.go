package models

import "time"

// MessageDetail represents a message in a stream
type MessageDetail struct {
	Subject  string              `json:"subject"`
	Sequence uint64              `json:"sequence"`
	Time     time.Time           `json:"time"`
	Data     string              `json:"data"`
	Headers  map[string][]string `json:"headers,omitempty"`
	Size     int                 `json:"size"`
}

// MessageListRequest is the request to list messages
type MessageListRequest struct {
	Stream      string `json:"stream"`
	Subject     string `json:"subject,omitempty"`      // optional filter
	StartSeq    uint64 `json:"start_seq,omitempty"`    // optional
	MaxMessages int    `json:"max_messages,omitempty"` // pagination
}
