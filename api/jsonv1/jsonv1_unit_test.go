//go:build unit

package jsonv1

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/ic2hrmk/promtail"
	"github.com/ic2hrmk/promtail/internal"
)

func Test_LokiJSONv1Exchanger_transformLogStreamsToDTO(t *testing.T) {
	timestamp := time.Now()
	type args struct {
		streams []*promtail.LogStream
	}
	tests := []struct {
		name string
		args args
		want *lokiDTOJsonV1PushRequest
	}{
		{
			name: "Regular transformation",
			args: args{
				streams: []*promtail.LogStream{
					{
						Level: promtail.Error,
						Labels: map[string]string{
							"instanceId": "instance-a1",
						},
						Entries: []*promtail.LogEntry{
							{
								Timestamp: timestamp,
								Format:    "regular error message, nothing to do with [%s] :)",
								Args:      []interface{}{"awesome argument"},
							},
						},
					},
				},
			},
			want: &lokiDTOJsonV1PushRequest{
				Streams: []*lokiDTOJsonV1Stream{
					{
						Stream: internal.CopyAndMergeLabels(
							map[string]string{
								"instanceId": "instance-a1",
							},
						),
						Values: [][2]string{{
							strconv.FormatInt(timestamp.UnixNano(), 10),
							promtail.Error.String() + ": " +
								fmt.Sprintf("regular error message, nothing to do with [%s] :)", []interface{}{"awesome argument"}...),
						}},
					},
				},
			},
		},
		{
			name: "NIL transformation",
			args: args{streams: nil},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exchanger := NewJSONv1Exchanger("loki").(*lokiJsonV1Exchanger)

			got := exchanger.transformLogStreamsToDTO(tt.args.streams)
			if !reflect.DeepEqual(got, tt.want) {
				rawGot, _ := json.Marshal(got)
				rawWant, _ := json.Marshal(tt.want)

				t.Errorf("transformLogStreamsToDTOJsonV1Push()\n got  = %s\n want = %s",
					string(rawGot), string(rawWant))
			}
		})
	}
}

func Test_LokiJSONv1Exchanger_format(t *testing.T) {
	timestamp := time.Now()
	type args struct {
		level   promtail.Level
		message string
		args    []interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Message with no arguments",
			args: args{
				level:   promtail.Info,
				message: "test with no arguments",
				args:    nil,
			},
			want: "INFO: test with no arguments",
		},
		{
			name: "Message with empty list of args",
			args: args{
				level:   promtail.Info,
				message: "test with no arguments",
				args:    []interface{}{},
			},
			want: "INFO: test with no arguments",
		},
		{
			name: "Message with with single argument",
			args: args{
				level:   promtail.Info,
				message: "test with arg [%d]",
				args:    []interface{}{timestamp.Unix()},
			},
			want: fmt.Sprintf("INFO: test with arg [%d]", timestamp.Unix()),
		},
	}

	exchanger := NewJSONv1Exchanger("loki").(*lokiJsonV1Exchanger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exchanger.formatMessage(promtail.Info, tt.args.message, tt.args.args...)
			if got != tt.want {
				t.Errorf("got unexpected format:\n"+
					"want: >>>%s<<<\ngot:  >>>%s<<<", tt.want, got)
			}
		})
	}
}
