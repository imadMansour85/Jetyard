package models

import "time"

// Stream represents a NATS JetStream stream
type Stream struct {
	Name      string                 `json:"name"`
	Subjects  []string               `json:"subjects"`
	Retention string                 `json:"retention"`
	MaxMsgs   int64                  `json:"max_msgs"`
	MaxBytes  int64                  `json:"max_bytes"`
	MaxAge    int64                  `json:"max_age"`
	Storage   string                 `json:"storage"`
	Replicas  int                    `json:"replicas"`
	Messages  uint64                 `json:"messages"`
	Bytes     uint64                 `json:"bytes"`
	FirstSeq  uint64                 `json:"first_seq"`
	LastSeq   uint64                 `json:"last_seq"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

// Consumer represents a NATS JetStream consumer
type Consumer struct {
	StreamName     string `json:"stream_name"`
	Name           string `json:"name"`
	DeliverPolicy  string `json:"deliver_policy"`
	AckPolicy      string `json:"ack_policy"`
	AckWait        int64  `json:"ack_wait"`
	MaxDeliver     int    `json:"max_deliver"`
	FilterSubject  string `json:"filter_subject,omitempty"`
	ReplayPolicy   string `json:"replay_policy"`
	NumPending     uint64 `json:"num_pending"`
	NumAckPending  uint64 `json:"num_ack_pending"`
	NumRedelivered uint64 `json:"num_redelivered"`
	Delivered      uint64 `json:"delivered"`
}

// Message represents a message to publish
type PublishRequest struct {
	Subject string            `json:"subject"`
	Data    string            `json:"data"`
	Headers map[string]string `json:"headers,omitempty"`
}

// StreamMessage represents a message in a stream
type StreamMessage struct {
	Subject  string            `json:"subject"`
	Sequence uint64            `json:"sequence"`
	Time     time.Time         `json:"time"`
	Data     string            `json:"data"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// CreateStreamRequest request to create a stream
type CreateStreamRequest struct {
	Name      string   `json:"name"`
	Subjects  []string `json:"subjects"`
	Retention string   `json:"retention,omitempty"`
	MaxMsgs   int64    `json:"max_msgs,omitempty"`
	MaxBytes  int64    `json:"max_bytes,omitempty"`
	MaxAge    int64    `json:"max_age,omitempty"`
	Storage   string   `json:"storage,omitempty"`
	Replicas  int      `json:"replicas,omitempty"`
}

// CreateConsumerRequest request to create a consumer
type CreateConsumerRequest struct {
	StreamName    string `json:"stream_name"`
	Name          string `json:"name"`
	DeliverPolicy string `json:"deliver_policy,omitempty"`
	AckPolicy     string `json:"ack_policy,omitempty"`
	AckWait       int64  `json:"ack_wait,omitempty"`
	MaxDeliver    int    `json:"max_deliver,omitempty"`
	FilterSubject string `json:"filter_subject,omitempty"`
	ReplayPolicy  string `json:"replay_policy,omitempty"`
}

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
