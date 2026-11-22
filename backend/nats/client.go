package nats

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/imadMansour85/Jetyard/backend/config"
	"github.com/imadMansour85/Jetyard/backend/models"
	"github.com/nats-io/nats.go"
)

type Client struct {
	conn           *nats.Conn
	js             nats.JetStreamContext
	eventListeners []chan models.SSEEvent
	mu             sync.RWMutex
}

// NewClientWithSettings creates a NATS client using full NATSSettings (auth + TLS).
func NewClientWithSettings(s *config.NATSSettings) (*Client, error) {
	opts := []nats.Option{}

	switch s.AuthType {
	case "userpass":
		opts = append(opts, nats.UserInfo(s.Username, s.Password))
	case "token":
		opts = append(opts, nats.Token(s.Token))
	}

	if s.TLSEnabled {
		tlsCfg := &tls.Config{}

		if s.CACert != "" {
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM([]byte(s.CACert))
			tlsCfg.RootCAs = pool
		}

		if s.ClientCert != "" && s.ClientKey != "" {
			cert, err := tls.X509KeyPair([]byte(s.ClientCert), []byte(s.ClientKey))
			if err != nil {
				return nil, fmt.Errorf("invalid TLS client certificate: %w", err)
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}

		opts = append(opts, nats.Secure(tlsCfg))
	}

	nc, err := nats.Connect(s.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	log.Printf("Connected to NATS at %s (auth: %s, tls: %v)", s.URL, s.AuthType, s.TLSEnabled)

	return &Client{
		conn:           nc,
		js:             js,
		eventListeners: make([]chan models.SSEEvent, 0),
	}, nil
}

func NewClient(url string) (*Client, error) {
	return NewClientWithSettings(&config.NATSSettings{URL: url, AuthType: "none"})
}

func (c *Client) Close() {
	c.mu.Lock()
	for _, ch := range c.eventListeners {
		close(ch)
	}
	c.eventListeners = nil
	c.mu.Unlock()

	c.conn.Close()
}

// AddEventListener adds a new SSE event listener
func (c *Client) AddEventListener() chan models.SSEEvent {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan models.SSEEvent, 100)
	c.eventListeners = append(c.eventListeners, ch)
	return ch
}

// RemoveEventListener removes an SSE event listener
func (c *Client) RemoveEventListener(ch chan models.SSEEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, listener := range c.eventListeners {
		if listener == ch {
			close(ch)
			c.eventListeners = append(c.eventListeners[:i], c.eventListeners[i+1:]...)
			break
		}
	}
}

// BroadcastEvent sends an event to all SSE listeners
func (c *Client) BroadcastEvent(eventType string, data interface{}) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	event := models.SSEEvent{
		Type: eventType,
		Data: data,
	}

	for _, ch := range c.eventListeners {
		select {
		case ch <- event:
		default:
			log.Printf("Warning: event listener channel full, dropping event")
		}
	}
}

// ListStreams returns all JetStream streams
func (c *Client) ListStreams() ([]*models.Stream, error) {
	var streams []*models.Stream

	for stream := range c.js.Streams() {
		info, err := c.js.StreamInfo(stream.Config.Name)
		if err != nil {
			log.Printf("Error getting stream info for %s: %v", stream.Config.Name, err)
			continue
		}

		streams = append(streams, &models.Stream{
			Name:      info.Config.Name,
			Subjects:  info.Config.Subjects,
			Retention: info.Config.Retention.String(),
			MaxMsgs:   info.Config.MaxMsgs,
			MaxBytes:  info.Config.MaxBytes,
			MaxAge:    int64(info.Config.MaxAge),
			Storage:   info.Config.Storage.String(),
			Replicas:  info.Config.Replicas,
			Messages:  info.State.Msgs,
			Bytes:     info.State.Bytes,
			FirstSeq:  info.State.FirstSeq,
			LastSeq:   info.State.LastSeq,
		})
	}

	return streams, nil
}

// GetStream returns information about a specific stream
func (c *Client) GetStream(name string) (*models.Stream, error) {
	info, err := c.js.StreamInfo(name)
	if err != nil {
		return nil, fmt.Errorf("stream not found: %w", err)
	}

	return &models.Stream{
		Name:      info.Config.Name,
		Subjects:  info.Config.Subjects,
		Retention: info.Config.Retention.String(),
		MaxMsgs:   info.Config.MaxMsgs,
		MaxBytes:  info.Config.MaxBytes,
		MaxAge:    int64(info.Config.MaxAge),
		Storage:   info.Config.Storage.String(),
		Replicas:  info.Config.Replicas,
		Messages:  info.State.Msgs,
		Bytes:     info.State.Bytes,
		FirstSeq:  info.State.FirstSeq,
		LastSeq:   info.State.LastSeq,
	}, nil
}

