//go:build externaljsonv1

package jsonv1_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ic2hrmk/promtail"
	"github.com/ic2hrmk/promtail/api/jsonv1"
	"github.com/ic2hrmk/promtail/internal/testsuite"
)

func TestJsonV1Client_Ping(t *testing.T) {
	t.Log("Starting JSON V1 client test")
	testsuite.PrintUsedEnvVarNames(t,
		testsuite.TestLokiAddressEnv,
	)
	testsuite.ValidateIsLokiReady(t)

	var (
		lokiAddress = testsuite.TestLokiAddress
	)

	client, err := promtail.NewClient(jsonv1.NewJSONv1Exchanger(lokiAddress), nil)
	if err != nil {
		t.Fatalf("unable to initialize client: %s", err)
	}

	defer client.Close()

	pong, err := client.Ping()

	if err != nil {
		t.Fatalf("unexpected error occured during ping: %s", err)
	}

	if !pong.IsReady {
		t.Error("pong response says that Loki is not ready, but it is")
	}
}

func TestJsonV1Client_Logf_External(t *testing.T) {
	t.Log("Starting JSON V1 client test")
	testsuite.PrintUsedEnvVarNames(t,
		testsuite.TestLokiAddressEnv,
		testsuite.TestRequestsNumberEnv,
	)
	testsuite.ValidateIsLokiReady(t)

	var (
		lokiAddress    = testsuite.TestLokiAddress
		requestsNumber = testsuite.TestRequestsNumber

		labels = map[string]string{
			"testName": fmt.Sprintf("json-v1-client-%s", time.Now().Format(time.RFC3339)),
		}
	)

	client, err := promtail.NewClient(jsonv1.NewJSONv1Exchanger(lokiAddress), labels)
	if err != nil {
		t.Fatalf("unable to initialize client: %s", err)
	}

	defer client.Close()

	for i := 0; i < requestsNumber; i++ {
		lvl, msg := generateLogMessage()
		client.Logf(lvl, msg)

		if i > 0 && i%10 == 0 {
			t.Logf("Requests performed: %d/%d", i, requestsNumber)
		}
		time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
	}

	t.Log("Done!")
}

func TestJsonV1Client_LogfWithLabels_External(t *testing.T) {
	t.Log("Starting JSON V1 client test")
	testsuite.PrintUsedEnvVarNames(t,
		testsuite.TestLokiAddressEnv,
		testsuite.TestRequestsNumberEnv,
	)
	testsuite.ValidateIsLokiReady(t)

	var (
		lokiAddress    = testsuite.TestLokiAddress
		requestsNumber = testsuite.TestRequestsNumber

		defaultLabels = map[string]string{
			"testName": fmt.Sprintf("json-v1-client-%s", time.Now().Format(time.RFC3339)),
		}
		additionalLabels = map[string]string{
			"instanceId": testsuite.GenerateRandString(6),
		}
	)

	client, err := promtail.NewClient(jsonv1.NewJSONv1Exchanger(lokiAddress), defaultLabels)
	if err != nil {
		t.Fatalf("unable to initialize client: %s", err)
	}

	defer client.Close()

	for i := 0; i < requestsNumber; i++ {
		lvl, msg := generateLogMessage()
		client.LogfWithLabels(lvl, additionalLabels, msg)

		if i > 0 && i%10 == 0 {
			t.Logf("Requests performed: %d/%d", i, requestsNumber)
		}

		time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
	}
	t.Log("Done!")
}

func generateLogMessage() (promtail.Level, string) {
	levels := []promtail.Level{
		promtail.Debug,
		promtail.Info,
		promtail.Warn,
		promtail.Error,
		promtail.Fatal,
		promtail.Panic,
	}

	return levels[rand.Intn(len(levels))], "it's a new log entry :)"
}
