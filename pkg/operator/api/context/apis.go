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
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

type APIs map[string]*API

type API struct {
	*userconfig.API
	*ComputedResourceFields
	Path      string `json:"path"`
	ModelName string `json:"model_name"` // This is just a convenience which removes the @ from userconfig.API.Model
}

func APIPath(apiName string, appName string) string {
	return "/" + appName + "/" + apiName
}

func (apis APIs) OneByID(id string) *API {
	for _, api := range apis {
		if api.ID == id {
			return api
		}
	}
	return nil
}

func APIResourcesAndComputesMatch(ctx1 *Context, ctx2 *Context) bool {
	if ctx1 == nil && ctx2 == nil {
		return true
	}
	if ctx1 == nil || ctx2 == nil {
		return false
	}

	if len(ctx1.APIs) != len(ctx2.APIs) {
		return false
	}

	for apiName, api1 := range ctx1.APIs {
		api2, ok := ctx2.APIs[apiName]
		if !ok {
			return false
		}
		if api1.ID != api2.ID {
			return false
		}
		if api1.Compute.ID() != api2.Compute.ID() {
			return false
		}
	}

	return true
}
