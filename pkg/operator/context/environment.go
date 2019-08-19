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

package context

import (
	"bytes"

	"github.com/cortexlabs/yaml"

	"github.com/cortexlabs/cortex/pkg/lib/hash"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func getEnvironment(config *userconfig.Config, deploymentVersion string) *context.Environment {
	return &context.Environment{
		Environment: config.Environment,
		ID:          dataID(config, deploymentVersion),
	}
}

func dataID(config *userconfig.Config, deploymentVersion string) string {
	var buf bytes.Buffer
	buf.WriteString(deploymentVersion)

	rawColumnTypeMap := make(map[string]userconfig.ColumnType, len(config.RawColumns))
	for _, rawColumnConfig := range config.RawColumns {
		rawColumnTypeMap[rawColumnConfig.GetName()] = rawColumnConfig.GetColumnType()
	}
	buf.WriteString(s.Obj(config.Environment.Limit))
	buf.WriteString(s.Obj(rawColumnTypeMap))

	data := config.Environment.Data
	switch typedData := data.(type) {
	case *userconfig.CSVData:
		buf.WriteString(s.Obj(typedData))
	case *userconfig.ParquetData:
		buf.WriteString(typedData.Type.String())
		buf.WriteString(typedData.Path)
		buf.WriteString(s.Bool(typedData.DropNull))
		schemaMap := map[string]string{} // use map to sort keys
		for _, parqCol := range typedData.Schema {
			colName, _ := yaml.ExtractAtSymbolText(parqCol.RawColumn)
			schemaMap[colName] = parqCol.ParquetColumnName
		}
		buf.WriteString(s.Obj(schemaMap))
	}

	return hash.Bytes(buf.Bytes())
}
