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

package cloud

import (
	"github.com/cortexlabs/cortex/pkg/lib/cloud"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/consts"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

type HadoopPathValidation struct {
	Required bool
	Default  string
}

func GetHadoopPathValidation(v *HadoopPathValidation) *cr.StringValidation {
	validator := func(val string) (string, error) {
		isValid, err := config.Cloud.IsValidHadoopPath(val)
		if err != nil {
			return "", err
		}
		if !isValid {
			return "", cloud.ErrorInvalidHadoopPath(val)
		}

		cloudType, err := cloud.ProviderTypeFromPath(val)
		if err != nil {
			return "", err
		}

		if cloudType == cloud.LocalProviderType {
			return filepath.Join(consts.ExternalDataDir, hash.String(val)), nil
		}

		return val, nil
	}

	return &cr.StringValidation{
		Required:  v.Required,
		Default:   v.Default,
		Validator: validator,
	}
}
