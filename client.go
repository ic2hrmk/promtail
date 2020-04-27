package promtail

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	defaultSendBatchSize    = 5
	defaultSendBatchTimeout = 5 * time.Second
	exchangeQueueSize       = 1024
)

//
// Creates a Promtail client with a custom Streams exchanger
//	NOTE: options are applied before client start
//
func NewClient(exchanger StreamsExchanger, labels map[string]string, options ...clientOption) (Client, error) {
	if exchanger == nil {
		return nil, errors.New("exchanger is nil, no operations could be performed")
	}

	c := &promtailClient{
		exchanger: exchanger,
		queue:     make(chan packedLogEntry, exchangeQueueSize),

		errorHandler: func(err error) {
			if err != nil {
				log.Printf("failed to perform logs exchange with Loki: %s", err)
			}
		},

		sendBatchTimeout: defaultSendBatchTimeout,
		sendBatchSize:    defaultSendBatchSize,

		stopSignal:  make(chan struct{}),
		stopAwaiter: make(chan struct{}),
	}

	for i := range options {
		options[i](c)
	}

	go c.exchange(copyLabels(labels))

	return c, nil
}

func NewJSONv1Client(lokiAddress string, defaultLabels map[string]string, options ...clientOption) (Client, error) {
	if !(strings.HasPrefix(lokiAddress, "http://") ||
		strings.HasPrefix(lokiAddress, "https://")) {
		lokiAddress = "http://" + lokiAddress
	}

	return NewClient(NewJSONv1Exchanger(lokiAddress), defaultLabels, options...)
}

func WithSendBatchSize(batchSize uint) clientOption {
	return func(c *promtailClient) {
		c.sendBatchSize = batchSize
	}
}

func WithSendBatchTimeout(sendTimeout time.Duration) clientOption {
	return func(c *promtailClient) {
		if sendTimeout <= 0 {
			return
		}

		c.sendBatchTimeout = sendTimeout
	}
}

func WithErrorCallback(errorHandler func(err error)) clientOption {
	return func(c *promtailClient) {
		c.errorHandler = errorHandler
	}
}

type clientOption func(c *promtailClient)

type packedLogEntry struct {
	level    Level
	labels   map[string]string
	logEntry *LogEntry
}

type promtailClient struct {
	errorHandler func(error)

	sendBatchSize    uint
	sendBatchTimeout time.Duration

	queue     chan packedLogEntry
	exchanger StreamsExchanger

	isStopped   bool
	stopSignal  chan struct{}
	stopAwaiter chan struct{}
	stopOnce    sync.Once
}

func (rcv *promtailClient) Ping() (*PongResponse, error) {
	return rcv.exchanger.Ping()
}

func (rcv *promtailClient) Logf(level Level, format string, args ...interface{}) {
	rcv.LogfWithLabels(level, nil, format, args...)
}

func (rcv *promtailClient) LogfWithLabels(level Level, labels map[string]string, format string, args ...interface{}) {
	if rcv.isStopped { // Escape from endless lock
		log.Println("promtail client is stopped, no log entries will be sent!")
		return
	}

	rcv.queue <- packedLogEntry{
		labels: copyLabels(labels),
		level:  level,
		logEntry: &LogEntry{
			Timestamp: time.Now(),
			Format:    format,
			Args:      args,
		},
	}
}

func (rcv *promtailClient) Debugf(format string, args ...interface{}) {
	rcv.Logf(Debug, format, args...)
}

func (rcv *promtailClient) Infof(format string, args ...interface{}) {
	rcv.Logf(Info, format, args...)
}

func (rcv *promtailClient) Warnf(format string, args ...interface{}) {
	rcv.Logf(Warn, format, args...)
}

func (rcv *promtailClient) Errorf(format string, args ...interface{}) {
	rcv.Logf(Error, format, args...)
}

func (rcv *promtailClient) Fatalf(format string, args ...interface{}) {
	rcv.Logf(Fatal, format, args...)
}

func (rcv *promtailClient) Panicf(format string, args ...interface{}) {
	rcv.Logf(Panic, format, args...)
}

func (rcv *promtailClient) Close() {
	rcv.stopOnce.Do(func() {
		rcv.isStopped = true  // Deny new incoming logs
		close(rcv.stopSignal) // Send stop signal
		<-rcv.stopAwaiter     // Await for stop signal response
	})
}

