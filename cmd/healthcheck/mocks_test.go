package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

// MockServerRequestHandler mocks Kafka Connect REST API responses for integration tests
type MockServerRequestHandler struct {
	MockName string
}

func (h *MockServerRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Always return 200 for status endpoints - the status is in the JSON, not the HTTP code
	statusCode := 200

	if r.Method == "GET" {
		if strings.Contains(h.MockName, "auth") {
			if !strings.Contains(r.Header.Get("Authorization"), "Basic") {
				h.response(w, 401, "{}")
				return
			}
		}

		if r.URL.Path == "/connectors" {
			filePath := fmt.Sprintf("../../tests/data/mocks/%s-connectors.json", h.MockName)
			data, err := os.ReadFile(filePath)
			if err != nil {
				h.response(w, 500, "")
				return
			}
			h.response(w, 200, string(data))
		} else {
			splitPath := strings.Split(r.URL.Path, "/")
			if len(splitPath) >= 4 && splitPath[3] == "status" {
				connectorName := splitPath[2]
				filePath := fmt.Sprintf("../../tests/data/mocks/%s-connector-%s.json", h.MockName, connectorName)
				data, err := os.ReadFile(filePath)
				if err != nil {
					h.response(w, 500, "")
					return
				}
				h.response(w, statusCode, string(data))
			} else {
				detailsStatusCode := 200
				if strings.Contains(h.MockName, "unhealthy-broker") {
					detailsStatusCode = 503
				}
				filePath := "../../tests/data/mocks/healthy-connector-details.json"
				data, err := os.ReadFile(filePath)
				if err != nil {
					h.response(w, 500, "")
					return
				}
				h.response(w, detailsStatusCode, string(data))
			}
		}
	}
}

func (h *MockServerRequestHandler) response(w http.ResponseWriter, statusCode int, payload string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write([]byte(payload))
}

func getFreePort() (int, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func startMockServer(port int, mockName string) *http.Server {
	handler := &MockServerRequestHandler{MockName: mockName}
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}
	go server.ListenAndServe()
	return server
}
