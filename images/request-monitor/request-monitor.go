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
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

const _tickOffset = 1 * time.Second
const _tickInterval = 10 * time.Second
const _requestSampleInterval = 1 * time.Second

var (
	client   *cloudwatch.CloudWatch
	apiName  string
	region   string
	logGroup string
)

type Counter struct {
	sync.Mutex
	s []int
}

func (c *Counter) Append(val int) {
	c.Lock()
	c.s = append(c.s, val)
	c.Unlock()
}

func (c *Counter) GetAllAndDelete() []int {
	var output []int
	c.Lock()
	output = c.s
	c.s = []int{}
	c.Unlock()
	return output
}

// ./request-monitor api_name log_group
func main() {
	apiName = os.Args[1]
	logGroup = os.Args[2]
	region = os.Getenv("CORTEX_REGION")

	sess, err := session.NewSession(&aws.Config{
		Credentials: nil,
		Region:      aws.String(region),
	})
	if err != nil {
		panic(err)
	}

	client = cloudwatch.New(sess)
	requestCounter := Counter{}

	for {
		if _, err := os.Stat("/mnt/health_check.txt"); err == nil {
			break
		} else if os.IsNotExist(err) {
			fmt.Println("waiting...")
			time.Sleep(_tickInterval)
		} else {
			log.Printf("error encountered while looking for /mnt/health_check.txt") // unexpected
		}
	}

	targetTime := time.Now()
	roundedTime := targetTime.Round(_tickInterval)
	if roundedTime.Before(targetTime) {
		roundedTime = roundedTime.Add(_tickInterval)
	}

	targetTime = roundedTime.Add(_tickOffset)
	startPublishing := time.NewTimer(targetTime.Sub(time.Now()))
	defer startPublishing.Stop()

	requestSampler := time.NewTimer(_requestSampleInterval)
	defer requestSampler.Stop()

	for {
		select {
		case <-startPublishing.C:
			go startPublisher(apiName, &requestCounter, client)
		case <-requestSampler.C:
			go updateOpenConnections(&requestCounter, requestSampler)
		}
	}
}

func startPublisher(apiName string, requestCounter *Counter, client *cloudwatch.CloudWatch) {
	metricsPublisher := time.NewTicker(_tickInterval)
	defer metricsPublisher.Stop()

	go publishStats(apiName, requestCounter, client)
	for {
		select {
		case <-metricsPublisher.C:
			go publishStats(apiName, requestCounter, client)
		}
	}
}

func publishStats(apiName string, counter *Counter, client *cloudwatch.CloudWatch) {
	requestCounts := counter.GetAllAndDelete()

	total := 0.0
	for _, val := range requestCounts {
		total += float64(val)
	}

	total /= float64(len(requestCounts))
	log.Printf("recorded %.2f in-flight requests", total)
	curTime := time.Now()
	metricData := cloudwatch.PutMetricDataInput{
		Namespace: aws.String(logGroup),
		MetricData: []*cloudwatch.MetricDatum{
			{
				MetricName: aws.String("in-flight"),
				Dimensions: []*cloudwatch.Dimension{
					{
						Name:  aws.String("apiName"),
						Value: aws.String(apiName),
					},
				},
				Timestamp:         &curTime,
				Value:             aws.Float64(total),
				Unit:              aws.String("Count"),
				StorageResolution: aws.Int64(1),
			},
		},
	}
	_, err := client.PutMetricData(&metricData)
	if err != nil {
		log.Printf("error: publishing metrics: %s", err.Error()) // TODO
	}
}

func updateOpenConnections(requestCounter *Counter, timer *time.Timer) {
	cmd := exec.Command("ss")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	output := out.String()
	count := 0
	for _, str := range strings.Split(output, "\n") {
		if strings.Contains(str, ":8888 ") {
			count++
		}
	}
	requestCounter.Append(count)
	timer.Reset(_requestSampleInterval)
}
