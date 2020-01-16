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

//go:generate python3 gen_resource_metadata.py
//go:generate gofmt -s -w resource_metadata.go

package aws

import (
	"sort"

	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

var EKSSupportedRegions strset.Set
var EKSSupportedRegionsSlice []string

func init() {
	EKSSupportedRegions = strset.New()
	for region := range InstanceMetadatas {
		EKSSupportedRegions.Add(region)
	}
	EKSSupportedRegionsSlice = EKSSupportedRegions.Slice()
	sort.Strings(EKSSupportedRegionsSlice)
}
