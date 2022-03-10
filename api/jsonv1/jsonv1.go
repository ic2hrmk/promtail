package jsonv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/ic2hrmk/promtail"
)

//
// NewJSONv1Exchanger Creates a client with direct send logic (nor batch neither queue)
// capable to exchange with Loki v1 API via JSON
//	Read more at: https://grafana.com/docs/loki/latest/api/#post-lokiapiv1push
//
func NewJSONv1Exchanger(lokiAddress string) promtail.StreamsExchanger {
	return &lokiJsonV1Exchanger{
		restClient:  &http.Client{},
		lokiAddress: lokiAddress,
	}
}

const (
	requestTimeout = 5 * time.Second
)

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

func (rcv *lokiJsonV1Exchanger) Push(streams []*promtail.LogStream) error {
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

func (rcv *lokiJsonV1Exchanger) Ping() (*promtail.PongResponse, error) {
	var (
		timeout, cancel  = context.WithTimeout(context.Background(), requestTimeout)
		pingRequest, err = http.NewRequestWithContext(timeout, http.MethodGet, rcv.lokiAddress+"/ready", nil)
	)
	defer cancel()

	if err != nil {
		return nil, fmt.Errorf("unable to build ping request: %s", err)
	}

	resp, err := rcv.restClient.Do(pingRequest)
	if err != nil {
		return nil, fmt.Errorf("pong is not received: %s", err)
	}

	defer func() { _ = resp.Body.Close() }()

	pong := &promtail.PongResponse{}

	if rcv.isSuccessHTTPCode(resp.StatusCode) {
		pong.IsReady = true
	}

	return pong, nil
}

func (rcv *lokiJsonV1Exchanger) transformLogStreamsToDTO(streams []*promtail.LogStream) *lokiDTOJsonV1PushRequest {
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

func (rcv *lokiJsonV1Exchanger) formatMessage(lvl promtail.Level, format string, args ...interface{}) string {
	return lvl.String() + ": " + fmt.Sprintf(format, args...)
}

func (rcv *lokiJsonV1Exchanger) isSuccessHTTPCode(code int) bool {
	return 199 < code && code < 300
}
