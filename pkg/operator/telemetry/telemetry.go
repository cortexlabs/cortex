/*
Copyright 2019 Cortex Labs, Inc.

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

package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	cc "github.com/cortexlabs/cortex/pkg/operator/cortexconfig"
)

const defaultTelemetryURL = "https://telemetry.cortexlabs.com"
const eventPath = "/events"
const errorPath = "/errors"

var telemetryURL string
var httpClient http.Client

func init() {
	timeout := time.Duration(10 * time.Second)
	httpClient = http.Client{
		Timeout: timeout,
	}

	telemetryURL = cr.MustStringFromEnv("CONST_TELEMETRY_URL", &cr.StringValidation{Required: false, Default: defaultTelemetryURL})
}

type UsageEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Version    string    `json:"version"`
	OperatorID string    `json:"operator_id"`
	Event      string    `json:"event"`
}

func ReportEvent(name string) {
	if cc.EnableTelemetry {
		go sendUsageEvent(aws.HashedAccountID, name)
	}
}

func sendUsageEvent(operatorID string, name string) {
	usageEvent := UsageEvent{
		Timestamp:  time.Now(),
		Version:    consts.CortexVersion,
		OperatorID: operatorID,
		Event:      name,
	}

	byteArray, _ := json.Marshal(usageEvent)
	resp, err := httpClient.Post(telemetryURL+eventPath, "application/json", bytes.NewReader(byteArray))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if byteArray, err := ioutil.ReadAll(resp.Body); err == nil {
			fmt.Println(string(byteArray))
		}
	}
}

type ErrorEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Version    string    `json:"version"`
	OperatorID string    `json:"operator_id"`
	Title      string    `json:"title"`
	Stacktrace string    `json:"stacktrace"`
}

func ReportErrorBlocking(err error) {
	if cc.EnableTelemetry {
		sendErrorEvent(aws.HashedAccountID, err)
	}
}

func ReportError(err error) {
	if cc.EnableTelemetry {
		go sendErrorEvent(aws.HashedAccountID, err)
	}
}

func sendErrorEvent(operatorID string, err error) {
	errorEvent := ErrorEvent{
		Timestamp:  time.Now(),
		Version:    consts.CortexVersion,
		OperatorID: operatorID,
		Title:      err.Error(),
		Stacktrace: fmt.Sprintf("%+v", err),
	}
	byteArray, _ := json.Marshal(errorEvent)
	resp, err := httpClient.Post(telemetryURL+errorPath, "application/json", bytes.NewReader(byteArray))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if byteArray, err := ioutil.ReadAll(resp.Body); err == nil {
			fmt.Println(string(byteArray))
		}
	}
}
