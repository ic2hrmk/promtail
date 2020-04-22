package promtail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
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

func NewLeveledStream(level Level, predefinedLabels ...map[string]string) *LogStream {
	return &LogStream{
		Level: level,
		Labels: copyAndMergeLabels(append(
			predefinedLabels,
			map[string]string{logLevelForcedLabel: level.String()},
		)...),
	}
}

type StreamsExchanger interface {
	Push(streams []*LogStream) error
}

//
// Creates a client with direct send logic (nor batch neither queue) capable to
// exchange with Loki v1 API via JSON
//	Read more at: https://github.com/grafana/loki/blob/master/docs/api.md#post-lokiapiv1push
//
func NewJSONv1Exchanger(lokiAddress string) StreamsExchanger {
	return &lokiJsonV1Exchanger{
		restClient:  &http.Client{},
		lokiAddress: lokiAddress,
	}
}

type lokiJsonV1Exchanger struct {
	restClient  *http.Client
	lokiAddress string
}

//
//	Data transfer objects are restored from `push API` description:
//		https://github.com/grafana/loki/blob/master/docs/api.md#post-lokiapiv1push
//	{
//		"streams": [
//			{
//				"stream": {
//					"label": "value"
//				},
//				"values": [
//					[ "<unix epoch in nanoseconds>", "<log line>" ],
//					[ "<unix epoch in nanoseconds>", "<log line>" ]
//				]
//			}
//		]
//	}
//
type (
	lokiDTOJsonV1PushRequest struct {
		Streams []*lokiDTOJsonV1Stream `json:"streams"`
	}

	lokiDTOJsonV1Stream struct {
		Stream map[string]string `json:"stream"`
		Values [][2]string       `json:"values"`
	}
)

func (rcv *lokiJsonV1Exchanger) Push(streams []*LogStream) error {
	var (
		pushMessage       = rcv.transformLogStreamsToDTO(streams)
		rawPushMessage, _ = json.Marshal(pushMessage)
	)

	resp, err := rcv.restClient.Post(
		rcv.lokiAddress+"/loki/api/v1/push",
		"application/json",
		bytes.NewBuffer(rawPushMessage),
	)
	if err != nil {
		return fmt.Errorf("failed to send push message: %s", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if !(199 < resp.StatusCode && resp.StatusCode < 300) {
		messageBody, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("unexpected response code [code=%d], message: %s",
			resp.StatusCode, string(messageBody))
	}

	return nil
}

func (rcv *lokiJsonV1Exchanger) transformLogStreamsToDTO(streams []*LogStream) *lokiDTOJsonV1PushRequest {
	if streams == nil {
		return nil
	}

	pushRequest := &lokiDTOJsonV1PushRequest{
		Streams: make([]*lokiDTOJsonV1Stream, 0, len(streams)),
	}

	for i := range streams {
		if streams[i] == nil || len(streams[i].Entries) == 0 {
			continue
		}

		lokiStream := &lokiDTOJsonV1Stream{
			Stream: streams[i].Labels,
			Values: make([][2]string, 0, len(streams[i].Entries)),
		}

		for j := range streams[i].Entries {
			if streams[i].Entries[j] == nil {
				continue
			}

			lokiStream.Values = append(lokiStream.Values, [2]string{
				strconv.FormatInt(streams[i].Entries[j].Timestamp.UnixNano(), 10),
				rcv.formatMessage(streams[i].Level, streams[i].Entries[j].Format, streams[i].Entries[j].Args...),
			})
		}

		pushRequest.Streams = append(pushRequest.Streams, lokiStream)
	}

	return pushRequest
}

func (rcv *lokiJsonV1Exchanger) formatMessage(lvl Level, format string, args ...interface{}) string {
	return lvl.String() + ": " + fmt.Sprintf(format, args...)
}
