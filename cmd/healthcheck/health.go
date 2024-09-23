package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Health struct {
	ConnectURL                 string
	WorkerID                   string
	UnhealthyStates            []string
	FailureThresholdPercentage int
	ConsideredContainers       []string
	Auth                       string
}

type ConnectorStatus struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	WorkerID string `json:"worker_id"`
	Tasks    []Task `json:"tasks"`
}

type Task struct {
	ID       int    `json:"id"`
	State    string `json:"state"`
	WorkerID string `json:"worker_id"`
	Trace    string `json:"trace,omitempty"`
}

type HealthResult struct {
	Failures         []Failure `json:"failures"`
	FailureStates    []string  `json:"failure_states"`
	FailureRate      float64   `json:"failure_rate"`
	FailureThreshold float64   `json:"failure_threshold"`
	Healthy          bool      `json:"healthy"`
}

type Failure struct {
	Type      string `json:"type"`
	Connector string `json:"connector"`
	State     string `json:"state,omitempty"`
	WorkerID  string `json:"worker_id,omitempty"`
	ID        int    `json:"id,omitempty"`
	Trace     string `json:"trace,omitempty"`
}

func NewHealth(connectURL, workerID string, unhealthyStates []string, auth string, failureThresholdPercentage int, consideredContainers []string) *Health {
	return &Health{
		ConnectURL:                 connectURL,
		WorkerID:                   workerID,
		UnhealthyStates:            unhealthyStates,
		FailureThresholdPercentage: failureThresholdPercentage,
		ConsideredContainers:       consideredContainers,
		Auth:                       auth,
	}
}

func (h *Health) GetHealthResult() (*HealthResult, error) {
	healthResult := &HealthResult{
		Failures:         []Failure{},
		FailureStates:    h.UnhealthyStates,
		FailureThreshold: float64(h.FailureThresholdPercentage) * 0.01,
	}

	connectorNames, err := h.GetConnectorNames()
	if err != nil {
		log.Printf("Error while attempting to get connector names. Error: %s\n", err)
		return nil, err
	}

	connectorStatuses, err := h.GetConnectorsHealth(connectorNames)
	if err != nil {
		log.Printf("Error while attempting to get connector statuses. Error: %s\n", err)
		return nil, err
	}

	h.HandleHealthCheck(connectorStatuses, healthResult)

	connectorCount := len(connectorNames)
	taskCount := 0
	for _, connector := range connectorStatuses {
		taskCount += len(connector.Tasks)
	}

	containerCount := 0
	if contains(h.ConsideredContainers, "connector") {
		containerCount += connectorCount
	}
	if contains(h.ConsideredContainers, "task") {
		containerCount += taskCount
	}

	failureCount := 0
	for _, failure := range healthResult.Failures {
		if contains(h.ConsideredContainers, failure.Type) {
			failureCount++
		}
	}

	if containerCount > 0 {
		healthResult.FailureRate = float64(failureCount) / float64(containerCount)
	} else {
		healthResult.FailureRate = 0.0
	}

	healthResult.Healthy = healthResult.FailureRate <= healthResult.FailureThreshold

	for _, failure := range healthResult.Failures {
		if failure.Type == "broker" {
			healthResult.Healthy = false
			break
		}
	}

	return healthResult, nil
}

func (h *Health) HandleHealthCheck(connectorStatuses []ConnectorStatus, healthResult *HealthResult) {
	connectorsOnThisWorker := false
	for _, connector := range connectorStatuses {
		if h.isOnThisWorker(connector.WorkerID) && contains(h.ConsideredContainers, "connector") {
			connectorsOnThisWorker = true
			if h.isInUnhealthyState(connector.State) {
				log.Printf("Connector '%s' is unhealthy in failure state: %s\n", connector.Name, connector.State)
				healthResult.Failures = append(healthResult.Failures, Failure{
					Type:      "connector",
					Connector: connector.Name,
					State:     connector.State,
					WorkerID:  connector.WorkerID,
				})
			} else {
				log.Printf("Connector '%s' is healthy in state: %s\n", connector.Name, connector.State)
			}
		}
		h.handleTaskHealthCheck(connector, healthResult)
	}
	if !connectorsOnThisWorker && len(connectorStatuses) > 0 {
		log.Printf("No connectors found on worker '%s'. Checking broker health\n", h.WorkerID)
		h.handleBrokerHealthCheck(healthResult, connectorStatuses[0].Name)
	}
}

