package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

// Mocks Kafka Connect REST API responses for integration tests
type MockServerRequestHandler struct {
	MockName string
}

func (h *MockServerRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	statusCode := 503
	if strings.Contains(h.MockName, "unhealthy") {
		statusCode = 200
	}

	if r.Method == "GET" {
		if strings.Contains(h.MockName, "auth") {
			if !strings.Contains(r.Header.Get("Authorization"), "Basic") {
				w.WriteHeader(401)
				w.Write([]byte("{}"))
				return
			}
		}

		if r.URL.Path == "/connectors" {
			filePath := fmt.Sprintf("./tests/data/mocks/%s-connectors.json", h.MockName)
			log.Println("filePath:", filePath)
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				log.Println(err.Error())

				w.WriteHeader(500)
				return
			}
			w.WriteHeader(200)
			w.Write(data)
		} else {
			splitPath := strings.Split(r.URL.Path, "/")
			if len(splitPath) >= 4 && splitPath[3] == "status" {
				connectorName := splitPath[2]
				filePath := fmt.Sprintf("./tests/data/mocks/%s-connector-%s.json", h.MockName, connectorName)
				log.Println("filePath:", filePath)

				data, err := ioutil.ReadFile(filePath)
				if err != nil {
					log.Println(err.Error())
					w.WriteHeader(statusCode)
					return
				}
				w.WriteHeader(statusCode)
				w.Write(data)
			} else if len(splitPath) >= 3 {
				connectorName := splitPath[2]
				filePath := fmt.Sprintf("./tests/data/mocks/%s-connector-%s.json", h.MockName, connectorName)
				log.Println("filePath:", filePath)

				data, err := ioutil.ReadFile(filePath)
				if err != nil {
					log.Println(err.Error())
					w.WriteHeader(statusCode)
					return
				}
				w.WriteHeader(statusCode)
				w.Write(data)
			} else {
				connectorName := splitPath[2]
				detailsStatusCode := 503
				if strings.Contains(h.MockName, "unhealthy-broker") {
					detailsStatusCode = 200
				}
				filePath := fmt.Sprintf("./tests/data/mocks/healthy-connector-details.json", h.MockName, connectorName)
				log.Println("filePath:", filePath)

				data, err := ioutil.ReadFile(filePath)
				if err != nil {
					log.Println(err.Error())

					w.WriteHeader(detailsStatusCode)
					return
				}
				w.WriteHeader(detailsStatusCode)
				w.Write(data)
			}
		}
	} else {
		w.WriteHeader(404)
	}
}

func getFreePort() (int, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func startMockServer(port int, mockName string) {
	handler := &MockServerRequestHandler{MockName: mockName}
	http.Handle("/", handler)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	fmt.Printf("\nMock Kafka Connect server running on port: %d\n", port)
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("mockName is required")
	}
	mockName := strings.TrimSpace(os.Args[1])
	log.Println("mockName:", mockName)

	startMockServer(8083, mockName)
	select {}
}
