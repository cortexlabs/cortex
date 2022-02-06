/*
Copyright 2022 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"go.uber.org/atomic"
)

// usage: go run load.go <url> <sample.json OR sample json string>

// configuration options (either set _numConcurrent > 0 or _requestInterval > 0, and configure the corresponding section)
const (
	// constant in-flight requests
	_numConcurrent        = 3
	_requestDelay         = 0 * time.Millisecond
	_numRequestsPerThread = 0 // 0 means loop infinitely
	_numMainLoops         = 1 // only relevant if _numRequestsPerThread > 0

	// constant requests per second
	_requestInterval        = 0 * time.Millisecond
	_numRequests     uint64 = 0 // 0 means loop infinitely
	_maxInFlight            = 5

	// other options
	_printSuccessDots = true
	_printBody        = false
	_printHTTPErrors  = true
	_printGoErrors    = true
)

type Counter struct {
	sync.Mutex
	count int64
}

var (
	_requestCount = atomic.Uint64{}
	_successCount = atomic.Uint64{}
	_httpErrCount = atomic.Uint64{} // HTTP error response codes
	_goErrCount   = atomic.Uint64{} // actual errors in go (includes "connection reset by peer")
)

var _client = &http.Client{
	Timeout: 0, // no timeout
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func main() {
	if _numConcurrent > 0 && _requestInterval > 0 {
		fmt.Println("error: you must set either _numConcurrent or _requestInterval > 0, but not both")
		os.Exit(1)
	}

	if _numConcurrent == 0 && _requestInterval == 0 {
		fmt.Println("error: you must set either _numConcurrent or _requestInterval > 0")
		os.Exit(1)
	}

	url, jsonPathOrString := mustExtractArgs()
	var jsonBytes []byte
	if jsonPathOrString == "" {
		jsonBytes = nil
	} else if strings.HasPrefix(jsonPathOrString, "{") {
		jsonBytes = []byte(jsonPathOrString)
	} else {
		jsonBytes = mustReadJSONBytes(jsonPathOrString)
	}

	if _numConcurrent > 0 {
		runConstantInFlight(url, jsonBytes)
	}

	if _requestInterval > 0 {
		runConstantRequestsPerSecond(url, jsonBytes)
	}
}

func runConstantRequestsPerSecond(url string, jsonBytes []byte) {
	inFlightCount := Counter{}
	ticker := time.NewTicker(_requestInterval)
	done := make(chan bool)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		done <- true
	}()

	start := time.Now()

FOR_LOOP:
	for {
		select {
		case <-done:
			break FOR_LOOP
		case <-ticker.C:
			go runConstantRequestsPerSecondIteration(url, jsonBytes, &inFlightCount, done)
		}
	}

	elapsed := time.Since(start)
	requestRate := float64(_requestCount.Load()) / elapsed.Seconds()
	fmt.Printf("\nelapsed time: %s | %d requests @ %f req/s | %d succeeded | %d http errors | %d go errors\n", elapsed, _requestCount.Load(), requestRate, _successCount.Load(), _httpErrCount.Load(), _goErrCount.Load())
}

func runConstantRequestsPerSecondIteration(url string, jsonBytes []byte, inFlightCount *Counter, done chan bool) {
	if _maxInFlight > 0 {
		inFlightCount.Lock()
		if inFlightCount.count >= _maxInFlight {
			inFlightCount.Unlock()
			fmt.Printf("\nreached max in-flight (%d)\n", _maxInFlight)
			return
		}
		inFlightCount.count++
		inFlightCount.Unlock()
	}

	makeRequest(url, jsonBytes)

	if _numRequests > 0 && _requestCount.Load() >= _numRequests {
		done <- true
	}

	if _maxInFlight > 0 {
		inFlightCount.Lock()
		inFlightCount.count--
		inFlightCount.Unlock()
	}
}

func runConstantInFlight(url string, jsonBytes []byte) {
	if _numRequestsPerThread > 0 {
		fmt.Printf("spawning %d threads, %d requests each, %s delay on each\n", _numConcurrent, _numRequestsPerThread, _requestDelay.String())
	} else {
		fmt.Printf("spawning %d infinite threads, %s delay on each\n", _numConcurrent, _requestDelay.String())
	}

	var summedRequestCount uint64
	var summedSuccessCount uint64
	var summedHTTPErrCount uint64
	var summedGoErrCount uint64

	start := time.Now()
	loopNum := 1
	for {
		wasKilled := runConstantInFlightIteration(url, jsonBytes, loopNum)

		summedRequestCount += _requestCount.Load()
		summedSuccessCount += _successCount.Load()
		summedHTTPErrCount += _httpErrCount.Load()
		summedGoErrCount += _goErrCount.Load()

		_requestCount.Store(0)
		_successCount.Store(0)
		_httpErrCount.Store(0)
		_goErrCount.Store(0)

		if loopNum >= _numMainLoops || wasKilled {
			break
		}
		loopNum++
	}

	if _numMainLoops > 1 {
		elapsed := time.Since(start)
		requestRate := float64(summedRequestCount) / elapsed.Seconds()
		fmt.Printf("\ntotal elapsed time: %s | %d requests @ %f req/s | %d succeeded | %d http errors | %d go errors\n", elapsed, summedRequestCount, requestRate, summedSuccessCount, summedHTTPErrCount, summedGoErrCount)
	}
}

func runConstantInFlightIteration(url string, jsonBytes []byte, loopNum int) bool {
	start := time.Now()

	wasKilled := false
	killed := make(chan bool)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		killed <- true
	}()

	doneChans := make([]chan struct{}, _numConcurrent)
	for i := range doneChans {
		doneChans[i] = make(chan struct{})
	}

	for i := range doneChans {
		doneChan := doneChans[i]

		go func() {
			makeRequestLoop(url, jsonBytes)
			doneChan <- struct{}{}
		}()
	}

LOOP:
	for _, doneChan := range doneChans {
		select {
		case <-killed:
			wasKilled = true
			break LOOP
		case <-doneChan:
			continue
		}
	}

	elapsed := time.Now().Sub(start)
	requestRate := float64(_requestCount.Load()) / elapsed.Seconds()
	fmt.Printf("\nelapsed time: %s | %d requests @ %f req/s | %d succeeded | %d http errors | %d go errors\n", elapsed, _requestCount.Load(), requestRate, _successCount.Load(), _httpErrCount.Load(), _goErrCount.Load())

	return wasKilled
}

func makeRequestLoop(url string, jsonBytes []byte) {
	var i int
	isFirstIteration := true
	for true {
		if !isFirstIteration && _requestDelay != 0 {
			time.Sleep(_requestDelay)
		}
		isFirstIteration = false

		if _numRequestsPerThread > 0 {
			if i >= _numRequestsPerThread {
				return
			}
			i++
		}

		makeRequest(url, jsonBytes)
	}
}

func makeRequest(url string, jsonBytes []byte) {
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		fmt.Print("\n" + debug.Sppg(err))
		return
	}

	request.Header.Set("Content-Type", "application/json")

	_requestCount.Inc()

	response, err := _client.Do(request)
	if err != nil {
		_goErrCount.Inc()
		if _printGoErrors {
			fmt.Print("\n" + debug.Sppg(err))
		}
		return
	}

	body, bodyReadErr := ioutil.ReadAll(response.Body)
	response.Body.Close()

	if response.StatusCode == 200 {
		_successCount.Inc()
	} else {
		_httpErrCount.Inc()
		if _printHTTPErrors {
			if bodyReadErr == nil {
				fmt.Printf("\nstatus code: %d; body: %s\n", response.StatusCode, string(body))
			} else {
				fmt.Printf("\nstatus code: %d; error reading body: %s\n", response.StatusCode, bodyReadErr.Error())
			}
			return
		}
	}

	if _printSuccessDots {
		fmt.Print(".")
	}
	if _printBody {
		bodyStr := string(body)
		if bodyStr == "" {
			bodyStr = "(no body)"
		}
		fmt.Print(bodyStr)
	}
}

func mustReadJSONBytes(jsonPath string) []byte {
	jsonBytes, err := files.ReadFileBytes(jsonPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return jsonBytes
}

func mustExtractArgs() (string, string) {
	if len(os.Args) != 3 {
		fmt.Println("usage: go run load.go <url> <sample.json OR sample json string>")
		os.Exit(1)
	}

	url := os.Args[1]
	jsonPathOrString := os.Args[2]

	return url, jsonPathOrString
}
