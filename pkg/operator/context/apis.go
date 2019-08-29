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
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func getAPIs(config *userconfig.Config,
	deploymentVersion string,
	impls map[string][]byte,
) (context.APIs, error) {
	apis := context.APIs{}

	for _, apiConfig := range config.APIs {
		var buf bytes.Buffer
		var requestHandlerImplKey *string
		buf.WriteString(apiConfig.Name)
		buf.WriteString(s.Obj(apiConfig.Tracker))
		buf.WriteString(apiConfig.ModelFormat.String())

		if apiConfig.RequestHandler != nil {
			impl, ok := impls[*apiConfig.RequestHandler]
			if !ok {
				return nil, errors.Wrap(userconfig.ErrorImplDoesNotExist(*apiConfig.RequestHandler), userconfig.Identify(apiConfig), userconfig.RequestHandlerKey)
			}
			implID := hash.Bytes(impl)
			buf.WriteString(implID)

			requestHandlerImplKey = pointer.String(filepath.Join(consts.RequestHandlersDir, implID))

			err := uploadRequestHandler(*requestHandlerImplKey, impls[*apiConfig.RequestHandler])
			if err != nil {
				return nil, errors.Wrap(err, userconfig.Identify(apiConfig))
			}
		}

		buf.WriteString(deploymentVersion)
		buf.WriteString(strings.TrimSuffix(apiConfig.Model, "/"))

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
	}
	return apis, nil
}

func uploadRequestHandler(implKey string, impl []byte) error {
	isUploaded, err := config.AWS.IsS3File(implKey)
	if err != nil {
		return errors.Wrap(err, "upload")
	}

	if !isUploaded {
		err = config.AWS.UploadBytesToS3(impl, implKey)
		if err != nil {
			return errors.Wrap(err, "upload")
		}
	}

	return nil
}
