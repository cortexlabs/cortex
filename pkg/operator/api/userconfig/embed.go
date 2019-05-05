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
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type Embeds []*Embed

type Embed struct {
	ResourceFields
	Template string                 `json:"template" yaml:"template"`
	Args     map[string]interface{} `json:"args" yaml:"args"`
}

var embedValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Template",
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		{
			StructField: "Args",
			InterfaceMapValidation: &cr.InterfaceMapValidation{
				Required:   false,
				AllowEmpty: true,
				Default:    make(map[string]interface{}),
			},
		},
		typeFieldValidation,
	},
}

func (embed *Embed) GetResourceType() resource.Type {
	return resource.EmbedType
}
