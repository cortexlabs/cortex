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
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

type Column interface {
	Resource
	IsRaw() bool
}

func (config *Config) ColumnNames() []string {
	return append(config.RawColumns.Names(), config.TransformedColumns.Names()...)
}

func (config *Config) IsRawColumn(name string) bool {
	return slices.HasString(config.RawColumns.Names(), name)
}

func (config *Config) IsTransformedColumn(name string) bool {
	return slices.HasString(config.TransformedColumns.Names(), name)
}
