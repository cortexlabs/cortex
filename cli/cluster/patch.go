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

package cluster

import (
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

func Patch(operatorConfig OperatorConfig, configPath string, force bool) ([]schema.DeployResult, error) {
	params := map[string]string{
		"force":          s.Bool(force),
		"configFileName": filepath.Base(configPath),
	}

	configBytes, err := files.ReadFileBytes(configPath)
	if err != nil {
		return nil, err
	}

	response, err := HTTPPostJSON(operatorConfig, "/patch", configBytes, params)
	if err != nil {
		return nil, err
	}

	var deployResults []schema.DeployResult
	if err := json.Unmarshal(response, &deployResults); err != nil {
		return nil, errors.Wrap(err, "/patch", string(response))
	}

	return deployResults, nil
}
