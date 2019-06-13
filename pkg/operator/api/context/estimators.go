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

type Estimators map[string]*Estimator

type Estimator struct {
	*userconfig.Estimator
	*ResourceFields
	Namespace *string `json:"namespace"`
	ImplKey   string  `json:"impl_key"`
}

func (estimators Estimators) OneByID(id string) *Estimator {
	for _, estimator := range estimators {
		if estimator.ID == id {
			return estimator
		}
	}
	return nil
}
