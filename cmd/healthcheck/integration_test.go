package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type testScenario struct {
	name           string
	env            map[string]string
	mockName       string
	endpoint       string
	expectedStatus int
	expectedFile   string
	checkSubstring bool
}

var scenarios = []testScenario{
	{
		name:           "0-healthcheck-server-healthy",
		env:            map[string]string{},
		mockName:       "0-healthcheck-server-healthy",
		endpoint:       "/ping",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/0-healthcheck-server-healthy.json",
	},
	{
		name:           "1-healthy",
		env:            map[string]string{"HEALTHCHECK_UNHEALTHY_STATES": "FAILED"},
		mockName:       "1-healthy",
		endpoint:       "/",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/1-healthy.json",
	},
	{
		name:           "2-unhealthy",
		env:            map[string]string{"HEALTHCHECK_UNHEALTHY_STATES": "FAILED"},
		mockName:       "2-unhealthy",
		endpoint:       "/",
		expectedStatus: 503,
		expectedFile:   "tests/data/expected/2-unhealthy.json",
	},
	{
		name:           "3-unhealthy-error",
		env:            map[string]string{"HEALTHCHECK_CONNECT_URL": "http://unknown-hostname:8083"},
		mockName:       "3-unhealthy-error",
		endpoint:       "/",
		expectedStatus: 503,
		expectedFile:   "tests/data/expected/3-unhealthy-error.json",
		checkSubstring: true,
	},
	{
		name:           "4-healthy-worker-id-correct",
		env:            map[string]string{"HEALTHCHECK_CONNECT_WORKER_ID": "kafka-connect:8083"},
		mockName:       "4-healthy-worker-id-correct",
		endpoint:       "/",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/4-healthy-worker-id-correct.json",
	},
	{
		name:           "5-healthy-worker-id-unused",
		env:            map[string]string{"HEALTHCHECK_CONNECT_WORKER_ID": "kafka-connect:8083"},
		mockName:       "5-healthy-worker-id-unused",
		endpoint:       "/",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/5-healthy-worker-id-unused.json",
	},
	{
		name:           "6-healthy-worker-id-with-other-workers-failing",
		env:            map[string]string{"HEALTHCHECK_CONNECT_WORKER_ID": "kafka-connect:8083"},
		mockName:       "6-healthy-worker-id-with-other-workers-failing",
		endpoint:       "/",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/6-healthy-worker-id-with-other-workers-failing.json",
	},
	{
		name:           "7-unhealthy-worker-id-with-other-workers-healthy",
		env:            map[string]string{"HEALTHCHECK_CONNECT_WORKER_ID": "kafka-connect:8083"},
		mockName:       "7-unhealthy-worker-id-with-other-workers-healthy",
		endpoint:       "/",
		expectedStatus: 503,
		expectedFile:   "tests/data/expected/7-unhealthy-worker-id-with-other-workers-healthy.json",
	},
	{
		name:           "8-healthy-multiple-tasks",
		env:            map[string]string{"HEALTHCHECK_CONNECT_WORKER_ID": "my.worker.name:8083"},
		mockName:       "8-healthy-multiple-tasks",
		endpoint:       "/",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/8-healthy-multiple-tasks.json",
	},
	{
		name:           "9-healthy-multiple-connectors",
		env:            map[string]string{"HEALTHCHECK_CONNECT_WORKER_ID": "my.worker.name:8083"},
		mockName:       "9-healthy-multiple-connectors",
		endpoint:       "/",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/9-healthy-multiple-connectors.json",
	},
	{
		name:           "10-unhealthy-multiple-connectors",
		env:            map[string]string{"HEALTHCHECK_CONNECT_WORKER_ID": "my.worker.name:8083"},
		mockName:       "10-unhealthy-multiple-connectors",
		endpoint:       "/",
		expectedStatus: 503,
		expectedFile:   "tests/data/expected/10-unhealthy-multiple-connectors.json",
	},
	{
		name:           "11-healthy-no-connectors",
		env:            map[string]string{"HEALTHCHECK_CONNECT_WORKER_ID": "my.worker.name:8083"},
		mockName:       "11-healthy-no-connectors",
		endpoint:       "/",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/11-healthy-no-connectors.json",
	},
	{
		name:           "12-unhealthy-task-with-trace",
		env:            map[string]string{"HEALTHCHECK_CONNECT_WORKER_ID": "my.worker.name:8083"},
		mockName:       "12-unhealthy-task-with-trace",
		endpoint:       "/",
		expectedStatus: 503,
		expectedFile:   "tests/data/expected/12-unhealthy-task-with-trace.json",
	},
	{
		name:           "13-unhealthy-broker-connection",
		env:            map[string]string{"HEALTHCHECK_CONNECT_WORKER_ID": "unhealthy.worker:8083"},
		mockName:       "13-unhealthy-broker-connection",
		endpoint:       "/",
		expectedStatus: 503,
		expectedFile:   "tests/data/expected/13-unhealthy-broker-connection.json",
	},
	{
		name:           "14-basic-auth",
		env:            map[string]string{"HEALTHCHECK_BASIC_AUTH": "username:password"},
		mockName:       "14-basic-auth",
		endpoint:       "/",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/14-basic-auth.json",
	},
	{
		name:           "15-unhealthy-threshold",
		env:            map[string]string{"HEALTHCHECK_FAILURE_THRESHOLD_PERCENTAGE": "10"},
		mockName:       "15-unhealthy-threshold",
		endpoint:       "/",
		expectedStatus: 503,
		expectedFile:   "tests/data/expected/15-unhealthy-threshold.json",
	},
	{
		name:           "16-healthy-threshold",
		env:            map[string]string{"HEALTHCHECK_FAILURE_THRESHOLD_PERCENTAGE": "50"},
		mockName:       "16-healthy-threshold",
		endpoint:       "/",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/16-healthy-threshold.json",
	},
	{
		name:           "17-healthy-container-connector",
		env:            map[string]string{"HEALTHCHECK_CONSIDERED_CONTAINERS": "CONNECTOR"},
		mockName:       "17-healthy-container-connector",
		endpoint:       "/",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/17-healthy-container-connector.json",
	},
	{
		name:           "18-healthy-container-task",
		env:            map[string]string{"HEALTHCHECK_CONSIDERED_CONTAINERS": "TASK"},
		mockName:       "18-healthy-container-task",
		endpoint:       "/",
		expectedStatus: 200,
		expectedFile:   "tests/data/expected/18-healthy-container-task.json",
	},
	{
		name:           "19-unhealthy-container-task",
		env:            map[string]string{"HEALTHCHECK_CONSIDERED_CONTAINERS": "TASK"},
		mockName:       "19-unhealthy-container-task",
		endpoint:       "/",
		expectedStatus: 503,
		expectedFile:   "tests/data/expected/19-unhealthy-container-task.json",
	},
}

