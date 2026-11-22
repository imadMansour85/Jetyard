package nats

import "testing"

func TestParseVarzJetStreamObject(t *testing.T) {
	// Shape returned by NATS 2.10+ with JetStream enabled.
	body := []byte(`{
		"server_name": "n1",
		"version": "2.10.22",
		"jetstream": {
			"config": {"max_memory": 1073741824, "max_storage": 10737418240, "store_dir": "/data/jetstream"},
			"stats": {"memory": 2048, "storage": 4096, "accounts": 3}
		}
	}`)
	info, err := parseVarz(body)
	if err != nil {
		t.Fatal(err)
	}
	if !info.JetStream {
		t.Fatal("JetStream should be enabled")
	}
	js := info.JetStreamInfo
	if js == nil {
		t.Fatal("JetStreamInfo missing")
	}
	if js.MaxMemory != 1073741824 || js.MaxStorage != 10737418240 || js.StoreDir != "/data/jetstream" {
		t.Fatalf("limits = %+v", js)
	}
	if js.Memory != 2048 || js.Storage != 4096 || js.Accounts != 3 {
		t.Fatalf("usage = %+v", js)
	}
}

func TestParseVarzJetStreamDisabled(t *testing.T) {
	// A server without -js still sends the key, as an empty object.
	info, err := parseVarz([]byte(`{"server_name":"n1","jetstream":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	if info.JetStream || info.JetStreamInfo != nil {
		t.Fatalf("JetStream should be disabled, got %v %+v", info.JetStream, info.JetStreamInfo)
	}
}

func TestParseVarzJetStreamLegacyBool(t *testing.T) {
	info, err := parseVarz([]byte(`{"jetstream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if !info.JetStream {
		t.Fatal("legacy bool true should mean enabled")
	}
}
