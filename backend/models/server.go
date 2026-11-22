package models

import "time"

// ServerInfo represents NATS server information
type ServerInfo struct {
	ServerID   string `json:"server_id"`
	ServerName string `json:"server_name"`
	Version    string `json:"version"`
	Proto      int    `json:"proto"`
	GitCommit  string `json:"git_commit"`
	GoVersion  string `json:"go"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	MaxPayload int64  `json:"max_payload"`
	JetStream  bool   `json:"jetstream"`

	// JetStreamInfo holds limits and usage when JetStream is enabled.
	JetStreamInfo *JetStreamInfo `json:"jetstream_info,omitempty"`

	// Runtime stats
	Now         time.Time `json:"now"`
	Uptime      string    `json:"uptime"`
	Mem         int64     `json:"mem"`
	Cores       int       `json:"cores"`
	CPU         float64   `json:"cpu"`
	Connections int       `json:"connections"`
	TotalConns  uint64    `json:"total_connections"`
	Routes      int       `json:"routes"`
	Remotes     int       `json:"remotes"`
	Leafnodes   int       `json:"leafnodes"`

	// Message stats
	InMsgs        uint64 `json:"in_msgs"`
	OutMsgs       uint64 `json:"out_msgs"`
	InBytes       uint64 `json:"in_bytes"`
	OutBytes      uint64 `json:"out_bytes"`
	SlowConsumers uint64 `json:"slow_consumers"`

	// Cluster info
	Cluster *ClusterInfo `json:"cluster,omitempty"`
}

// JetStreamInfo is the JetStream section of /varz: configured limits and
// current usage, in bytes.
type JetStreamInfo struct {
	MaxMemory  int64  `json:"max_memory"`
	MaxStorage int64  `json:"max_storage"`
	StoreDir   string `json:"store_dir,omitempty"`
	Memory     uint64 `json:"memory"`
	Storage    uint64 `json:"storage"`
	Accounts   int    `json:"accounts"`
}

// ClusterInfo represents cluster information
type ClusterInfo struct {
	Name string   `json:"name,omitempty"`
	Addr string   `json:"addr,omitempty"`
	URLs []string `json:"urls,omitempty"`
}

// ClientConnection represents a single active NATS client connection,
// including which subjects it is subscribed to.
type ClientConnection struct {
	CID           uint64   `json:"cid"`
	IP            string   `json:"ip"`
	Port          int      `json:"port"`
	Name          string   `json:"name,omitempty"`
	Lang          string   `json:"lang,omitempty"`
	Version       string   `json:"version,omitempty"`
	Start         string   `json:"start,omitempty"`
	InMsgs        uint64   `json:"in_msgs"`
	OutMsgs       uint64   `json:"out_msgs"`
	InBytes       uint64   `json:"in_bytes"`
	OutBytes      uint64   `json:"out_bytes"`
	Subscriptions []string `json:"subscriptions,omitempty"`
}
