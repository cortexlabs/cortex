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

package userconfig

import (
	"github.com/cortexlabs/cortex/pkg/api/resource"
	cr "github.com/cortexlabs/cortex/pkg/utils/configreader"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type APIs []*API

type API struct {
	Name      string      `json:"name" yaml:"name"`
	ModelName string      `json:"model_name" yaml:"model_name"`
	Compute   *APICompute `json:"compute" yaml:"compute"`
	Tags      Tags        `json:"tags" yaml:"tags"`
}

var apiValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		&cr.StructFieldValidation{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required: true,
				Dns1035:  true,
			},
		},
		&cr.StructFieldValidation{
			StructField:  "ModelName",
			DefaultField: "Name",
			StringValidation: &cr.StringValidation{
				Required:                   false,
				AlphaNumericDashUnderscore: true,
			},
		},
		apiComputeFieldValidation,
		tagsFieldValidation,
		typeFieldValidation,
	},
}

func (apis APIs) Validate() error {
	dups := util.FindDuplicateStrs(apis.Names())
	if len(dups) > 0 {
		return ErrorDuplicateConfigName(dups[0], resource.APIType)
	}
	return nil
}

func (api *API) GetName() string {
	return api.Name
}

func (api *API) GetResourceType() resource.Type {
	return resource.APIType
}

func (apis APIs) Names() []string {
	names := make([]string, len(apis))
	for i, api := range apis {
		names[i] = api.Name
	}
	return names
}
