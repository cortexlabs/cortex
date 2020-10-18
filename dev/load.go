/*
Copyright 2020 Cortex Labs, Inc.

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
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/files"
)

const _numConcurrent = 5
const _numRequestsPerThread = -1
const _requestDelay = 0 * time.Second
const _numMainLoops = 10

var _client = &http.Client{
	Timeout: 600 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func main() {
	start := time.Now()
	loopNum := 1
	for true {
		run()
		if loopNum >= _numMainLoops {
			break
		}
		loopNum++
	}
	fmt.Println("total elapsed time:", time.Now().Sub(start))
}

func run() {
	start := time.Now()

	if _numRequestsPerThread > 0 {
		fmt.Printf("spawning %d threads, %d requests each, %s delay on each\n", _numConcurrent, _numRequestsPerThread, _requestDelay.String())
	} else {
		fmt.Printf("spawning %d infinite threads, %s delay on each\n", _numConcurrent, _requestDelay.String())
	}

	url, jsonPath := mustExtractArgs()
	jsonBytes := mustReadJSONBytes(jsonPath)

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
	for true {
		if _numRequestsPerThread > 0 {
			if i >= _numRequestsPerThread {
				return
			}
			i++
		}

		response, _, err := makeRequest(url, jsonBytes)

		if err != nil {
			fmt.Print(err.Error())
			continue
		}

		if response.StatusCode != 200 {
			fmt.Print(response.StatusCode)
			fmt.Print(" ")
			continue
		}

		fmt.Print(".")

		if _requestDelay != 0 {
			time.Sleep(_requestDelay)
		}
	}
}

func makeRequest(url string, jsonBytes []byte) (*http.Response, string, error) {
	payload := bytes.NewBuffer(jsonBytes)

	request, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return nil, "", err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := _client.Do(request)
	if err != nil {
		return nil, "", err
	}

	defer response.Body.Close()

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, "", err
	}

	return response, string(bodyBytes), nil
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
