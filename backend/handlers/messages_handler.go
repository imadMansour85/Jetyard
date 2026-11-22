package handlers

import (
	"net/http"
	"strconv"
)

// ListStreamMessages returns messages from a stream
func (h *Handler) ListStreamMessages(w http.ResponseWriter, r *http.Request) {
	streamName := r.PathValue("stream")
	if streamName == "" {
		writeError(w, http.StatusBadRequest, "stream name is required")
		return
	}

	// Parse query parameters
	subject := r.URL.Query().Get("subject")
	startSeqStr := r.URL.Query().Get("start")
	limitStr := r.URL.Query().Get("limit")

	var startSeq uint64
	var limit int = 50

	if startSeqStr != "" {
		seq, err := strconv.ParseUint(startSeqStr, 10, 64)
		if err == nil {
			startSeq = seq
		}
	}

	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := h.natsClient.ListMessages(streamName, subject, startSeq, limit)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

// GetMessageBySeq returns a specific message by sequence
func (h *Handler) GetMessageBySeq(w http.ResponseWriter, r *http.Request) {
	streamName := r.PathValue("stream")
	seqStr := r.PathValue("seq")

	if streamName == "" || seqStr == "" {
		writeError(w, http.StatusBadRequest, "stream name and sequence are required")
		return
	}

	seq, err := strconv.ParseUint(seqStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid sequence number")
		return
	}

	message, err := h.natsClient.GetMessageBySeq(streamName, seq)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, message)
}