func (h *Health) handleBrokerHealthCheck(healthResult *HealthResult, connectorName string) {
	_, err := h.getConnectorDetails(connectorName)
	if err != nil {
		log.Printf("Error while attempting to get details for %s. Assuming unhealthy. Error: %s\n", connectorName, err)
		healthResult.Failures = append(healthResult.Failures, Failure{
			Type:      "broker",
			Connector: connectorName,
		})
	}
}

func (h *Health) handleTaskHealthCheck(connector ConnectorStatus, healthResult *HealthResult) {
	if contains(h.ConsideredContainers, "task") {
		for _, task := range connector.Tasks {
			if h.isOnThisWorker(task.WorkerID) {
				if h.isInUnhealthyState(task.State) {
					log.Printf("Connector '%s' task '%s' is unhealthy in failure state: %s\n", connector.Name, task.ID, task.State)
					healthResult.Failures = append(healthResult.Failures, Failure{
						Type:      "task",
						Connector: connector.Name,
						ID:        task.ID,
						State:     task.State,
						WorkerID:  task.WorkerID,
						Trace:     task.Trace,
					})
				} else {
					log.Printf("Connector '%s' task '%s' is healthy in state: %s\n", connector.Name, task.ID, task.State)
				}
			}
		}
	}
}

func (h *Health) GetConnectorsHealth(connectorNames []string) ([]ConnectorStatus, error) {
	statuses := []ConnectorStatus{}
	for _, connectorName := range connectorNames {
		status, err := h.GetConnectorHealth(connectorName)
		if err != nil {
			log.Printf("Error while attempting to get status for %s. Error: %s\n", connectorName, err)
			return nil, err
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (h *Health) GetConnectorHealth(connectorName string) (ConnectorStatus, error) {
	connectorStatus := ConnectorStatus{}
	err := h.getJSON(fmt.Sprintf("%s/connectors/%s/status", h.ConnectURL, connectorName), &connectorStatus)
	if err != nil {
		return ConnectorStatus{}, err
	}
	return connectorStatus, nil
}

func (h *Health) GetConnectorNames() ([]string, error) {
	var connectorNames []string
	err := h.getJSON(fmt.Sprintf("%s/connectors", h.ConnectURL), &connectorNames)
	if err != nil {
		log.Printf("Error while attempting to get connector names. Error: %s\n", err)
		return nil, err
	}
	return connectorNames, nil
}

func (h *Health) getConnectorDetails(connectorName string) (interface{}, error) {
	var response interface{}
	err := h.getJSON(fmt.Sprintf("%s/connectors/%s", h.ConnectURL, connectorName), &response)
	if err != nil {
		log.Printf("Error while attempting to get details for %s. Error: %s\n", connectorName, err)
		return nil, err
	}
	return response, nil
}

func (h *Health) isInUnhealthyState(state string) bool {
	state = strings.ToUpper(strings.TrimSpace(state))
	for _, unhealthyState := range h.UnhealthyStates {
		log.Printf("Checking if state '%s' is in unhealthy states: %v\n", state, h.UnhealthyStates)
		if state == strings.ToUpper(strings.TrimSpace(unhealthyState)) {
			log.Printf("State '%s' is in unhealthy states: %v\n", state, h.UnhealthyStates)
			return true
		}
	}
	return false
}

func (h *Health) isOnThisWorker(responseWorkerID string) bool {
	if h.WorkerID != "" {
		return strings.ToLower(responseWorkerID) == strings.ToLower(h.WorkerID)
	}
	return true
}

func (h *Health) getJSON(url string, target interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error while attempting to create request. Error: %s\n", err)
		return err
	}
	if h.Auth != "" {
		req.SetBasicAuth(h.Auth, "")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error while attempting to make request. Error: %s\n", err)
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(target)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
