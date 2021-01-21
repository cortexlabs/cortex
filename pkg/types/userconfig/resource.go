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

package userconfig

import "fmt"

type Resource struct {
	Name string `json:"name" yaml:"name"`
	Kind Kind   `json:"kind" yaml:"kind"`
}

func (r Resource) UserString() string {
	return fmt.Sprintf("%s (%s)", r.Name, r.Kind.String())
}
