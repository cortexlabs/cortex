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

func realtimeAPITable(realtimeAPI schema.APIResponse, env cliconfig.Environment) (string, error) {
	var out string

	t := realtimeAPIsTable([]schema.APIResponse{realtimeAPI}, []string{env.Name})
	out += t.MustFormat()

	if realtimeAPI.DashboardURL != nil && *realtimeAPI.DashboardURL != "" {
		out += "\n" + console.Bold("metrics dashboard: ") + *realtimeAPI.DashboardURL + "\n"
	}

	if realtimeAPI.Endpoint != nil {
		out += "\n" + console.Bold("endpoint: ") + *realtimeAPI.Endpoint + "\n"
	}

	out += "\n" + apiHistoryTable(realtimeAPI.APIVersions)

	if !_flagVerbose {
		return out, nil
	}

	out += titleStr("configuration") + strings.TrimSpace(realtimeAPI.Spec.UserStr())

	return out, nil
}

func realtimeDescribeAPITable(realtimeAPI schema.APIResponse, env cliconfig.Environment) (string, error) {
	if realtimeAPI.Metadata == nil {
		return "", errors.ErrorUnexpected("missing metadata from operator response")
	}

	if realtimeAPI.Status == nil {
		return "", errors.ErrorUnexpected(fmt.Sprintf("missing status for %s api", realtimeAPI.Metadata.Name))
	}

	t := realtimeAPIsTable([]schema.APIResponse{realtimeAPI}, []string{env.Name})
	out := t.MustFormat()

	if realtimeAPI.DashboardURL != nil && *realtimeAPI.DashboardURL != "" {
		out += "\n" + console.Bold("metrics dashboard: ") + *realtimeAPI.DashboardURL + "\n"
	}

	if realtimeAPI.Endpoint != nil {
		out += "\n" + console.Bold("endpoint: ") + *realtimeAPI.Endpoint + "\n"
	}

	t = replicaCountTable(realtimeAPI.Status.ReplicaCounts)
	out += "\n" + t.MustFormat()

	return out, nil
}

func realtimeAPIsTable(realtimeAPIs []schema.APIResponse, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(realtimeAPIs))

	for i, realtimeAPI := range realtimeAPIs {
		if realtimeAPI.Metadata == nil || realtimeAPI.Status == nil {
			continue
		}
		lastUpdated := time.Unix(realtimeAPI.Metadata.LastUpdated, 0)
		rows = append(rows, []interface{}{
			envNames[i],
			realtimeAPI.Metadata.Name,
			fmt.Sprintf("%d/%d", realtimeAPI.Status.Ready, realtimeAPI.Status.Requested),
			realtimeAPI.Status.UpToDate,
			libtime.SinceStr(&lastUpdated),
		})
	}

	return table.Table{
		Headers: []table.Header{
			{Title: _titleEnvironment},
			{Title: _titleRealtimeAPI},
			{Title: _titleLive},
			{Title: _titleUpToDate},
			{Title: _titleLastUpdated},
		},
		Rows: rows,
	}
}
