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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	cc "github.com/cortexlabs/cortex/pkg/operator/cortexconfig"
)

const defaultTelemetryURL = "https://telemetry.cortexlabs.dev"
const eventPath = "/events"
const errorPath = "/errors"

var defaultClient *Client

type Client struct {
	url  string
	http http.Client
}

func getDefaultClient() *Client {
	if defaultClient != nil {
		return defaultClient
	}

	timeout := time.Duration(10 * time.Second)
	httpClient := http.Client{
		Timeout: timeout,
	}
	defaultClient = &Client{
		url:  cr.MustStringFromEnv("CONST_TELEMETRY_URL", &cr.StringValidation{Required: false, Default: defaultTelemetryURL}),
		http: httpClient,
	}

	return defaultClient
}

type UsageEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Version    string    `json:"version"`
	OperatorID string    `json:"operator_id"`
	Event      string    `json:"event"`
}

func (client *Client) sendUsageEvent(operatorID string, name string) {
	usageEvent := UsageEvent{
		Timestamp:  time.Now(),
		Version:    consts.CortexVersion,
		OperatorID: operatorID,
		Event:      name,
	}

	byteArray, _ := json.Marshal(usageEvent)
	resp, err := client.http.Post(client.url+eventPath, "application/json", bytes.NewReader(byteArray))
	if err != nil {
		errors.PrintError(err)
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

func (client *Client) sendErrorEvent(operatorID string, err error) {
	errorEvent := ErrorEvent{
		Timestamp:  time.Now(),
		Version:    consts.CortexVersion,
		OperatorID: operatorID,
		Title:      err.Error(),
		Stacktrace: fmt.Sprintf("%+v", err),
	}
	byteArray, _ := json.Marshal(errorEvent)
	resp, err := client.http.Post(client.url+errorPath, "application/json", bytes.NewReader(byteArray))
	if err != nil {
		errors.PrintError(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if byteArray, err := ioutil.ReadAll(resp.Body); err == nil {
			fmt.Println(string(byteArray))
		}
	}
}

func ReportEvent(name string) {
	if cc.EnableTelemetry {
		fmt.Println(cc.EnableTelemetry)
		go getDefaultClient().sendUsageEvent(aws.HashedAccountID, name)
	}
}

func ReportErrorBlocking(err error) {
	if cc.EnableTelemetry {
		getDefaultClient().sendErrorEvent(aws.HashedAccountID, err)
	}
}

func ReportError(err error) {
	if cc.EnableTelemetry {
		go getDefaultClient().sendErrorEvent(aws.HashedAccountID, err)
	}
}
