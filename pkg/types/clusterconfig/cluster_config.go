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

package clusterconfig

import (
	"github.com/cortexlabs/cortex/pkg/consts"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/types"
)

const (
	ClusterNameTag       = "cortex.dev/cluster-name"
	MaxNodePoolsOrGroups = 100
)

var (
	_invalidTagPrefixes = []string{"kubernetes.io/", "k8s.io/", "eksctl.", "alpha.eksctl.", "beta.eksctl.", "aws:", "Aws:", "aWs:", "awS:", "aWS:", "AwS:", "aWS:", "AWS:"}
)

func validateImageVersion(image string) (string, error) {
	return cr.ValidateImageVersion(image, consts.CortexVersion)
}

// Will return an error if the provider is not provided in the config file, or is not valid
func GetClusterProviderType(clusterConfigPath string) (types.ProviderType, error) {
	providerHolder := struct {
		Provider types.ProviderType `json:"provider" yaml:"provider"`
	}{}

	providerValidation := &cr.StructValidation{
		AllowExtraFields: true,
		StructFieldValidations: []*cr.StructFieldValidation{
			{
				StructField: "Provider",
				StringValidation: &cr.StringValidation{
					Required:      true,
					AllowedValues: types.ClusterProviderTypeStrings(),
				},
				Parser: func(str string) (interface{}, error) {
					return types.ProviderTypeFromString(str), nil
				},
			},
		},
	}

	errs := cr.ParseYAMLFile(&providerHolder, providerValidation, clusterConfigPath)
	if errors.HasError(errs) {
		return types.UnknownProviderType, errors.FirstError(errs...)
	}

	return providerHolder.Provider, nil
}

func specificProviderTypeValidator(expectedProvider types.ProviderType) func(string) (string, error) {
	return func(providerType string) (string, error) {
		if providerType != expectedProvider.String() {
			return "", ErrorProviderMismatch(expectedProvider, providerType)
		}
		return providerType, nil
	}
}

func validateClusterName(clusterName string) (string, error) {
	if !_strictS3BucketRegex.MatchString(clusterName) {
		return "", errors.Wrap(ErrorDidNotMatchStrictS3Regex(), clusterName)
	}
	return clusterName, nil
}
