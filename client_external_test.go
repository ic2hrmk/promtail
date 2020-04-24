// +build external

package promtail

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestJsonV1Client_Ping(t *testing.T) {
	t.Log("Starting JSON V1 client test")
	printUsedEnvVarNames(t,
		TestLokiAddressEnv,
	)
	validateIsLokiReady(t)

	var (
		lokiAddress = TestLokiAddress
	)

	client, err := NewJSONv1Client(lokiAddress, nil)
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
	printUsedEnvVarNames(t,
		TestLokiAddressEnv,
		TestRequestsNumberEnv,
	)
	validateIsLokiReady(t)

	var (
		lokiAddress    = TestLokiAddress
		requestsNumber = TestRequestsNumber

		labels = map[string]string{
			"testName": fmt.Sprintf("json-v1-client-%s", time.Now().Format(time.RFC3339)),
		}
	)

	client, err := NewJSONv1Client(lokiAddress, labels)
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
	printUsedEnvVarNames(t,
		TestLokiAddressEnv,
		TestRequestsNumberEnv,
	)
	validateIsLokiReady(t)

	var (
		lokiAddress    = TestLokiAddress
		requestsNumber = TestRequestsNumber

		defaultLabels = map[string]string{
			"testName": fmt.Sprintf("json-v1-client-%s", time.Now().Format(time.RFC3339)),
		}
		additionalLabels = map[string]string{
			"instanceId": generateRandString(6),
		}
	)

	client, err := NewJSONv1Client(lokiAddress, defaultLabels)
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
