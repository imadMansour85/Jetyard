package models

import "time"

// KVBucket represents a NATS KV bucket
type KVBucket struct {
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	MaxValueSize int64     `json:"max_value_size"`
	History      uint8     `json:"history"`
	TTL          int64     `json:"ttl"` // nanoseconds
	Bucket       string    `json:"bucket"`
	Values       uint64    `json:"values"`
	Bytes        uint64    `json:"bytes"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateKVBucketRequest is the request to create a KV bucket
type CreateKVBucketRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	History      uint8  `json:"history,omitempty"`        // default: 1
	TTL          int64  `json:"ttl,omitempty"`            // nanoseconds, 0 = no expiry
	MaxValueSize int32  `json:"max_value_size,omitempty"` // bytes
}

// KVEntry represents a key-value entry
type KVEntry struct {
	Bucket    string    `json:"bucket"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Revision  uint64    `json:"revision"`
	Created   time.Time `json:"created"`
	Operation string    `json:"operation"`
}

// KVPutRequest is the request to put a key-value
type KVPutRequest struct {
	Value string `json:"value"`
}
