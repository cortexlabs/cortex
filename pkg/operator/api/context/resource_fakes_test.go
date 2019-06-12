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

type fakeResource struct {
	Name         string
	ID           string
	ResourceType resource.Type
}

func (r *fakeResource) GetID() string {
	return r.ID
}

func (r *fakeResource) GetName() string {
	return r.Name
}

func (r *fakeResource) GetResourceType() resource.Type {
	return r.ResourceType
}

func (r *fakeResource) GetIndex() int {
	return -1
}

func (r *fakeResource) SetIndex(int) {}

func (r *fakeResource) GetFilePath() string {
	return "path/to/file"
}

func (r *fakeResource) SetFilePath(string) {}

func (r *fakeResource) GetEmbed() *userconfig.Embed {
	return nil
}

func (r *fakeResource) SetEmbed(*userconfig.Embed) {}

var c1 = &fakeResource{
	Name:         "c1",
	ID:           "a_c1",
	ResourceType: resource.ConstantType,
}

var c2 = &fakeResource{
	Name:         "c2",
	ID:           "b_c2",
	ResourceType: resource.ConstantType,
}

var c3 = &fakeResource{
	Name:         "c3",
	ID:           "c_c3",
	ResourceType: resource.ConstantType,
}

var rc1 = &fakeResource{
	Name:         "rc1",
	ID:           "d_rc1",
	ResourceType: resource.RawColumnType,
}

var rc2 = &fakeResource{
	Name:         "rc2",
	ID:           "e_rc2",
	ResourceType: resource.RawColumnType,
}

var rc3 = &fakeResource{
	Name:         "rc3",
	ID:           "f_rc3",
	ResourceType: resource.RawColumnType,
}

var agg1 = &fakeResource{
	Name:         "agg1",
	ID:           "g_agg1",
	ResourceType: resource.AggregateType,
}

var agg2 = &fakeResource{
	Name:         "agg2",
	ID:           "h_agg2",
	ResourceType: resource.AggregateType,
}

var agg3 = &fakeResource{
	Name:         "agg3",
	ID:           "i_agg3",
	ResourceType: resource.AggregateType,
}

var tc1 = &fakeResource{
	Name:         "tc1",
	ID:           "j_tc1",
	ResourceType: resource.TransformedColumnType,
}

var tc2 = &fakeResource{
	Name:         "tc2",
	ID:           "k_tc2",
	ResourceType: resource.TransformedColumnType,
}

var tc3 = &fakeResource{
	Name:         "tc3",
	ID:           "l_tc3",
	ResourceType: resource.TransformedColumnType,
}

var constants = []Resource{c1, c2, c3}
var rawCols = []Resource{rc1, rc2, rc3}
var aggregates = []Resource{agg1, agg2, agg3}
var transformedCols = []Resource{tc1, tc2, tc3}
var allResources = []Resource{c1, c2, c3, rc1, rc2, rc3, agg1, agg2, agg3, tc1, tc2, tc3}
