package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/imadMansour85/Jetyard/backend/models"
	"github.com/imadMansour85/Jetyard/backend/nats"
)

type KVHandler struct {
	natsClient *nats.Client
}

func NewKVHandler(nc *nats.Client) *KVHandler {
	return &KVHandler{
		natsClient: nc,
	}
}

// ListBuckets returns all KV buckets
func (h *KVHandler) ListBuckets(w http.ResponseWriter, r *http.Request) {
	buckets, err := h.natsClient.ListKVBuckets()
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buckets)
}

// GetBucket returns a specific KV bucket
func (h *KVHandler) GetBucket(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "bucket name is required")
		return
	}

	bucket, err := h.natsClient.GetKVBucket(name)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, bucket)
}

// CreateBucket creates a new KV bucket
func (h *KVHandler) CreateBucket(w http.ResponseWriter, r *http.Request) {
	var req models.CreateKVBucketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "bucket name is required")
		return
	}

	bucket, err := h.natsClient.CreateKVBucket(&req)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, bucket)
}

// DeleteBucket deletes a KV bucket
func (h *KVHandler) DeleteBucket(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "bucket name is required")
		return
	}

	if err := h.natsClient.DeleteKVBucket(name); err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "bucket deleted"})
}

// ListKeys returns all keys in a bucket, optionally paginated via
// ?limit=&offset=
func (h *KVHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	if bucket == "" {
		writeError(w, http.StatusBadRequest, "bucket name is required")
		return
	}

	keys, err := h.natsClient.KVKeys(bucket)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, paginate(r, keys))
}

// GetKey returns a value by key
func (h *KVHandler) GetKey(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	if bucket == "" || key == "" {
		writeError(w, http.StatusBadRequest, "bucket and key are required")
		return
	}

	entry, err := h.natsClient.KVGet(bucket, key)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

// PutKey puts a key-value pair
func (h *KVHandler) PutKey(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	if bucket == "" || key == "" {
		writeError(w, http.StatusBadRequest, "bucket and key are required")
		return
	}

	var req models.KVPutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	revision, err := h.natsClient.KVPut(bucket, key, req.Value)
	if err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "key updated",
		"revision": revision,
	})
}

// DeleteKey deletes a key
func (h *KVHandler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	if bucket == "" || key == "" {
		writeError(w, http.StatusBadRequest, "bucket and key are required")
		return
	}

	if err := h.natsClient.KVDelete(bucket, key); err != nil {
		writeNATSError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "key deleted"})
}
