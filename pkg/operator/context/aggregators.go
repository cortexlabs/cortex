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
	"io/ioutil"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

var builtinAggregators = make(map[string]*context.Aggregator)
var uploadedAggregators = make(map[string]bool)

func init() {
	configPath := filepath.Join(OperatorAggregatorsDir, "aggregators.yaml")

	config, err := userconfig.NewPartialPath(configPath)
	if err != nil {
		errors.Exit(err)
	}

	for _, aggregatorConfig := range config.Aggregators {
		implPath := filepath.Join(OperatorAggregatorsDir, filepath.Base(aggregatorConfig.Path))
		impl, err := ioutil.ReadFile(implPath)
		if err != nil {
			errors.Exit(err, userconfig.Identify(aggregatorConfig), s.ErrReadFile(implPath))
		}
		aggregator, err := newAggregator(*aggregatorConfig, impl, util.StrPtr("cortex"), nil)
		if err != nil {
			errors.Exit(err)
		}
		builtinAggregators["cortex."+aggregatorConfig.Name] = aggregator
	}
}

func loadUserAggregators(
	aggregatorConfigs userconfig.Aggregators,
	impls map[string][]byte,
	pythonPackage context.PythonPackages,
) (map[string]*context.Aggregator, error) {

	userAggregators := make(map[string]*context.Aggregator)
	for _, aggregatorConfig := range aggregatorConfigs {
		impl, ok := impls[aggregatorConfig.Path]
		if !ok {
			return nil, errors.New(userconfig.Identify(aggregatorConfig), s.ErrFileDoesNotExist(aggregatorConfig.Path))
		}
		aggregator, err := newAggregator(*aggregatorConfig, impl, nil, &pythonPackage)
		if err != nil {
			return nil, err
		}
		userAggregators[aggregator.Name] = aggregator
	}

	return userAggregators, nil
}

func newAggregator(
	aggregatorConfig userconfig.Aggregator,
	impl []byte,
	namespace *string,
	pythonPackages *context.PythonPackages,
) (*context.Aggregator, error) {

	implID := util.HashBytes(impl)

	var buf bytes.Buffer
	buf.WriteString(context.DataTypeID(aggregatorConfig.Inputs))
	buf.WriteString(context.DataTypeID(aggregatorConfig.OutputType))
	buf.WriteString(implID)
	if pythonPackages != nil {
		for _, pythonPackage := range *pythonPackages {
			buf.WriteString(pythonPackage.GetID())
		}
	}
	id := util.HashBytes(buf.Bytes())

	aggregator := &context.Aggregator{
		ResourceFields: &context.ResourceFields{
			ID:           id,
			IDWithTags:   id,
			ResourceType: resource.AggregatorType,
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
	if _, ok := uploadedAggregators[aggregator.ID]; ok {
		return nil
	}

	isUploaded, err := aws.IsS3File(aggregator.ImplKey)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(aggregator), "upload")
	}

	if !isUploaded {
		err = aws.UploadBytesToS3(impl, aggregator.ImplKey)
		if err != nil {
			return errors.Wrap(err, userconfig.Identify(aggregator), "upload")
		}
	}

	uploadedAggregators[aggregator.ID] = true
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
		aggregatorName := aggregateConfig.Aggregator
		if _, ok := aggregators[aggregatorName]; ok {
			continue
		}
		aggregator, err := getAggregator(aggregatorName, userAggregators)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(aggregateConfig), userconfig.AggregatorKey)
		}
		aggregators[aggregatorName] = aggregator
	}

	return aggregators, nil
}
