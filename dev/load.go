/*
Copyright 2021 Cortex Labs, Inc.

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
	"sync"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/files"
)

// usage: go run load.go <url> <sample.json>

// either set _numConcurrent > 0 or _requestInterval > 0 (and configure the corresponding sections)

// constant in-flight requests
const (
	_numConcurrent        = 5
	_numRequestsPerThread = -1 // -1 means loop infinitely
	_requestDelay         = 0 * time.Second
	_numMainLoops         = 10 // only relevant if _numRequestsPerThread != -1
)

// constant requests per second
const (
	_requestInterval = -1 * time.Millisecond
	_maxInFlight     = 5
)

// other options
const (
	_printSuccessDots = true
)

type Counter struct {
	sync.Mutex
	count int
}

var _client = &http.Client{
	Timeout: 0, // no timeout
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func main() {
	if _numConcurrent > 0 && _requestInterval > 0 {
		fmt.Println("error: you must set either _numConcurrent or _requestInterval, but not both")
		os.Exit(1)
	}

	if _numConcurrent == 0 && _requestInterval == 0 {
		fmt.Println("error: you must set either _numConcurrent or _requestInterval")
		os.Exit(1)
	}

	url, jsonPath := mustExtractArgs()
	jsonBytes := mustReadJSONBytes(jsonPath)

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

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			go runConstantRequestsPerSecondIteration(url, jsonBytes, &inFlightCount)
		}
	}
}

func runConstantRequestsPerSecondIteration(url string, jsonBytes []byte, inFlightCount *Counter) {
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

	if _maxInFlight > 0 {
		inFlightCount.Lock()
		inFlightCount.count--
		inFlightCount.Unlock()
	}
}

func runConstantInFlight(url string, jsonBytes []byte) {
	start := time.Now()
	loopNum := 1
	for true {
		runConstantInFlightIteration(url, jsonBytes)
		if loopNum >= _numMainLoops {
			break
		}
		loopNum++
	}
	fmt.Println("total elapsed time:", time.Now().Sub(start))
}

func runConstantInFlightIteration(url string, jsonBytes []byte) {
	start := time.Now()

	if _numRequestsPerThread > 0 {
		fmt.Printf("spawning %d threads, %d requests each, %s delay on each\n", _numConcurrent, _numRequestsPerThread, _requestDelay.String())
	} else {
		fmt.Printf("spawning %d infinite threads, %s delay on each\n", _numConcurrent, _requestDelay.String())
	}

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

	for _, doneChan := range doneChans {
		<-doneChan
	}

	fmt.Println()
	fmt.Println("elapsed time:", time.Now().Sub(start))
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
	response, err := _client.Do(request)
	if err != nil {
		fmt.Print("\n" + debug.Sppg(err))
		return
	}

	body, bodyReadErr := ioutil.ReadAll(response.Body)
	response.Body.Close()

	if response.StatusCode != 200 {
		if bodyReadErr == nil {
			fmt.Printf("\nstatus code: %d; body: %s\n", response.StatusCode, string(body))
		} else {
			fmt.Printf("\nstatus code: %d; error reading body: %s\n", response.StatusCode, bodyReadErr.Error())
		}
		return
	}

	if _printSuccessDots {
		fmt.Print(".")
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
		fmt.Println("usage: go run load.go <url> <sample.json>")
		os.Exit(1)
	}

	url := os.Args[1]
	jsonPath := os.Args[2]

	return url, jsonPath
}