func TestIntegration(t *testing.T) {
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Start mock Kafka Connect server
			var mockServer *http.Server
			var kafkaConnectURL string

			if scenario.env["HEALTHCHECK_CONNECT_URL"] == "" && scenario.mockName != "" {
				port, err := getFreePort()
				if err != nil {
					t.Fatalf("Failed to get free port: %v", err)
				}
				kafkaConnectURL = fmt.Sprintf("http://localhost:%d", port)
				mockServer = startMockServerOnPort(port, scenario.mockName)
				defer mockServer.Shutdown(context.Background())

				scenario.env["HEALTHCHECK_CONNECT_URL"] = kafkaConnectURL
			}

			// Set environment variables
			for key, value := range scenario.env {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Get a free port for healthcheck server
			healthcheckPort, err := getFreePort()
			if err != nil {
				t.Fatalf("Failed to get free port: %v", err)
			}
			os.Setenv("HEALTHCHECK_PORT", fmt.Sprintf("%d", healthcheckPort))
			defer os.Unsetenv("HEALTHCHECK_PORT")

			healthcheckURL := fmt.Sprintf("http://localhost:%d", healthcheckPort)

			// Start healthcheck server
			healthcheckServer := startHealthcheckServer(t, healthcheckPort)
			defer healthcheckServer.Shutdown(context.Background())

			// Wait for servers to be ready
			time.Sleep(500 * time.Millisecond)

			// Make request
			resp, err := http.Get(healthcheckURL + scenario.endpoint)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != scenario.expectedStatus {
				t.Errorf("Expected status %d, got %d", scenario.expectedStatus, resp.StatusCode)
			}

			// Check response body
			if scenario.expectedFile != "" {
				expected, err := os.ReadFile("../../" + scenario.expectedFile)
				if err != nil {
					t.Fatalf("Failed to read expected file: %v", err)
				}

				actual, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				if scenario.checkSubstring {
					// For error scenarios, just check key fields
					var expectedMap, actualMap map[string]interface{}
					json.Unmarshal(expected, &expectedMap)
					json.Unmarshal(actual, &actualMap)

					if actualMap["healthy"] != expectedMap["healthy"] {
						t.Errorf("healthy mismatch")
					}
				} else {
					compareJSON(t, expected, actual)
				}
			}
		})
	}
}

func startMockServerOnPort(port int, mockName string) *http.Server {
	handler := &MockServerRequestHandler{MockName: mockName}
	server := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: handler,
	}
	go server.ListenAndServe()
	return server
}

func startHealthcheckServer(t *testing.T, port int) *http.Server {
	t.Helper()

	// Get configuration from environment
	testConnectURL := defaultString("HEALTHCHECK_CONNECT_URL", "http://localhost:8083")
	testConnectWorkerID := defaultString("HEALTHCHECK_CONNECT_WORKER_ID", "")
	testUnhealthyStates := defaultString("HEALTHCHECK_UNHEALTHY_STATES", "FAILED")
	testConsideredContainers := defaultString("HEALTHCHECK_CONSIDERED_CONTAINERS", "CONNECTOR,TASK")
	testFailureThresholdPercentage := defaultInt("HEALTHCHECK_FAILURE_THRESHOLD_PERCENTAGE", 0)
	testBasicAuth := defaultString("HEALTHCHECK_BASIC_AUTH", "")

	healthObject := NewHealth(testConnectURL, testConnectWorkerID, strings.Split(testUnhealthyStates, ","), testBasicAuth, testFailureThresholdPercentage, strings.Split(strings.ToLower(testConsideredContainers), ","))
	handler := &RequestHandler{health: *healthObject}

	server := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: handler,
	}

	go server.ListenAndServe()

	return server
}

func compareJSON(t *testing.T, expected, actual []byte) {
	t.Helper()
	var expectedMap, actualMap map[string]interface{}
	if err := json.Unmarshal(expected, &expectedMap); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v", err)
	}
	if err := json.Unmarshal(actual, &actualMap); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %v", err)
	}

	expectedJSON, _ := json.MarshalIndent(expectedMap, "", "  ")
	actualJSON, _ := json.MarshalIndent(actualMap, "", "  ")

	if string(expectedJSON) != string(actualJSON) {
		t.Errorf("JSON mismatch.\nExpected:\n%s\n\nActual:\n%s", expectedJSON, actualJSON)
	}
}
