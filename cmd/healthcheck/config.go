package main

import (
	"flag"
	"os"
	"strconv"
)

var (
	defaultHealthcheckPort            = defaultInt("HEALTHCHECK_PORT", 18083)
	defaultConnectURL                 = defaultString("HEALTHCHECK_CONNECT_URL", "http://localhost:8083")
	defaultConnectWorkerID            = defaultString("HEALTHCHECK_CONNECT_WORKER_ID", "")
	defaultUnhealthyStates            = defaultString("HEALTHCHECK_UNHEALTHY_STATES", "FAILED")
	defaultConsideredContainers       = defaultString("HEALTHCHECK_CONSIDERED_CONTAINERS", "CONNECTOR,TASK")
	defaultFailureThresholdPercentage = defaultInt("HEALTHCHECK_FAILURE_THRESHOLD_PERCENTAGE", 0.0)
	defaultBasicAuth                  = defaultString("HEALTHCHECK_BASIC_AUTH", "")

	parser = flag.NewFlagSet("healthcheck", flag.ExitOnError)

	healthcheckPort            = parser.Int("port", defaultHealthcheckPort, "The port for the healthcheck HTTP server.")
	connectURL                 = parser.String("connect-url", defaultConnectURL, "The Kafka Connect REST API URL that the health check will be run against.")
	connectWorkerID            = parser.String("connect-worker-id", defaultConnectWorkerID, "The Kafka Connect REST API URL that the health check will be run against.")
	unhealthyStates            = parser.String("unhealthy-states", defaultUnhealthyStates, "A comma separated lists of connector and task states to be marked as unhealthy. Default: FAILED.")
	consideredContainers       = parser.String("considered-containers", defaultConsideredContainers, "A comma separated lists of container types to consider for failure calculations. Default: CONNECTOR,TASK.")
	failureThresholdPercentage = parser.Int("failure-threshold-percentage", defaultFailureThresholdPercentage, "A number between 1 and 100. If set, this is the percentage of connectors that must fail for the healthcheck to fail.")
	basicAuth                  = parser.String("basic-auth", defaultBasicAuth, "Colon-separated credentials for basic HTTP authentication. Default: empty.")
)

func defaultString(name string, value string) string {
	if name == "" {
		return value
	}
	res := os.Getenv(name)
	if res == "" {
		return value
	}

	return res
}
func defaultInt(name string, value int) int {
	if name == "" {
		return value
	}
	res := os.Getenv(name)
	if res == "" {
		return value
	}

	i, err := strconv.Atoi(res)
	if err != nil {
		return value
	}

	return i
}
