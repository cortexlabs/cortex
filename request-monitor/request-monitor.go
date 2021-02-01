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
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	_tickInterval          = 10 * time.Second
	_requestSampleInterval = 1 * time.Second
	_defaultPort           = "15000"
)

var (
	logger           *zap.Logger
	requestCounter   Counter
	inFlightReqGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cortex_in_flight_requests",
		Help: "The number of in-flight requests for a cortex API",
	})
)

type Counter struct {
	sync.Mutex
	s []int
}

func (c *Counter) Append(val int) {
	c.Lock()
	defer c.Unlock()
	c.s = append(c.s, val)
}

func (c *Counter) GetAllAndDelete() []int {
	var output []int
	c.Lock()
	defer c.Unlock()
	output = c.s
	c.s = []int{}
	return output
}

// ./request-monitor -p port
func main() {
	var port = flag.String("p", _defaultPort, "port on which the server runs on")

	logLevelEnv := os.Getenv("CORTEX_LOG_LEVEL")
	var logLevelZap zapcore.Level
	switch logLevelEnv {
	case "DEBUG":
		logLevelZap = zapcore.DebugLevel
	case "INFO":
		logLevelZap = zapcore.InfoLevel
	case "WARNING":
		logLevelZap = zapcore.WarnLevel
	case "ERROR":
		logLevelZap = zapcore.ErrorLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.MessageKey = "message"

	var err error
	logger, err = zap.Config{
		Level:            zap.NewAtomicLevelAt(logLevelZap),
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}.Build()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	if _, err = os.OpenFile("/request_monitor_ready.txt", os.O_RDONLY|os.O_CREATE, 0666); err != nil {
		panic(err)
	}

	for {
		if _, err := os.Stat("/mnt/workspace/api_readiness.txt"); err == nil {
			break
		} else if os.IsNotExist(err) {
			logger.Debug("waiting for replica to be ready ...")
			time.Sleep(_tickInterval)
		} else {
			logger.Error("error encountered while looking for /mnt/workspace/api_readiness.txt") // unexpected
			time.Sleep(_tickInterval)
		}
	}

	requestCounter = Counter{}
	go launchCrons(&requestCounter)

	http.Handle("/metrics", promhttp.Handler())

	logger.Info(fmt.Sprintf("starting request monitor on :%s", *port))
	if err := http.ListenAndServe(fmt.Sprintf(":%s", *port), nil); err != nil {
		logger.Fatal(err.Error())
	}
}

func launchCrons(requestCounter *Counter) {
	updateGaugeTicker := time.NewTicker(_tickInterval)
	defer updateGaugeTicker.Stop()

	requestSamplingTimer := time.NewTimer(_requestSampleInterval)
	defer requestSamplingTimer.Stop()

	for {
		select {
		case <-updateGaugeTicker.C:
			go updateGauge(requestCounter)
		case <-requestSamplingTimer.C:
			go updateOpenConnections(requestCounter, requestSamplingTimer)
		}
	}
}

func updateGauge(counter *Counter) {
	requestCounts := counter.GetAllAndDelete()

	total := 0.0
	if len(requestCounts) > 0 {
		for _, val := range requestCounts {
			total += float64(val)
		}

		total /= float64(len(requestCounts))
	}
	logger.Debug(fmt.Sprintf("recorded %.2f in-flight requests on replica", total))
	inFlightReqGauge.Set(total)
}

func getFileCount() int {
	dir, err := os.Open("/mnt/requests")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = dir.Close()
	}()

	fileNames, err := dir.Readdirnames(0)
	if err != nil {
		panic(err)
	}
	return len(fileNames)
}

func updateOpenConnections(requestCounter *Counter, timer *time.Timer) {
	count := getFileCount()
	requestCounter.Append(count)
	timer.Reset(_requestSampleInterval)
}
