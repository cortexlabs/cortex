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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/yaml"
)

func getAPIs(config *userconfig.Config,
	models context.Models,
) (context.APIs, error) {
	apis := context.APIs{}

	for _, apiConfig := range config.APIs {
		modelName, _ := yaml.ExtractAtSymbolText(apiConfig.Model)

		model := models[modelName]
		if model == nil {
			return nil, errors.Wrap(userconfig.ErrorUndefinedResource(modelName, resource.ModelType), userconfig.Identify(apiConfig), userconfig.ModelNameKey)
		}

		var buf bytes.Buffer
		buf.WriteString(apiConfig.Name)
		if model, ok := models[apiConfig.Model]; ok {
			buf.WriteString(model.ID)
		}

		if apiConfig.ModelPath != nil {
			buf.WriteString(*apiConfig.ModelPath)
		}

		id := hash.Bytes(buf.Bytes())

		apis[apiConfig.Name] = &context.API{
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           id,
					ResourceType: resource.APIType,
				},
			},
			API:       apiConfig,
			Path:      context.APIPath(apiConfig.Name, config.App.Name),
			ModelName: modelName,
		}
	}
	return apis, nil
}
