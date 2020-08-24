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

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

const (
	_titleTrafficSplitter   = "traffic splitter"
	_trafficSplitterWeights = "weights"
	_titleAPIs              = "apis"
)

func trafficSplitterTable(trafficSplitter *schema.TrafficSplitter, env cliconfig.Environment) (string, error) {
	var out string

	lastUpdated := time.Unix(trafficSplitter.Spec.LastUpdated, 0)

	t, err := trafficSplitTable(*trafficSplitter, env)
	if err != nil {
		return "", err
	}
	t.FindHeaderByTitle(_titleEnvironment).Hidden = true

	out += t.MustFormat()

	out += "\n" + console.Bold("last updated: ") + libtime.SinceStr(&lastUpdated)
	out += "\n" + console.Bold("endpoint: ") + trafficSplitter.Endpoint
	out += fmt.Sprintf("\n%s curl %s -X POST -H \"Content-Type: application/json\" -d @sample.json\n", console.Bold("curl:"), trafficSplitter.Endpoint)

	out += titleStr("configuration") + strings.TrimSpace(trafficSplitter.Spec.UserStr(env.Provider))

	return out, nil
}

func trafficSplitTable(trafficSplitter schema.TrafficSplitter, env cliconfig.Environment) (table.Table, error) {
	rows := make([][]interface{}, 0, len(trafficSplitter.Spec.APIs))

	for _, api := range trafficSplitter.Spec.APIs {
		apiRes, err := cluster.GetAPI(MustGetOperatorConfig(env.Name), api.Name)
		if err != nil {
			return table.Table{}, err
		}
		lastUpdated := time.Unix(apiRes.RealtimeAPI.Spec.LastUpdated, 0)
		rows = append(rows, []interface{}{
			env.Name,
			apiRes.RealtimeAPI.Spec.Name,
			api.Weight,
			apiRes.RealtimeAPI.Status.Message(),
			apiRes.RealtimeAPI.Status.Requested,
			libtime.SinceStr(&lastUpdated),
			latencyStr(&apiRes.RealtimeAPI.Metrics),
			code2XXStr(&apiRes.RealtimeAPI.Metrics),
			code5XXStr(&apiRes.RealtimeAPI.Metrics),
		})
	}

	return table.Table{
		Headers: []table.Header{
			{Title: _titleEnvironment},
			{Title: _titleAPIs},
			{Title: _trafficSplitterWeights},
			{Title: _titleStatus},
			{Title: _titleRequested},
			{Title: _titleLastupdated},
			{Title: _titleAvgRequest},
			{Title: _title2XX},
			{Title: _title5XX},
		},
		Rows: rows,
	}, nil
}

func trafficSplitterListTable(trafficSplitter []schema.TrafficSplitter, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(trafficSplitter))
	for i, splitAPI := range trafficSplitter {
		lastUpdated := time.Unix(splitAPI.Spec.LastUpdated, 0)
		var apis []string
		for _, api := range splitAPI.Spec.APIs {
			apis = append(apis, api.Name+":"+s.Int32(api.Weight))
		}
		apisStr := s.TruncateEllipses(strings.Join(apis, " "), 50)
		rows = append(rows, []interface{}{
			envNames[i],
			splitAPI.Spec.Name,
			apisStr,
			libtime.SinceStr(&lastUpdated),
		})
	}

	return table.Table{
		Headers: []table.Header{
			{Title: _titleEnvironment},
			{Title: _titleTrafficSplitter},
			{Title: _titleAPIs},
			{Title: _titleLastupdated},
		},
		Rows: rows,
	}
}
