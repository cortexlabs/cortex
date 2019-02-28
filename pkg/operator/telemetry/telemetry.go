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
	"github.com/cortexlabs/cortex/pkg/consts"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	cc "github.com/cortexlabs/cortex/pkg/operator/cortexconfig"
	"io/ioutil"
	"net/http"
	"time"
)

const defaultTelemetryURL = "https://reporting.cortexlabs.com"
const eventPath = "/events"
const errorPath = "/errors"

var telemetryURL string

func init() {
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
	resp, err := http.Post(telemetryURL+eventPath, "application/json", bytes.NewReader(byteArray))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 && resp.StatusCode >= 300 {
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

func ReportError(err error) {
	if cc.EnableTelemetry {
		sendErrorEvent(aws.HashedAccountID, err)
	}
}

func ReportErrorAsync(err error) {
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
	resp, err := http.Post(telemetryURL+errorPath, "application/json", bytes.NewReader(byteArray))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 && resp.StatusCode >= 300 {
		if byteArray, err := ioutil.ReadAll(resp.Body); err == nil {
			fmt.Println(string(byteArray))
		}
	}
}
