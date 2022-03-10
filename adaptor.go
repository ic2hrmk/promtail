package promtail

import (
	"time"
)

type LogStream struct {
	Level   Level
	Labels  map[string]string
	Entries []*LogEntry
}

type LogEntry struct {
	Timestamp time.Time
	Format    string
	Args      []interface{}
}

const (
	logLevelForcedLabel = "logLevel"
)

type StreamsExchanger interface {
	Push(streams []*LogStream) error
	Ping() (*PongResponse, error)
}
