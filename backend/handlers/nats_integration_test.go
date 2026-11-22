//go:build integration

// Integration tests: the real HTTP handlers against a real nats-server with
// JetStream and monitoring enabled. Run with
//
//	nats-server -js -m 8222 &
//	go test -tags integration ./handlers/
//
// NATS_TEST_URL (default nats://127.0.0.1:4222) and NATS_MONITOR_PORT
// (default 8222) point the tests at another server.
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/imadMansour85/Jetyard/backend/models"
	"github.com/imadMansour85/Jetyard/backend/nats"
)

func natsRouter(t *testing.T) http.Handler {
	t.Helper()
	url := os.Getenv("NATS_TEST_URL")
	if url == "" {
		url = "nats://127.0.0.1:4222"
	}
	nc, err := nats.NewClient(url)
	if err != nil {
		t.Fatalf("connect to %s (start nats-server -js -m 8222): %v", url, err)
	}
	t.Cleanup(nc.Close)

	h := NewHandler(nc)
	kv := NewKVHandler(nc)
	mux := http.NewServeMux()
	// Same routes as main.go, without the auth layer.
	mux.HandleFunc("GET /api/server", h.GetServerInfo)
	mux.HandleFunc("GET /api/connections", h.ListConnections)
	mux.HandleFunc("GET /api/streams", h.ListStreams)
	mux.HandleFunc("GET /api/streams/{name}", h.GetStream)
	mux.HandleFunc("POST /api/streams", h.CreateStream)
	mux.HandleFunc("DELETE /api/streams/{name}", h.DeleteStream)
	mux.HandleFunc("GET /api/consumers", h.ListConsumers)
	mux.HandleFunc("POST /api/consumers", h.CreateConsumer)
	mux.HandleFunc("DELETE /api/consumers/{name}", h.DeleteConsumer)
	mux.HandleFunc("POST /api/publish", h.PublishMessage)
	mux.HandleFunc("GET /api/messages/{stream}", h.GetMessages)
	mux.HandleFunc("GET /api/streams/{stream}/messages", h.ListStreamMessages)
	mux.HandleFunc("GET /api/streams/{stream}/messages/{seq}", h.GetMessageBySeq)
	mux.HandleFunc("GET /api/kv/buckets", kv.ListBuckets)
	mux.HandleFunc("POST /api/kv/buckets", kv.CreateBucket)
	mux.HandleFunc("GET /api/kv/buckets/{name}", kv.GetBucket)
	mux.HandleFunc("DELETE /api/kv/buckets/{name}", kv.DeleteBucket)
	mux.HandleFunc("GET /api/kv/buckets/{bucket}/keys", kv.ListKeys)
	mux.HandleFunc("GET /api/kv/buckets/{bucket}/keys/{key}", kv.GetKey)
	mux.HandleFunc("PUT /api/kv/buckets/{bucket}/keys/{key}", kv.PutKey)
	mux.HandleFunc("DELETE /api/kv/buckets/{bucket}/keys/{key}", kv.DeleteKey)
	return mux
}

// expect performs a request and fails unless it returns wantStatus. It
// decodes a JSON body into out when out is non-nil.
func expect(t *testing.T, h http.Handler, method, path, body string, wantStatus int, out any) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	if w.Code != wantStatus {
		t.Fatalf("%s %s = %d, want %d: %s", method, path, w.Code, wantStatus, w.Body)
	}
	if out != nil {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			t.Fatalf("%s %s: bad JSON %q: %v", method, path, w.Body, err)
		}
	}
}

