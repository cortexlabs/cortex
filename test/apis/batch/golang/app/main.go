package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/DataDog/datadog-go/statsd"
)

func main() {
	var (
		url           string
		statsdAddress string
		concurrency   int
		numRequests   int
		id            string
		port          string
	)
	flag.StringVar(&url, "url", "", "url")
	flag.StringVar(&id, "id", "", "id")
	flag.StringVar(&statsdAddress, "statsdAddress", "prometheus-statsd-exporter.default:9125", "statsd-address")
	flag.StringVar(&port, "port", "", "port")
	flag.IntVar(&concurrency, "concurrency", 1, "concurrency")
	flag.IntVar(&numRequests, "num-requests", 1, "num requests")

	flag.Parse()
	metrics, _ := statsd.New(statsdAddress)

	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "ok")
	}

	mainHandler := func(w http.ResponseWriter, req *http.Request) {
		runConstantInFlightIteration(url, id, metrics, concurrency, 0*time.Second, numRequests)
		io.WriteString(w, "ok")
	}

	http.HandleFunc("/healthz", helloHandler)
	http.HandleFunc("/on-job-complete", helloHandler)
	http.HandleFunc("/", mainHandler)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func runConstantInFlightIteration(url string, id string, statsdClient statsd.ClientInterface, concurrent int, requestDelay time.Duration, requestsPerThread int) {
	doneChans := make([]chan struct{}, concurrent)
	for i := range doneChans {
		doneChans[i] = make(chan struct{})
	}

	for i := range doneChans {
		doneChan := doneChans[i]

		go func() {
			makeRequestLoop(url, id, statsdClient, requestDelay, requestsPerThread)
			doneChan <- struct{}{}
		}()
	}

	for _, doneChan := range doneChans {
		<-doneChan
	}
}

func makeRequestLoop(url string, id string, statsdClient statsd.ClientInterface, requestDelay time.Duration, requestsPerThread int) {
	var i int
	isFirstIteration := true
	tags := []string{
		"id:" + id,
	}

	for true {
		if !isFirstIteration && requestDelay != 0 {
			time.Sleep(requestDelay)
		}
		isFirstIteration = false

		if requestsPerThread > 0 {
			if i >= requestsPerThread {
				return
			}
			i++
		}

		code, err := makeRequest(url)
		if err != nil {
			tags := append(tags, "code:err")
			statsdClient.Incr("load_test", append(tags, "code:err"), 1.0)
			log.Print(err.Error())
			continue
		}

		statsdClient.Incr("load_test", append(tags, fmt.Sprintf("code:%d", code)), 1.0)
	}
}

// statsdClient statsd.ClientInterface
func makeRequest(url string) (int, error) {
	log.Print("makeRequest" + url)
	response, err := http.Post(url, "application/json", bytes.NewBufferString("\"\""))
	if err != nil {
		return 0, err
	}

	body, bodyReadErr := ioutil.ReadAll(response.Body)
	response.Body.Close()

	if response.StatusCode != 200 {
		if bodyReadErr == nil {
			fmt.Printf("\nstatus code: %d; body: %s\n", response.StatusCode, string(body))
		} else {
			fmt.Printf("\nstatus code: %d; error reading body: %s\n", response.StatusCode, bodyReadErr.Error())
		}
	}

	return response.StatusCode, nil
}
