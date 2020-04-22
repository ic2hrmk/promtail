// +build unit

package promtail

import (
	"math/rand"
	"testing"
	"time"
)

func TestPromtailClient_Constructor(t *testing.T) {
	type args struct {
		exchanger StreamsExchanger
		options   []clientOption
	}
	tests := []struct {
		name    string
		args    args
		want    *promtailClient
		wantErr bool
	}{
		{
			name: "JSON client with no options",
			args: args{
				exchanger: NewJSONv1Exchanger("loki:3100"),
				options:   nil,
			},
			want: &promtailClient{
				sendBatchTimeout: defaultSendBatchTimeout,
				sendBatchSize:    defaultSendBatchSize,
			},
			wantErr: false,
		},
		{
			name: "NIL stream exchanger client with no options",
			args: args{
				exchanger: nil,
				options:   nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "JSON client with custom send batch size and batch timeout",
			args: args{
				exchanger: NewJSONv1Exchanger("loki:3100"),
				options: []clientOption{
					WithSendBatchTimeout(100 * time.Second),
					WithSendBatchSize(100),
				},
			},
			want: &promtailClient{
				sendBatchTimeout: 100 * time.Second,
				sendBatchSize:    100,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := NewClient(tt.args.exchanger, nil, tt.args.options...)
			{
				if err != nil {
					if !tt.wantErr {
						t.Errorf("unexpected error on client initialization: %s", err)
					}
					return
				}

				if tt.wantErr {
					t.Errorf("expected error on initilization, but not occured :(")
					got.Close()
					return
				}
			}

			// Prevent logs from being send
			got.Close()

			unsafeClient := got.(*promtailClient) // Will panic if we would produce some strange client

			if unsafeClient.sendBatchSize != tt.want.sendBatchSize {
				t.Errorf("batch size is invalid, got: %d, expected: %d",
					unsafeClient.sendBatchSize, tt.want.sendBatchSize)
			}

			if unsafeClient.sendBatchTimeout != tt.want.sendBatchTimeout {
				t.Errorf("batch timeout is invalid, got: %d, expected: %d",
					unsafeClient.sendBatchTimeout, tt.want.sendBatchTimeout)
			}
		})
	}
}

func TestPromtailClient_Batch_Scenario(t *testing.T) {
	var (
		predefinedlabels = map[string]string{
			"instanceId": "abc123",
		}
	)

	batch := newBatch(predefinedlabels)

	//
	// Verify initialization
	//

	if batch.countEntries() != 0 {
		t.Fatal("incorrect number of entries at empty batch")
	}
	if len(batch.streams) != len(batch._getCachedLevels()) {
		t.Fatal("incorrect number of precached streams in empty batch")
	}

	//
	// Fulfil batch
	//

	var (
		randomLogsNumber           = 100 + rand.Intn(100)
		logsWithCustomLabelsNumber = 0
	)

	for i := 0; i < randomLogsNumber; i++ {
		var (
			logLvl, logMsg = generateLogMessage()
			logLabels      = make(map[string]string)
		)

		// Add custom labels with 25% probability
		if rand.Intn(100) > 75 {
			logLabels["customlabel"+time.Now().String()] = time.Now().String()
			logsWithCustomLabelsNumber += 1
		}

		batch.add(packedLogEntry{
			level:  logLvl,
			labels: logLabels,
			logEntry: &LogEntry{
				Timestamp: time.Now(),
				Format:    logMsg,
			},
		})
	}

	if batch.countEntries() != uint(randomLogsNumber) {
		t.Fatalf("incorrect number of fetched log entries, want = %d, got = %d",
			randomLogsNumber, batch.countEntries())
	}

	if (len(batch._getCachedLevels()) + logsWithCustomLabelsNumber) != len(batch.getStreams()) {
		t.Fatalf("incorrect number streams, probably, caching works incorrect with custom labels")
	}

	//
	// Verify batch reset
	//

	batch.reset()

	if batch.countEntries() != 0 {
		t.Fatalf("incorrect number of entries in re-initialized batch, want = %d, got  = %d",
			0, batch.countEntries())
	}
	if len(batch.getStreams()) != len(batch._getCachedLevels()) {
		t.Fatalf("incorrect number of precached streams in re-initialized batch, want = %d, got  = %d",
			len(batch._getCachedLevels()), len(batch.getStreams()))
	}
}
