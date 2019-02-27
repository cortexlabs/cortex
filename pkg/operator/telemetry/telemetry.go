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
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	cc "github.com/cortexlabs/cortex/pkg/operator/cortexconfig"
	"time"

	"net/http"
)

const defaultReportingURL = "https://reporting.cortexlabs.com"

var reportingURL string

func init() {
	if len(cc.BaseReportingURL) > 0 {
		reportingURL = cc.BaseReportingURL
	} else {
		reportingURL = defaultReportingURL
	}
}

type UsageEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Version    string    `json:"version"`
	OperatorID string    `json:"operator_id"`
	Event      string    `json:"event"`
}

func Report(name string) {
	if cc.EnableUsageReporting {
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
	http.Post(reportingURL, "application/json", bytes.NewReader(byteArray))
}
