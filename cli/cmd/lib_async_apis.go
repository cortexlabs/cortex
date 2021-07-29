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

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

func asyncAPITable(asyncAPI schema.APIResponse, env cliconfig.Environment) (string, error) {
	var out string

	t := asyncAPIsTable([]schema.APIResponse{asyncAPI}, []string{env.Name})

	out += t.MustFormat()

	if asyncAPI.DashboardURL != nil && *asyncAPI.DashboardURL != "" {
		out += "\n" + console.Bold("metrics dashboard: ") + *asyncAPI.DashboardURL + "\n"
	}

	if asyncAPI.Endpoint != nil {
		out += "\n" + console.Bold("endpoint: ") + *asyncAPI.Endpoint + "\n"
	}

	out += "\n" + apiHistoryTable(asyncAPI.APIVersions)

	if !_flagVerbose {
		return out, nil
	}

	out += titleStr("configuration") + strings.TrimSpace(asyncAPI.Spec.UserStr())

	return out, nil
}

func asyncDescribeAPITable(asyncAPI schema.APIResponse, env cliconfig.Environment) (string, error) {
	if asyncAPI.Metadata == nil {
		return "", errors.ErrorUnexpected("missing metadata from operator response")
	}

	if asyncAPI.ReplicaCounts == nil {
		return "", errors.ErrorUnexpected(fmt.Sprintf("missing replica counts for %s api", asyncAPI.Metadata.Name))
	}

	t := asyncAPIsTable([]schema.APIResponse{asyncAPI}, []string{env.Name})
	out := t.MustFormat()

	if asyncAPI.DashboardURL != nil && *asyncAPI.DashboardURL != "" {
		out += "\n" + console.Bold("metrics dashboard: ") + *asyncAPI.DashboardURL + "\n"
	}

	if asyncAPI.Endpoint != nil {
		out += "\n" + console.Bold("endpoint: ") + *asyncAPI.Endpoint + "\n"
	}

	t = replicaCountTable(asyncAPI.ReplicaCounts)
	out += "\n" + t.MustFormat()

	return out, nil
}

func asyncAPIsTable(asyncAPIs []schema.APIResponse, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(asyncAPIs))

	for i, asyncAPI := range asyncAPIs {
		if asyncAPI.Metadata == nil || (asyncAPI.Status == nil && asyncAPI.ReplicaCounts == nil) {
			continue
		}

		var ready, requested, upToDate int32
		if asyncAPI.Status != nil {
			ready = asyncAPI.Status.Ready
			requested = asyncAPI.Status.Requested
			upToDate = asyncAPI.Status.UpToDate
		} else {
			ready = asyncAPI.ReplicaCounts.Ready
			requested = asyncAPI.ReplicaCounts.Requested
			upToDate = asyncAPI.ReplicaCounts.UpToDate
		}

		lastUpdated := time.Unix(asyncAPI.Metadata.LastUpdated, 0)
		rows = append(rows, []interface{}{
			envNames[i],
			asyncAPI.Metadata.Name,
			fmt.Sprintf("%d/%d", ready, requested),
			upToDate,
			libtime.SinceStr(&lastUpdated),
		})
	}

	return table.Table{
		Headers: []table.Header{
			{Title: _titleEnvironment},
			{Title: _titleAsyncAPI},
			{Title: _titleLive},
			{Title: _titleUpToDate},
			{Title: _titleLastUpdated},
		},
		Rows: rows,
	}
}
