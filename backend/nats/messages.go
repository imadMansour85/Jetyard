package nats

import (
	"fmt"

	"github.com/imadMansour85/Jetyard/backend/models"
)

// ListMessages retrieves messages from a stream with optional filters
func (c *Client) ListMessages(streamName string, subject string, startSeq uint64, limit int) ([]*models.MessageDetail, error) {
	// Get stream info First
	info, err := c.js.StreamInfo(streamName)
	if err != nil {
		return nil, fmt.Errorf("stream not found: %w", err)
	}

	if limit <= 0 || limit > 100 {
		limit = 50 // default
	}

	// If no start sequence, start from the most recent
	if startSeq == 0 {
		if info.State.Msgs == 0 {
			return []*models.MessageDetail{}, nil
		}
		// Start from last minus limit
		if info.State.LastSeq > uint64(limit) {
			startSeq = info.State.LastSeq - uint64(limit) + 1
		} else {
			startSeq = info.State.FirstSeq
		}
	}

	var messages []*models.MessageDetail
	count := 0
	scanned := 0

	// Cap how many sequences we'll walk regardless of limit — a subject
	// filter that matches rarely would otherwise force a scan of the entire
	// stream (one synchronous NATS call per sequence) to fill a small limit.
	const maxScan = 5000

	// Fetch messages using GetMsg
	for seq := startSeq; seq <= info.State.LastSeq && count < limit && scanned < maxScan; seq++ {
		scanned++

		msg, err := c.js.GetMsg(streamName, seq)
		if err != nil {
			// Message might be deleted, skip
			continue
		}

		// Filter by subject if provided
		if subject != "" && msg.Subject != subject {
			continue
		}

		headers := make(map[string][]string)
		if msg.Header != nil {
			for k, v := range msg.Header {
				headers[k] = v
			}
		}

		messages = append(messages, &models.MessageDetail{
			Subject:  msg.Subject,
			Sequence: msg.Sequence,
			Time:     msg.Time,
			Data:     string(msg.Data),
			Headers:  headers,
			Size:     len(msg.Data),
		})

		count++
	}

	return messages, nil
}

// GetMessageBySeq retrieves a specific message by sequence number
func (c *Client) GetMessageBySeq(streamName string, seq uint64) (*models.MessageDetail, error) {
	msg, err := c.js.GetMsg(streamName, seq)
	if err != nil {
		return nil, fmt.Errorf("message not found: %w", err)
	}

	headers := make(map[string][]string)
	if msg.Header != nil {
		for k, v := range msg.Header {
			headers[k] = v
		}
	}

	return &models.MessageDetail{
		Subject:  msg.Subject,
		Sequence: msg.Sequence,
		Time:     msg.Time,
		Data:     string(msg.Data),
		Headers:  headers,
		Size:     len(msg.Data),
	}, nil
}
