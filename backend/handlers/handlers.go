package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/imadMansour85/Jetyard/backend/middleware"
	"github.com/imadMansour85/Jetyard/backend/models"
	"github.com/imadMansour85/Jetyard/backend/nats"
)

type Handler struct {
	natsClient *nats.Client
}

func NewHandler(nc *nats.Client) *Handler {
	return &Handler{
		natsClient: nc,
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes {"error": message}, plus the request ID set by
// middleware.RequestID so users can quote it when reporting a problem.
func writeError(w http.ResponseWriter, status int, message string) {
	body := map[string]string{"error": message}
	if id := w.Header().Get(middleware.RequestIDHeader); id != "" {
		body["request_id"] = id
	}
	writeJSON(w, status, body)
}

// paginate slices items according to optional limit/offset query params,
// defaulting to returning everything when they're absent — kept backward
// compatible with clients that don't send them. limit<=0 or a missing param
// means "no limit".
func paginate[T any](r *http.Request, items []T) []T {
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			offset = n
		}
	}
	if offset >= len(items) {
		return []T{}
	}
	items = items[offset:]

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < len(items) {
			items = items[:n]
		}
	}

	return items
}

// GetStats returns overall statistics
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.natsClient.GetStats()
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// GetServerInfo returns NATS server information
func (h *Handler) GetServerInfo(w http.ResponseWriter, r *http.Request) {
	serverInfo, err := h.natsClient.GetServerInfo()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get server info: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, serverInfo)
}

// ListConnections returns active NATS client connections and their subscriptions
func (h *Handler) ListConnections(w http.ResponseWriter, r *http.Request) {
	connections, err := h.natsClient.GetConnections()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get connections: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, connections)
}

// ListStreams returns all streams, optionally paginated via ?limit=&offset=
func (h *Handler) ListStreams(w http.ResponseWriter, r *http.Request) {
	streams, err := h.natsClient.ListStreams()
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, paginate(r, streams))
}

// GetStream returns a specific stream
func (h *Handler) GetStream(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "stream name is required")
		return
	}

	stream, err := h.natsClient.GetStream(name)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, stream)
}

// CreateStream creates a new stream
func (h *Handler) CreateStream(w http.ResponseWriter, r *http.Request) {
	var req models.CreateStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "stream name is required")
		return
	}

	if len(req.Subjects) == 0 {
		writeError(w, http.StatusBadRequest, "at least one subject is required")
		return
	}

	stream, err := h.natsClient.CreateStream(&req)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, stream)
}

// DeleteStream deletes a stream
func (h *Handler) DeleteStream(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "stream name is required")
		return
	}

	if err := h.natsClient.DeleteStream(name); err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "stream deleted"})
}

// ListConsumers returns all consumers for a stream, optionally paginated via
// ?limit=&offset=
func (h *Handler) ListConsumers(w http.ResponseWriter, r *http.Request) {
	streamName := r.URL.Query().Get("stream")
	if streamName == "" {
		writeError(w, http.StatusBadRequest, "stream name is required")
		return
	}

	consumers, err := h.natsClient.ListConsumers(streamName)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, paginate(r, consumers))
}

// CreateConsumer creates a new consumer
func (h *Handler) CreateConsumer(w http.ResponseWriter, r *http.Request) {
	var req models.CreateConsumerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.StreamName == "" {
		writeError(w, http.StatusBadRequest, "stream name is required")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "consumer name is required")
		return
	}

	consumer, err := h.natsClient.CreateConsumer(&req)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, consumer)
}

// DeleteConsumer deletes a consumer
func (h *Handler) DeleteConsumer(w http.ResponseWriter, r *http.Request) {
	streamName := r.URL.Query().Get("stream")
	consumerName := r.PathValue("name")

	if streamName == "" {
		writeError(w, http.StatusBadRequest, "stream name is required")
		return
	}

	if consumerName == "" {
		writeError(w, http.StatusBadRequest, "consumer name is required")
		return
	}

	if err := h.natsClient.DeleteConsumer(streamName, consumerName); err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "consumer deleted"})
}

// PublishMessage publishes a message
func (h *Handler) PublishMessage(w http.ResponseWriter, r *http.Request) {
	var req models.PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Subject == "" {
		writeError(w, http.StatusBadRequest, "subject is required")
		return
	}

	if err := h.natsClient.Publish(&req); err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "published"})
}

// GetMessages retrieves messages from a stream
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	streamName := r.PathValue("stream")
	if streamName == "" {
		writeError(w, http.StatusBadRequest, "stream name is required")
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	messages, err := h.natsClient.GetMessages(streamName, limit)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

// SSEHandler handles Server-Sent Events
func (h *Handler) SSEHandler(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this client
	eventChan := h.natsClient.AddEventListener()
	defer h.natsClient.RemoveEventListener(eventChan)

	// Send initial connection message
	json.NewEncoder(w).Encode(models.SSEEvent{
		Type: "connected",
		Data: map[string]string{"message": "Connected to Jetyard"},
	})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Listen for events
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return
			}

			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			// Write SSE formatted event
			w.Write([]byte("data: "))
			w.Write(data)
			w.Write([]byte("\n\n"))

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

		case <-r.Context().Done():
			return
		}
	}
}