// unique avoids clashes between runs against a shared server.
func unique(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func TestIntegrationServerInfo(t *testing.T) {
	h := natsRouter(t)
	var info models.ServerInfo
	expect(t, h, "GET", "/api/server", "", http.StatusOK, &info)
	if !info.JetStream || info.JetStreamInfo == nil {
		t.Fatalf("server must report JetStream enabled with limits, got %v %+v", info.JetStream, info.JetStreamInfo)
	}
	if info.Version == "" {
		t.Fatal("server version is empty")
	}

	var conns []models.ClientConnection
	expect(t, h, "GET", "/api/connections", "", http.StatusOK, &conns)
	if len(conns) == 0 {
		t.Fatal("expected at least the backend's own connection")
	}
}

func TestIntegrationStreamConsumerMessageLifecycle(t *testing.T) {
	h := natsRouter(t)
	stream := unique("IT_ORDERS")
	subject := strings.ToLower(stream) + ".created"

	expect(t, h, "POST", "/api/streams",
		fmt.Sprintf(`{"name":%q,"subjects":[%q],"storage":"memory"}`, stream, strings.ToLower(stream)+".>"),
		http.StatusCreated, nil)
	t.Cleanup(func() {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("DELETE", "/api/streams/"+stream, nil))
	})

	// Creating it again with overlapping subjects is a client error, not a 500.
	expect(t, h, "POST", "/api/streams",
		fmt.Sprintf(`{"name":%q,"subjects":[%q]}`, stream+"_DUP", strings.ToLower(stream)+".>"),
		http.StatusBadRequest, nil)

	var s models.Stream
	expect(t, h, "GET", "/api/streams/"+stream, "", http.StatusOK, &s)
	if s.Name != stream {
		t.Fatalf("got stream %q", s.Name)
	}

	for i := 1; i <= 3; i++ {
		expect(t, h, "POST", "/api/publish", fmt.Sprintf(`{"subject":%q,"data":"order-%d"}`, subject, i), http.StatusOK, nil)
	}

	// Publish is fire-and-forget; give the stream a moment to store them.
	deadline := time.Now().Add(2 * time.Second)
	for {
		expect(t, h, "GET", "/api/streams/"+stream, "", http.StatusOK, &s)
		if s.Messages == 3 || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if s.Messages != 3 {
		t.Fatalf("stream has %d messages, want 3", s.Messages)
	}

	var msgs []models.MessageDetail
	expect(t, h, "GET", "/api/streams/"+stream+"/messages?limit=10", "", http.StatusOK, &msgs)
	if len(msgs) != 3 || msgs[0].Data != "order-1" {
		t.Fatalf("browser messages = %+v", msgs)
	}
	var one models.MessageDetail
	expect(t, h, "GET", "/api/streams/"+stream+"/messages/2", "", http.StatusOK, &one)
	if one.Data != "order-2" {
		t.Fatalf("message 2 = %+v", one)
	}
	expect(t, h, "GET", "/api/streams/"+stream+"/messages/99", "", http.StatusNotFound, nil)

	consumer := "it_worker"
	expect(t, h, "POST", "/api/consumers",
		fmt.Sprintf(`{"stream_name":%q,"name":%q,"deliver_policy":"all","ack_policy":"explicit"}`, stream, consumer),
		http.StatusCreated, nil)
	var consumers []models.Consumer
	expect(t, h, "GET", "/api/consumers?stream="+stream, "", http.StatusOK, &consumers)
	if len(consumers) != 1 || consumers[0].Name != consumer || consumers[0].NumPending != 3 {
		t.Fatalf("consumers = %+v", consumers)
	}
	expect(t, h, "DELETE", "/api/consumers/"+consumer+"?stream="+stream, "", http.StatusOK, nil)
	expect(t, h, "DELETE", "/api/consumers/"+consumer+"?stream="+stream, "", http.StatusNotFound, nil)

	expect(t, h, "DELETE", "/api/streams/"+stream, "", http.StatusOK, nil)
	expect(t, h, "DELETE", "/api/streams/"+stream, "", http.StatusNotFound, nil)
	expect(t, h, "GET", "/api/streams/"+stream, "", http.StatusNotFound, nil)
	expect(t, h, "GET", "/api/consumers?stream="+stream, "", http.StatusNotFound, nil)
}

func TestIntegrationKVLifecycle(t *testing.T) {
	h := natsRouter(t)
	bucket := unique("it_cfg")

	expect(t, h, "POST", "/api/kv/buckets", fmt.Sprintf(`{"name":%q,"history":3}`, bucket), http.StatusCreated, nil)
	t.Cleanup(func() {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("DELETE", "/api/kv/buckets/"+bucket, nil))
	})

	// An empty bucket lists no keys instead of failing.
	var keys []string
	expect(t, h, "GET", "/api/kv/buckets/"+bucket+"/keys", "", http.StatusOK, &keys)
	if len(keys) != 0 {
		t.Fatalf("new bucket has keys %v", keys)
	}

	expect(t, h, "PUT", "/api/kv/buckets/"+bucket+"/keys/feature.flag", `{"value":"on"}`, http.StatusOK, nil)
	var entry models.KVEntry
	expect(t, h, "GET", "/api/kv/buckets/"+bucket+"/keys/feature.flag", "", http.StatusOK, &entry)
	if entry.Value != "on" {
		t.Fatalf("entry = %+v", entry)
	}
	expect(t, h, "GET", "/api/kv/buckets/"+bucket+"/keys", "", http.StatusOK, &keys)
	if len(keys) != 1 || keys[0] != "feature.flag" {
		t.Fatalf("keys = %v", keys)
	}

	expect(t, h, "DELETE", "/api/kv/buckets/"+bucket+"/keys/feature.flag", "", http.StatusOK, nil)
	expect(t, h, "GET", "/api/kv/buckets/"+bucket+"/keys/feature.flag", "", http.StatusNotFound, nil)

	expect(t, h, "DELETE", "/api/kv/buckets/"+bucket, "", http.StatusOK, nil)
	expect(t, h, "GET", "/api/kv/buckets/"+bucket, "", http.StatusNotFound, nil)
	expect(t, h, "DELETE", "/api/kv/buckets/"+bucket, "", http.StatusNotFound, nil)
}