// CreateStream creates a new JetStream stream
func (c *Client) CreateStream(req *models.CreateStreamRequest) (*models.Stream, error) {
	cfg := &nats.StreamConfig{
		Name:     req.Name,
		Subjects: req.Subjects,
	}

	if req.Retention != "" {
		switch req.Retention {
		case "limits":
			cfg.Retention = nats.LimitsPolicy
		case "interest":
			cfg.Retention = nats.InterestPolicy
		case "workqueue":
			cfg.Retention = nats.WorkQueuePolicy
		}
	}

	if req.MaxMsgs > 0 {
		cfg.MaxMsgs = req.MaxMsgs
	}
	if req.MaxBytes > 0 {
		cfg.MaxBytes = req.MaxBytes
	}
	if req.MaxAge > 0 {
		cfg.MaxAge = time.Duration(req.MaxAge)
	}

	if req.Storage != "" {
		if req.Storage == "file" {
			cfg.Storage = nats.FileStorage
		} else {
			cfg.Storage = nats.MemoryStorage
		}
	}

	if req.Replicas > 0 {
		cfg.Replicas = req.Replicas
	}

	info, err := c.js.AddStream(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	stream := &models.Stream{
		Name:      info.Config.Name,
		Subjects:  info.Config.Subjects,
		Retention: info.Config.Retention.String(),
		MaxMsgs:   info.Config.MaxMsgs,
		MaxBytes:  info.Config.MaxBytes,
		MaxAge:    int64(info.Config.MaxAge),
		Storage:   info.Config.Storage.String(),
		Replicas:  info.Config.Replicas,
		Messages:  info.State.Msgs,
		Bytes:     info.State.Bytes,
		FirstSeq:  info.State.FirstSeq,
		LastSeq:   info.State.LastSeq,
	}

	c.BroadcastEvent("stream_created", stream)

	return stream, nil
}

// DeleteStream deletes a JetStream stream
func (c *Client) DeleteStream(name string) error {
	err := c.js.DeleteStream(name)
	if err != nil {
		return fmt.Errorf("failed to delete stream: %w", err)
	}

	c.BroadcastEvent("stream_deleted", map[string]string{"name": name})

	return nil
}

// ListConsumers returns all consumers for a stream
func (c *Client) ListConsumers(streamName string) ([]*models.Consumer, error) {
	var consumers []*models.Consumer

	// js.Consumers swallows errors and just closes the channel, so a missing
	// stream would look like a stream with no consumers.
	if _, err := c.js.StreamInfo(streamName); err != nil {
		return nil, fmt.Errorf("stream not found: %w", err)
	}

	for consumer := range c.js.Consumers(streamName) {
		info, err := c.js.ConsumerInfo(streamName, consumer.Name)
		if err != nil {
			log.Printf("Error getting consumer info: %v", err)
			continue
		}

		consumers = append(consumers, &models.Consumer{
			StreamName:     streamName,
			Name:           info.Name,
			DeliverPolicy:  deliverPolicyToString(info.Config.DeliverPolicy),
			AckPolicy:      ackPolicyToString(info.Config.AckPolicy),
			AckWait:        int64(info.Config.AckWait),
			MaxDeliver:     info.Config.MaxDeliver,
			FilterSubject:  info.Config.FilterSubject,
			ReplayPolicy:   replayPolicyToString(info.Config.ReplayPolicy),
			NumPending:     info.NumPending,
			NumAckPending:  uint64(info.NumAckPending),
			NumRedelivered: uint64(info.NumRedelivered),
			Delivered:      info.Delivered.Consumer,
		})
	}

	return consumers, nil
}

// CreateConsumer creates a new consumer
func (c *Client) CreateConsumer(req *models.CreateConsumerRequest) (*models.Consumer, error) {
	cfg := &nats.ConsumerConfig{
		Durable:       req.Name,
		FilterSubject: req.FilterSubject,
	}

	if req.DeliverPolicy != "" {
		switch req.DeliverPolicy {
		case "all":
			cfg.DeliverPolicy = nats.DeliverAllPolicy
		case "last":
			cfg.DeliverPolicy = nats.DeliverLastPolicy
		case "new":
			cfg.DeliverPolicy = nats.DeliverNewPolicy
		}
	}

	if req.AckPolicy != "" {
		switch req.AckPolicy {
		case "explicit":
			cfg.AckPolicy = nats.AckExplicitPolicy
		case "none":
			cfg.AckPolicy = nats.AckNonePolicy
		case "all":
			cfg.AckPolicy = nats.AckAllPolicy
		}
	}

	if req.AckWait > 0 {
		cfg.AckWait = time.Duration(req.AckWait)
	}

	if req.MaxDeliver > 0 {
		cfg.MaxDeliver = req.MaxDeliver
	}

	if req.ReplayPolicy != "" {
		if req.ReplayPolicy == "instant" {
			cfg.ReplayPolicy = nats.ReplayInstantPolicy
		} else {
			cfg.ReplayPolicy = nats.ReplayOriginalPolicy
		}
	}

	info, err := c.js.AddConsumer(req.StreamName, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	consumer := &models.Consumer{
		StreamName:     req.StreamName,
		Name:           info.Name,
		DeliverPolicy:  deliverPolicyToString(info.Config.DeliverPolicy),
		AckPolicy:      ackPolicyToString(info.Config.AckPolicy),
		AckWait:        int64(info.Config.AckWait),
		MaxDeliver:     info.Config.MaxDeliver,
		FilterSubject:  info.Config.FilterSubject,
		ReplayPolicy:   replayPolicyToString(info.Config.ReplayPolicy),
		NumPending:     info.NumPending,
		NumAckPending:  uint64(info.NumAckPending),
		NumRedelivered: uint64(info.NumRedelivered),
		Delivered:      info.Delivered.Consumer,
	}

	c.BroadcastEvent("consumer_created", consumer)

	return consumer, nil
}

// DeleteConsumer deletes a consumer
func (c *Client) DeleteConsumer(streamName, consumerName string) error {
	err := c.js.DeleteConsumer(streamName, consumerName)
	if err != nil {
		return fmt.Errorf("failed to delete consumer: %w", err)
	}

	c.BroadcastEvent("consumer_deleted", map[string]string{
		"stream_name": streamName,
		"name":        consumerName,
	})

	return nil
}

// Publish publishes a message
func (c *Client) Publish(req *models.PublishRequest) error {
	msg := &nats.Msg{
		Subject: req.Subject,
		Data:    []byte(req.Data),
	}

	if len(req.Headers) > 0 {
		msg.Header = make(nats.Header)
		for k, v := range req.Headers {
			msg.Header.Set(k, v)
		}
	}

	_, err := c.js.PublishMsg(msg)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	c.BroadcastEvent("message_published", map[string]string{
		"subject": req.Subject,
	})

	return nil
}

// GetMessages retrieves messages from a stream
func (c *Client) GetMessages(streamName string, limit int) ([]*models.StreamMessage, error) {
	info, err := c.js.StreamInfo(streamName)
	if err != nil {
		return nil, fmt.Errorf("stream not found: %w", err)
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var messages []*models.StreamMessage
	startSeq := info.State.LastSeq
	if startSeq > uint64(limit) {
		startSeq = info.State.LastSeq - uint64(limit) + 1
	} else {
		startSeq = info.State.FirstSeq
	}

	for i := startSeq; i <= info.State.LastSeq; i++ {
		msg, err := c.js.GetMsg(streamName, i)
		if err != nil {
			continue
		}

		headers := make(map[string]string)
		if msg.Header != nil {
			for k := range msg.Header {
				headers[k] = msg.Header.Get(k)
			}
		}

		messages = append(messages, &models.StreamMessage{
			Subject:  msg.Subject,
			Sequence: msg.Sequence,
			Time:     msg.Time,
			Data:     string(msg.Data),
			Headers:  headers,
		})
	}

	return messages, nil
}

// GetStats returns overall statistics
func (c *Client) GetStats() (map[string]interface{}, error) {
	streams, err := c.ListStreams()
	if err != nil {
		return nil, err
	}

	totalMessages := uint64(0)
	totalBytes := uint64(0)
	totalConsumers := 0

	for _, stream := range streams {
		totalMessages += stream.Messages
		totalBytes += stream.Bytes

		consumers, err := c.ListConsumers(stream.Name)
		if err == nil {
			totalConsumers += len(consumers)
		}
	}

	return map[string]interface{}{
		"total_streams":   len(streams),
		"total_consumers": totalConsumers,
		"total_messages":  totalMessages,
		"total_bytes":     totalBytes,
		"connected":       c.conn.IsConnected(),
		"server_info":     c.conn.ConnectedServerName(),
	}, nil
}

// Helper functions to convert NATS enums to strings
func deliverPolicyToString(policy nats.DeliverPolicy) string {
	switch policy {
	case nats.DeliverAllPolicy:
		return "DeliverAll"
	case nats.DeliverLastPolicy:
		return "DeliverLast"
	case nats.DeliverNewPolicy:
		return "DeliverNew"
	case nats.DeliverByStartSequencePolicy:
		return "DeliverByStartSequence"
	case nats.DeliverByStartTimePolicy:
		return "DeliverByStartTime"
	case nats.DeliverLastPerSubjectPolicy:
		return "DeliverLastPerSubject"
	default:
		return "Unknown"
	}
}

func ackPolicyToString(policy nats.AckPolicy) string {
	switch policy {
	case nats.AckNonePolicy:
		return "AckNone"
	case nats.AckAllPolicy:
		return "AckAll"
	case nats.AckExplicitPolicy:
		return "AckExplicit"
	default:
		return "Unknown"
	}
}

func replayPolicyToString(policy nats.ReplayPolicy) string {
	switch policy {
	case nats.ReplayInstantPolicy:
		return "ReplayInstant"
	case nats.ReplayOriginalPolicy:
		return "ReplayOriginal"
	default:
		return "Unknown"
	}
}
