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
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/yaml"
)

var uploadedRequestHandlers = strset.New()

func getAPIs(config *userconfig.Config,
	models context.Models,
	datasetVersion string,
	impls map[string][]byte,
	pythonPackages context.PythonPackages,
) (context.APIs, error) {
	apis := context.APIs{}

	for _, apiConfig := range config.APIs {

		var buf bytes.Buffer
		var requestHandlerImplKey *string
		buf.WriteString(apiConfig.Name)
		buf.WriteString(apiConfig.ModelType.String())

		if apiConfig.RequestHandlerPath != nil {
			for _, pythonPackage := range pythonPackages {
				buf.WriteString(pythonPackage.GetID())
			}

			impl, ok := impls[*apiConfig.RequestHandlerPath]
			if !ok {
				return nil, errors.Wrap(userconfig.ErrorImplDoesNotExist(*apiConfig.RequestHandlerPath), userconfig.Identify(apiConfig))
			}
			implID := hash.Bytes(impl)
			buf.WriteString(implID)

			key := filepath.Join(consts.RequestHandlersDir, implID)
			requestHandlerImplKey = &key
		}

		if yaml.StartsWithEscapedAtSymbol(apiConfig.Model) {
			modelName, _ := yaml.ExtractAtSymbolText(apiConfig.Model)
			model := models[modelName]
			if model == nil {
				return nil, errors.Wrap(userconfig.ErrorUndefinedResource(modelName, resource.ModelType), userconfig.Identify(apiConfig), userconfig.ModelKey)
			}
			buf.WriteString(model.ID)
		} else {
			buf.WriteString(datasetVersion)
			buf.WriteString(apiConfig.Model)
		}

		id := hash.Bytes(buf.Bytes())

		apis[apiConfig.Name] = &context.API{
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           id,
					ResourceType: resource.APIType,
				},
			},
			API:                   apiConfig,
			Path:                  context.APIPath(apiConfig.Name, config.App.Name),
			RequestHandlerImplKey: requestHandlerImplKey,
		}

		if apiConfig.RequestHandlerPath != nil {
			uploadRequestHandlers(apis[apiConfig.Name], impls[*apiConfig.RequestHandlerPath])
		}
	}
	return apis, nil
}

func uploadRequestHandlers(api *context.API, impl []byte) error {
	implID := hash.Bytes(impl)

	if uploadedRequestHandlers.Has(implID) {
		return nil
	}

	isUploaded, err := config.AWS.IsS3File(*api.RequestHandlerImplKey)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(api), "upload")
	}

	if !isUploaded {
		err = config.AWS.UploadBytesToS3(impl, *api.RequestHandlerImplKey)
		if err != nil {
			return errors.Wrap(err, userconfig.Identify(api), "upload")
		}
	}

	uploadedRequestHandlers.Add(implID)
	return nil
}
