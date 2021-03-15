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

func trafficSplitterTable(trafficSplitter schema.APIResponse, env cliconfig.Environment) (string, error) {
	var out string

	lastUpdated := time.Unix(trafficSplitter.Spec.LastUpdated, 0)

	t, err := trafficSplitTable(trafficSplitter, env)
	if err != nil {
		return "", err
	}
	t.FindHeaderByTitle(_titleEnvironment).Hidden = true

	out += t.MustFormat()

	out += "\n" + console.Bold("last updated: ") + libtime.SinceStr(&lastUpdated)
	out += "\n" + console.Bold("endpoint: ") + trafficSplitter.Endpoint + "\n"

	out += "\n" + apiHistoryTable(trafficSplitter.APIVersions)

	if !_flagVerbose {
		return out, nil
	}

	out += titleStr("configuration") + strings.TrimSpace(trafficSplitter.Spec.UserStr(env.Provider))

	return out, nil
}

func trafficSplitTable(trafficSplitter schema.APIResponse, env cliconfig.Environment) (table.Table, error) {
	rows := make([][]interface{}, 0, len(trafficSplitter.Spec.APIs))

	for _, api := range trafficSplitter.Spec.APIs {
		apisRes, err := cluster.GetAPI(MustGetOperatorConfig(env.Name), api.Name)
		if err != nil {
			return table.Table{}, err
		}

		apiRes := apisRes[0]
		lastUpdated := time.Unix(apiRes.Spec.LastUpdated, 0)

		apiName := apiRes.Spec.Name
		if api.Shadow {
			apiName += " (shadow)"
		}
		rows = append(rows, []interface{}{
			env.Name,
			apiName,
			api.Weight,
			apiRes.Status.Message(),
			apiRes.Status.Requested,
			libtime.SinceStr(&lastUpdated),
			latencyStr(apiRes.Metrics),
			code2XXStr(apiRes.Metrics),
			code5XXStr(apiRes.Metrics),
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

func trafficSplitterListTable(trafficSplitter []schema.APIResponse, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(trafficSplitter))
	for i, splitAPI := range trafficSplitter {
		lastUpdated := time.Unix(splitAPI.Spec.LastUpdated, 0)
		var apis []string
		for _, api := range splitAPI.Spec.APIs {
			apiName := api.Name
			if api.Shadow {
				apiName += " (shadow)"
			}
			apis = append(apis, apiName+":"+s.Int32(api.Weight))
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
