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
	"bytes"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

func getRawFeatures(
	config *userconfig.Config,
	env *context.Environment,
	pythonPackages context.PythonPackages,
) (context.RawFeatures, error) {

	rawFeatures := context.RawFeatures{}

	for _, featureConfig := range config.RawFeatures {
		var buf bytes.Buffer
		buf.WriteString(env.ID)
		for _, pythonPackage := range pythonPackages {
			buf.WriteString(pythonPackage.GetID())
		}
		buf.WriteString(featureConfig.GetName())
		buf.WriteString(featureConfig.GetType())

		var rawFeature context.RawFeature
		switch cTypedFeature := featureConfig.(type) {
		case *userconfig.RawIntFeature:
			buf.WriteString(s.Bool(cTypedFeature.Required))
			buf.WriteString(s.Obj(cTypedFeature.Min))
			buf.WriteString(s.Obj(cTypedFeature.Max))
			buf.WriteString(s.Obj(util.SortInt64sCopy(cTypedFeature.Values)))
			id := util.HashBytes(buf.Bytes())
			idWithTags := util.HashStr(id + cTypedFeature.Tags.ID())
			rawFeature = &context.RawIntFeature{
				ComputedResourceFields: &context.ComputedResourceFields{
					ResourceFields: &context.ResourceFields{
						ID:           id,
						IDWithTags:   idWithTags,
						ResourceType: resource.RawFeatureType,
					},
				},
				RawIntFeature: cTypedFeature,
			}
		case *userconfig.RawFloatFeature:
			buf.WriteString(s.Bool(cTypedFeature.Required))
			buf.WriteString(s.Obj(cTypedFeature.Min))
			buf.WriteString(s.Obj(cTypedFeature.Max))
			buf.WriteString(s.Obj(util.SortFloat32sCopy(cTypedFeature.Values)))
			id := util.HashBytes(buf.Bytes())
			idWithTags := util.HashStr(id + cTypedFeature.Tags.ID())
			rawFeature = &context.RawFloatFeature{
				ComputedResourceFields: &context.ComputedResourceFields{
					ResourceFields: &context.ResourceFields{
						ID:           id,
						IDWithTags:   idWithTags,
						ResourceType: resource.RawFeatureType,
					},
				},
				RawFloatFeature: cTypedFeature,
			}
		case *userconfig.RawStringFeature:
			buf.WriteString(s.Bool(cTypedFeature.Required))
			buf.WriteString(s.Obj(util.SortStrsCopy(cTypedFeature.Values)))
			id := util.HashBytes(buf.Bytes())
			idWithTags := util.HashStr(id + cTypedFeature.Tags.ID())
			rawFeature = &context.RawStringFeature{
				ComputedResourceFields: &context.ComputedResourceFields{
					ResourceFields: &context.ResourceFields{
						ID:           id,
						IDWithTags:   idWithTags,
						ResourceType: resource.RawFeatureType,
					},
				},
				RawStringFeature: cTypedFeature,
			}
		default:
			return nil, errors.New(userconfig.Identify(featureConfig), s.ErrInvalidStr(userconfig.TypeKey, userconfig.IntegerFeatureType.String(), userconfig.FloatFeatureType.String(), userconfig.StringFeatureType.String())) // unexpected error
		}

		rawFeatures[featureConfig.GetName()] = rawFeature
	}

	return rawFeatures, nil
}
