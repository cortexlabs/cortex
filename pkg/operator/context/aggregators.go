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
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func loadUserAggregators(
	config *userconfig.Config,
	impls map[string][]byte,
	pythonPackages context.PythonPackages,
) (map[string]*context.Aggregator, error) {

	userAggregators := make(map[string]*context.Aggregator)
	for _, aggregatorConfig := range config.Aggregators {
		impl, ok := impls[aggregatorConfig.Path]
		if !ok {
			return nil, errors.Wrap(ErrorImplDoesNotExist(aggregatorConfig.Path), userconfig.Identify(aggregatorConfig))
		}
		aggregator, err := newAggregator(*aggregatorConfig, impl, nil, pythonPackages)
		if err != nil {
			return nil, err
		}
		userAggregators[aggregator.Name] = aggregator
	}

	for _, aggregateConfig := range config.Aggregates {
		if aggregateConfig.AggregatorPath == nil {
			continue
		}

		impl, ok := impls[*aggregateConfig.AggregatorPath]
		if !ok {
			return nil, errors.Wrap(ErrorImplDoesNotExist(*aggregateConfig.AggregatorPath), userconfig.Identify(aggregateConfig))
		}

		implHash := hash.Bytes(impl)
		if _, ok := userAggregators[implHash]; ok {
			continue
		}

		anonAggregatorConfig := &userconfig.Aggregator{
			ResourceFields: userconfig.ResourceFields{
				Name: implHash,
			},
			Path: *aggregateConfig.AggregatorPath,
		}
		aggregator, err := newAggregator(*anonAggregatorConfig, impl, nil, pythonPackages)
		if err != nil {
			return nil, err
		}

		aggregateConfig.Aggregator = aggregator.Name
		userAggregators[anonAggregatorConfig.Name] = aggregator
	}

	return userAggregators, nil
}

func newAggregator(
	aggregatorConfig userconfig.Aggregator,
	impl []byte,
	namespace *string,
	pythonPackages context.PythonPackages,
) (*context.Aggregator, error) {

	implID := hash.Bytes(impl)

	var buf bytes.Buffer
	buf.WriteString(context.DataTypeID(aggregatorConfig.Inputs))
	buf.WriteString(context.DataTypeID(aggregatorConfig.OutputType))
	buf.WriteString(implID)

	for _, pythonPackage := range pythonPackages {
		buf.WriteString(pythonPackage.GetID())
	}

	id := hash.Bytes(buf.Bytes())

	aggregator := &context.Aggregator{
		ResourceFields: &context.ResourceFields{
			ID:           id,
			IDWithTags:   id,
			ResourceType: resource.AggregatorType,
			MetadataKey:  filepath.Join(consts.AggregatorsDir, id+"_metadata.json"),
		},
		Aggregator: &aggregatorConfig,
		Namespace:  namespace,
		ImplKey:    filepath.Join(consts.AggregatorsDir, implID+".py"),
	}
	aggregator.Aggregator.Path = ""

	if err := uploadAggregator(aggregator, impl); err != nil {
		return nil, err
	}

	return aggregator, nil
}

func uploadAggregator(aggregator *context.Aggregator, impl []byte) error {
	if uploadedAggregators.Has(aggregator.ID) {
		return nil
	}

	isUploaded, err := config.AWS.IsS3File(aggregator.ImplKey)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(aggregator), "upload")
	}

	if !isUploaded {
		err = config.AWS.UploadBytesToS3(impl, aggregator.ImplKey)
		if err != nil {
			return errors.Wrap(err, userconfig.Identify(aggregator), "upload")
		}
	}

	uploadedAggregators.Add(aggregator.ID)
	return nil
}

func getAggregator(
	name string,
	userAggregators map[string]*context.Aggregator,
) (*context.Aggregator, error) {

	if aggregator, ok := builtinAggregators[name]; ok {
		return aggregator, nil
	}
	if aggregator, ok := userAggregators[name]; ok {
		return aggregator, nil
	}
	return nil, userconfig.ErrorUndefinedResourceBuiltin(name, resource.AggregatorType)
}

func getAggregators(
	config *userconfig.Config,
	userAggregators map[string]*context.Aggregator,
) (context.Aggregators, error) {

	aggregators := context.Aggregators{}
	for _, aggregateConfig := range config.Aggregates {
		if _, ok := aggregators[aggregateConfig.Aggregator]; ok {
			continue
		}
		aggregator, err := getAggregator(aggregateConfig.Aggregator, userAggregators)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(aggregateConfig), userconfig.AggregatorKey)
		}
		aggregators[aggregateConfig.Aggregator] = aggregator
	}

	return aggregators, nil
}
