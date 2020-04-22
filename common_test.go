package promtail

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

const (
	TestLokiAddressEnv      = "TEST_LOKI_ADDRESS"
	TestLokiAddressFallback = "127.0.0.1:3100"

	TestRequestsNumberEnv      = "TEST_REQUESTS_NUMBER"
	TestRequestsNumberFallback = 20
)

// Overwrite test values ENV
var (
	TestLokiAddress    string
	TestRequestsNumber int
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func init() {
	var err error

	//
	// Resolve configurations
	//
	TestLokiAddress = getEnvStringWithFallback(TestLokiAddressEnv, TestLokiAddressFallback)
	TestRequestsNumber, err = getEnvIntWithFallback(TestRequestsNumberEnv, TestRequestsNumberFallback)
	if err != nil {
		log.Fatalf("env. var. [%s] required to be integer value", TestRequestsNumberEnv)
	}

	//
	// Print running configurations
	//
	log.Printf("Resolved test configurations: [could be changed via env. variables]")
	log.Printf("Loki Test server address: %s [%s]", TestLokiAddress, TestLokiAddressEnv)
	log.Printf("Number of requests to external Loki server:  %d [%s]", TestRequestsNumber, TestRequestsNumberEnv)
	log.Print()
}

func validateIsLokiReady(t *testing.T) {
	resp, err := http.Get(fmt.Sprintf("http://%s/ready", TestLokiAddress))
	if err != nil {
		t.Fatalf("unable to connect to Loki server, %s", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if !(199 < resp.StatusCode && resp.StatusCode < 300) {
		responseMessage, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			defer t.Errorf("also, failed to read Loki's response message :(")
		}

		t.Fatalf("Loki server is not ready, status code: %s, response: ", string(responseMessage))
	}
}

func printUsedEnvVarNames(t *testing.T, envVars ...string) {
	t.Log("using configurations from env variables:")
	for i := range envVars {
		t.Logf(" - %s", envVars[i])
	}
}

func generateLogMessage() (Level, string) {
	levels := []Level{
		Debug,
		Info,
		Warn,
		Error,
		Fatal,
		Panic,
	}

	return levels[rand.Intn(len(levels))], "it's a new log entry :)"
}

func generateRandString(length uint) string {
	var (
		letterBytes = "abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		b           = make([]byte, length)
	)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func getEnvStringWithFallback(varName, fallbackValue string) string {
	if resolved := os.Getenv(varName); resolved != "" {
		return resolved
	}
	return fallbackValue
}

func getEnvIntWithFallback(varName string, fallbackValue int) (int, error) {
	if resolved := os.Getenv(varName); resolved != "" {
		return strconv.Atoi(resolved)
	}
	return fallbackValue, nil
}