func (rcv *promtailClient) exchange(defaultLabels map[string]string) {
	var (
		err error

		incomeLogEntry packedLogEntry
		batch          = newBatch(defaultLabels)
		batchTimer     = time.NewTimer(rcv.sendBatchTimeout)
	)

exchangeLoop:
	for {

		select {

		// On new log message
		case incomeLogEntry = <-rcv.queue:
			{
				batch.add(incomeLogEntry)

				if batch.countEntries() >= rcv.sendBatchSize {
					err = rcv.exchanger.Push(batch.getStreams())
					if err != nil {
						rcv.errorHandler(err)
					}

					batch.reset()
					batchTimer.Reset(rcv.sendBatchTimeout)
				}
			}

		// On send timeout
		case <-batchTimer.C:
			{
				if batch.countEntries() > 0 {
					err = rcv.exchanger.Push(batch.getStreams())
					if err != nil {
						rcv.errorHandler(err)
					}

					batch.reset()
				}

				batchTimer.Reset(rcv.sendBatchTimeout)
			}

		// On client stop
		case <-rcv.stopSignal:
			{
				batchTimer.Stop()
				if batch.countEntries() > 0 {
					err = rcv.exchanger.Push(batch.getStreams())
					if err != nil {
						rcv.errorHandler(err)
					}
				}

				rcv.stopAwaiter <- struct{}{}
				break exchangeLoop
			}

		}
	}
}

type logStreamBatch struct {
	size             uint
	predefinedLabels map[string]string
	streams          []*LogStream
}

func newBatch(predefinedLabels map[string]string) *logStreamBatch {
	rcv := &logStreamBatch{predefinedLabels: copyLabels(predefinedLabels)}
	rcv.reset()
	return rcv
}

func (rcv *logStreamBatch) add(entry packedLogEntry) {
	rcv.size += 1

	cachedIndex := rcv._getLevelIndex(entry.level)

	// For both use cases (custom labels and unknown log level we would add entry in a separate stream)
	if len(entry.labels) > 0 || cachedIndex < 0 {
		dedicatedStream := newLeveledStream(entry.level, rcv.predefinedLabels, entry.labels)
		dedicatedStream.Entries = []*LogEntry{entry.logEntry}
		rcv.streams = append(rcv.streams, dedicatedStream)
	} else {
		// Or add to a cached stream :)
		rcv.streams[cachedIndex].Entries = append(rcv.streams[cachedIndex].Entries,
			entry.logEntry)
	}
}

func (rcv *logStreamBatch) reset() {
	rcv.size = 0
	rcv.streams = make([]*LogStream, len(rcv._getCachedLevels()))
	rcv.streams[rcv._getLevelIndex(Debug)] = newLeveledStream(Debug, rcv.predefinedLabels)
	rcv.streams[rcv._getLevelIndex(Info)] = newLeveledStream(Info, rcv.predefinedLabels)
	rcv.streams[rcv._getLevelIndex(Warn)] = newLeveledStream(Warn, rcv.predefinedLabels)
	rcv.streams[rcv._getLevelIndex(Error)] = newLeveledStream(Error, rcv.predefinedLabels)
	rcv.streams[rcv._getLevelIndex(Panic)] = newLeveledStream(Panic, rcv.predefinedLabels)
	rcv.streams[rcv._getLevelIndex(Fatal)] = newLeveledStream(Fatal, rcv.predefinedLabels)
}

func (rcv *logStreamBatch) getStreams() []*LogStream {
	return rcv.streams
}

func (rcv *logStreamBatch) countEntries() uint {
	return rcv.size
}

func (rcv *logStreamBatch) _getLevelIndex(level Level) int {
	switch level {
	case Debug:
		return 0
	case Info:
		return 1
	case Warn:
		return 2
	case Error:
		return 3
	case Panic:
		return 4
	case Fatal:
		return 5
	default:
		return -1
	}
}

func (rcv *logStreamBatch) _getCachedLevels() []Level {
	return []Level{Debug, Info, Warn, Error, Panic, Fatal}
}

func newLeveledStream(level Level, predefinedLabels ...map[string]string) *LogStream {
	return &LogStream{
		Level: level,
		Labels: copyAndMergeLabels(append(
			predefinedLabels,
			map[string]string{logLevelForcedLabel: level.String()},
		)...),
	}
}
