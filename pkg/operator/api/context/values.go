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
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func GetValueResource(
	name string,
	constants Constants,
	aggregates Aggregates,
) (ValueResource, error) {

	if constant, ok := constants[name]; ok {
		return constant, nil
	}
	if aggregate, ok := aggregates[name]; ok {
		return aggregate, nil
	}

	return nil, userconfig.ErrorUndefinedResource(name, resource.ConstantType, resource.AggregateType)
}
