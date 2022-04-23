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
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/types/status"
)

func replicaCountTable(counts *status.ReplicaCounts) table.Table {
	var rows [][]interface{}
	for _, replicaCountType := range status.ReplicaCountTypes {
		count := counts.GetCountBy(replicaCountType)
		canBeHiddenIfZero := false
		switch replicaCountType {
		case status.ReplicaCountFailed:
			canBeHiddenIfZero = true
		case status.ReplicaCountKilled:
			canBeHiddenIfZero = true
		case status.ReplicaCountKilledOOM:
			canBeHiddenIfZero = true
		case status.ReplicaCountErrImagePull:
			canBeHiddenIfZero = true
		case status.ReplicaCountUnknown:
			canBeHiddenIfZero = true
		case status.ReplicaCountStalled:
			canBeHiddenIfZero = true
		}
		if count == 0 && canBeHiddenIfZero {
			continue
		}
		rows = append(rows, []interface{}{
			replicaCountType,
			count,
		})
	}

	return table.Table{
		Headers: []table.Header{
			{Title: _titleReplicaStatus, MinWidth: 32, MaxWidth: 32},
			{Title: _titleReplicaCount},
		},
		Rows: rows,
	}
}
