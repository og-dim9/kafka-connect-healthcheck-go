package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

func main() {

	parser.Parse(os.Args[1:])

	log.Printf("Initializing healthcheck server...")

	healthObject := NewHealth(*connectURL, *connectWorkerID, strings.Split(*unhealthyStates, ","), *basicAuth, *failureThresholdPercentage, strings.Split(strings.ToLower(*consideredContainers), ","))

	handler := &RequestHandler{health: *healthObject}
	http.HandleFunc("/", handler.ServeHTTP)

	go func() {
		log.Printf("Healthcheck server started at: http://localhost:%d", *healthcheckPort)
		log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(*healthcheckPort), nil))
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, os.Kill)
	<-stop
	log.Printf("Shutting down healthcheck server...")
}
