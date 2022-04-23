/*
Copyright 2022 Cortex Labs, Inc.

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

package spec

import (
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func FindDuplicateNames(apis []userconfig.API) []userconfig.API {
	names := make(map[string][]userconfig.API)

	for _, api := range apis {
		names[api.Name] = append(names[api.Name], api)
	}

	for name := range names {
		if len(names[name]) > 1 {
			return names[name]
		}
	}

	return nil
}

func surgeOrUnavailableValidator(str string) (string, error) {
	if strings.HasSuffix(str, "%") {
		parsed, ok := s.ParseInt32(strings.TrimSuffix(str, "%"))
		if !ok {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
		if parsed < 0 || parsed > 100 {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
	} else {
		parsed, ok := s.ParseInt32(str)
		if !ok {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
		if parsed < 0 {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
	}

	return str, nil
}

func verifyTotalWeight(apis []*userconfig.TrafficSplit) error {
	totalWeight := int32(0)
	for _, api := range apis {
		if !api.Shadow {
			totalWeight += api.Weight
		}
	}
	if totalWeight == 100 {
		return nil
	}
	return errors.Wrap(ErrorIncorrectTrafficSplitterWeightTotal(totalWeight), userconfig.APIsKey)
}

// areTrafficSplitterAPIsUnique gives error if the same API is used multiple times in TrafficSplitter
func areTrafficSplitterAPIsUnique(apis []*userconfig.TrafficSplit) error {
	names := make(map[string][]userconfig.TrafficSplit)
	for _, api := range apis {
		names[api.Name] = append(names[api.Name], *api)
	}
	var notUniqueAPIs []string
	for name := range names {
		if len(names[name]) > 1 {
			notUniqueAPIs = append(notUniqueAPIs, names[name][0].Name)
		}
	}
	if len(notUniqueAPIs) > 0 {
		return errors.Wrap(ErrorTrafficSplitterAPIsNotUnique(notUniqueAPIs), userconfig.APIsKey)
	}
	return nil
}
