package nats

import (
	"errors"
	"fmt"
	"time"

	"github.com/imadMansour85/Jetyard/backend/models"
	"github.com/nats-io/nats.go"
)

// ListKVBuckets returns all KV buckets
func (c *Client) ListKVBuckets() ([]*models.KVBucket, error) {
	var buckets []*models.KVBucket

	// KV buckets are implemented as streams with "KV_" prefix
	// Use streams API to find them
	streamsInfo := c.js.StreamsInfo()
	for stream := range streamsInfo {
		if stream == nil {
			continue
		}

		// KV buckets have stream names starting with "KV_"
		streamName := stream.Config.Name
		if len(streamName) > 3 && streamName[:3] == "KV_" {
			bucketName := streamName[3:] // Remove "KV_" prefix

			// Get the actual KV bucket
			kv, err := c.js.KeyValue(bucketName)
			if err != nil {
				continue
			}

			status, err := kv.Status()
			if err != nil {
				continue
			}

			buckets = append(buckets, &models.KVBucket{
				Name:         status.Bucket(),
				History:      uint8(status.History()),
				TTL:          int64(status.TTL()),
				Bucket:       status.Bucket(),
				Values:       status.Values(),
				Bytes:        status.Bytes(),
				MaxValueSize: int64(stream.Config.MaxMsgSize),
			})
		}
	}

	return buckets, nil
}

// GetKVBucket returns information about a specific KV bucket
func (c *Client) GetKVBucket(name string) (*models.KVBucket, error) {
	kv, err := c.js.KeyValue(name)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	status, err := kv.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket status: %w", err)
	}

	return &models.KVBucket{
		Name:         status.Bucket(),
		History:      uint8(status.History()),
		TTL:          int64(status.TTL()),
		Bucket:       status.Bucket(),
		Values:       status.Values(),
		Bytes:        status.Bytes(),
		MaxValueSize: int64(status.(*nats.KeyValueBucketStatus).StreamInfo().Config.MaxMsgSize),
	}, nil
}

// CreateKVBucket creates a new KV bucket
func (c *Client) CreateKVBucket(req *models.CreateKVBucketRequest) (*models.KVBucket, error) {
	cfg := &nats.KeyValueConfig{
		Bucket: req.Name,
	}

	if req.Description != "" {
		cfg.Description = req.Description
	}
	if req.History > 0 {
		cfg.History = uint8(req.History)
	} else {
		cfg.History = 1
	}
	if req.TTL > 0 {
		cfg.TTL = time.Duration(req.TTL)
	}
	if req.MaxValueSize > 0 {
		cfg.MaxValueSize = req.MaxValueSize
	}

	kv, err := c.js.CreateKeyValue(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create KV bucket: %w", err)
	}

	status, err := kv.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket status: %w", err)
	}

	bucket := &models.KVBucket{
		Name:         status.Bucket(),
		Description:  req.Description,
		History:      uint8(status.History()),
		TTL:          int64(status.TTL()),
		Bucket:       status.Bucket(),
		Values:       status.Values(),
		Bytes:        status.Bytes(),
		MaxValueSize: int64(req.MaxValueSize),
		CreatedAt:    time.Now(),
	}

	c.BroadcastEvent("kv_bucket_created", bucket)

	return bucket, nil
}

// DeleteKVBucket deletes a KV bucket
func (c *Client) DeleteKVBucket(name string) error {
	err := c.js.DeleteKeyValue(name)
	if err != nil {
		return fmt.Errorf("failed to delete KV bucket: %w", err)
	}

	c.BroadcastEvent("kv_bucket_deleted", map[string]string{"name": name})

	return nil
}

// KVPut puts a key-value pair
func (c *Client) KVPut(bucket, key, value string) (uint64, error) {
	kv, err := c.js.KeyValue(bucket)
	if err != nil {
		return 0, fmt.Errorf("bucket not found: %w", err)
	}

	revision, err := kv.PutString(key, value)
	if err != nil {
		return 0, fmt.Errorf("failed to put value: %w", err)
	}

	c.BroadcastEvent("kv_key_put", map[string]interface{}{
		"bucket":   bucket,
		"key":      key,
		"revision": revision,
	})

	return revision, nil
}

// KVGet gets a value by key
func (c *Client) KVGet(bucket, key string) (*models.KVEntry, error) {
	kv, err := c.js.KeyValue(bucket)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	entry, err := kv.Get(key)
	if err != nil {
		return nil, fmt.Errorf("key not found: %w", err)
	}

	return &models.KVEntry{
		Bucket:    bucket,
		Key:       entry.Key(),
		Value:     string(entry.Value()),
		Revision:  entry.Revision(),
		Created:   entry.Created(),
		Operation: entry.Operation().String(),
	}, nil
}

// KVDelete deletes a key
func (c *Client) KVDelete(bucket, key string) error {
	kv, err := c.js.KeyValue(bucket)
	if err != nil {
		return fmt.Errorf("bucket not found: %w", err)
	}

	err = kv.Delete(key)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	c.BroadcastEvent("kv_key_deleted", map[string]string{
		"bucket": bucket,
		"key":    key,
	})

	return nil
}

// KVKeys returns all keys in a bucket
func (c *Client) KVKeys(bucket string) ([]string, error) {
	kv, err := c.js.KeyValue(bucket)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	keys, err := kv.Keys()
	if errors.Is(err, nats.ErrNoKeysFound) {
		// An empty bucket is not an error.
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	return keys, nil
}
