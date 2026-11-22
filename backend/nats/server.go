package nats

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/imadMansour85/Jetyard/backend/models"
)

// monitorURL builds a NATS monitoring endpoint URL (e.g. /varz, /connz) from
// the server this client is actually connected to, rather than assuming a
// fixed hostname.
func (c *Client) monitorURL(path string) (string, error) {
	addr := c.conn.ConnectedAddr()
	if addr == "" {
		return "", fmt.Errorf("no connected servers")
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("failed to parse connected server address %q: %w", addr, err)
	}

	monitorPort := os.Getenv("NATS_MONITOR_PORT")
	if monitorPort == "" {
		monitorPort = "8222"
	}

	scheme := "http"
	if c.conn.TLSRequired() {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s", scheme, net.JoinHostPort(host, monitorPort), path), nil
}

var monitorHTTP = &http.Client{Timeout: 5 * time.Second}

// fetchMonitor GETs a monitoring endpoint and returns its body, failing on a
// non-200 response instead of handing an error page to the JSON decoder.
func (c *Client) fetchMonitor(path string) ([]byte, error) {
	url, err := c.monitorURL(path)
	if err != nil {
		return nil, err
	}

	resp, err := monitorHTTP.Get(url)
	if err != nil {
		return nil, fmt.Errorf("monitoring endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("monitoring endpoint %s returned %s", path, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return body, nil
}

// GetServerInfo fetches server information from NATS monitoring endpoint
func (c *Client) GetServerInfo() (*models.ServerInfo, error) {
	body, err := c.fetchMonitor("/varz")
	if err != nil {
		return nil, err
	}
	return parseVarz(body)
}

// parseJetStream reads the "jetstream" field of /varz. Old servers sent a
// bool; NATS 2.2+ sends an object that has "config" only when JetStream is
// enabled (a disabled server sends {}).
func parseJetStream(v interface{}) (bool, *models.JetStreamInfo) {
	switch js := v.(type) {
	case bool:
		return js, nil
	case map[string]interface{}:
		cfg, ok := js["config"].(map[string]interface{})
		if !ok {
			return false, nil
		}
		info := &models.JetStreamInfo{
			MaxMemory:  getInt64(cfg, "max_memory"),
			MaxStorage: getInt64(cfg, "max_storage"),
			StoreDir:   getString(cfg, "store_dir"),
		}
		if stats, ok := js["stats"].(map[string]interface{}); ok {
			info.Memory = getUint64(stats, "memory")
			info.Storage = getUint64(stats, "storage")
			info.Accounts = getInt(stats, "accounts")
		}
		return true, info
	}
	return false, nil
}

// parseVarz turns a /varz response body into ServerInfo.
func parseVarz(body []byte) (*models.ServerInfo, error) {
	var varz map[string]interface{}
	if err := json.Unmarshal(body, &varz); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	jsEnabled, jsInfo := parseJetStream(varz["jetstream"])

	// Parse the varz data into our model
	serverInfo := &models.ServerInfo{
		ServerID:      getString(varz, "server_id"),
		ServerName:    getString(varz, "server_name"),
		Version:       getString(varz, "version"),
		Proto:         getInt(varz, "proto"),
		GitCommit:     getString(varz, "git_commit"),
		GoVersion:     getString(varz, "go"),
		Host:          getString(varz, "host"),
		Port:          getInt(varz, "port"),
		MaxPayload:    getInt64(varz, "max_payload"),
		JetStream:     jsEnabled,
		JetStreamInfo: jsInfo,
		Now:           time.Now(),
		Uptime:        getString(varz, "uptime"),
		Mem:           getInt64(varz, "mem"),
		Cores:         getInt(varz, "cores"),
		CPU:           getFloat64(varz, "cpu"),
		Connections:   getInt(varz, "connections"),
		TotalConns:    getUint64(varz, "total_connections"),
		Routes:        getInt(varz, "routes"),
		Remotes:       getInt(varz, "remotes"),
		Leafnodes:     getInt(varz, "leaf_nodes"),
		InMsgs:        getUint64(varz, "in_msgs"),
		OutMsgs:       getUint64(varz, "out_msgs"),
		InBytes:       getUint64(varz, "in_bytes"),
		OutBytes:      getUint64(varz, "out_bytes"),
		SlowConsumers: getUint64(varz, "slow_consumers"),
	}

	// Parse cluster info if present
	if cluster, ok := varz["cluster"].(map[string]interface{}); ok {
		serverInfo.Cluster = &models.ClusterInfo{
			Name: getString(cluster, "name"),
			Addr: getString(cluster, "addr"),
		}
		if urls, ok := cluster["urls"].([]interface{}); ok {
			for _, url := range urls {
				if urlStr, ok := url.(string); ok {
					serverInfo.Cluster.URLs = append(serverInfo.Cluster.URLs, urlStr)
				}
			}
		}
	}

	return serverInfo, nil
}

// GetConnections fetches active client connections, including their
// subscriptions, from the NATS monitoring endpoint.
func (c *Client) GetConnections() ([]models.ClientConnection, error) {
	body, err := c.fetchMonitor("/connz?subs=1")
	if err != nil {
		return nil, err
	}

	var connz struct {
		Connections []map[string]interface{} `json:"connections"`
	}
	if err := json.Unmarshal(body, &connz); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	connections := make([]models.ClientConnection, 0, len(connz.Connections))
	for _, conn := range connz.Connections {
		client := models.ClientConnection{
			CID:      getUint64(conn, "cid"),
			IP:       getString(conn, "ip"),
			Port:     getInt(conn, "port"),
			Name:     getString(conn, "name"),
			Lang:     getString(conn, "lang"),
			Version:  getString(conn, "version"),
			Start:    getString(conn, "start"),
			InMsgs:   getUint64(conn, "in_msgs"),
			OutMsgs:  getUint64(conn, "out_msgs"),
			InBytes:  getUint64(conn, "in_bytes"),
			OutBytes: getUint64(conn, "out_bytes"),
		}

		if subs, ok := conn["subscriptions_list"].([]interface{}); ok {
			for _, s := range subs {
				if subj, ok := s.(string); ok {
					client.Subscriptions = append(client.Subscriptions, subj)
				}
			}
		}

		connections = append(connections, client)
	}

	return connections, nil
}

// Helper functions to safely extract values from map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func getUint64(m map[string]interface{}, key string) uint64 {
	if v, ok := m[key].(float64); ok {
		return uint64(v)
	}
	return 0
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}
