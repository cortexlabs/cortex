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

func loadUserEstimators(
	config *userconfig.Config,
	impls map[string][]byte,
	pythonPackages context.PythonPackages,
) (map[string]*context.Estimator, error) {

	userEstimators := make(map[string]*context.Estimator)
	for _, estimatorConfig := range config.Estimators {
		impl, ok := impls[estimatorConfig.Path]
		if !ok {
			return nil, errors.Wrap(userconfig.ErrorImplDoesNotExist(estimatorConfig.Path), userconfig.Identify(estimatorConfig))
		}
		estimator, err := newEstimator(*estimatorConfig, impl, nil, pythonPackages)
		if err != nil {
			return nil, err
		}
		userEstimators[estimator.Name] = estimator
	}

	for _, modelConfig := range config.Models {
		if modelConfig.EstimatorPath == nil {
			continue
		}

		impl, ok := impls[*modelConfig.EstimatorPath]
		if !ok {
			return nil, errors.Wrap(userconfig.ErrorImplDoesNotExist(*modelConfig.EstimatorPath), userconfig.Identify(modelConfig))
		}

		implHash := hash.Bytes(impl)
		if _, ok := userEstimators[implHash]; ok {
			continue
		}

		anonEstimatorConfig := &userconfig.Estimator{
			ResourceFields: userconfig.ResourceFields{
				Name: implHash,
			},
			TargetColumn:  userconfig.InferredColumnType,
			PredictionKey: modelConfig.PredictionKey,
			Path:          *modelConfig.EstimatorPath,
		}
		estimator, err := newEstimator(*anonEstimatorConfig, impl, nil, pythonPackages)
		if err != nil {
			return nil, err
		}

		modelConfig.Estimator = estimator.Name
		userEstimators[anonEstimatorConfig.Name] = estimator
	}

	return userEstimators, nil
}

func newEstimator(
	estimatorConfig userconfig.Estimator,
	impl []byte,
	namespace *string,
	pythonPackages context.PythonPackages,
) (*context.Estimator, error) {

	implID := hash.Bytes(impl)

	var buf bytes.Buffer
	buf.WriteString(context.DataTypeID(estimatorConfig.TargetColumn))
	buf.WriteString(context.DataTypeID(estimatorConfig.Input))
	buf.WriteString(context.DataTypeID(estimatorConfig.TrainingInput))
	buf.WriteString(context.DataTypeID(estimatorConfig.Hparams))
	buf.WriteString(estimatorConfig.PredictionKey)
	buf.WriteString(implID)

	for _, pythonPackage := range pythonPackages {
		buf.WriteString(pythonPackage.GetID())
	}

	id := hash.Bytes(buf.Bytes())

	estimator := &context.Estimator{
		ResourceFields: &context.ResourceFields{
			ID:           id,
			ResourceType: resource.EstimatorType,
		},
		Estimator: &estimatorConfig,
		Namespace: namespace,
		ImplKey:   filepath.Join(consts.EstimatorsDir, implID+".py"),
	}

	if err := uploadEstimator(estimator, impl); err != nil {
		return nil, err
	}

	return estimator, nil
}

func uploadEstimator(estimator *context.Estimator, impl []byte) error {
	if uploadedEstimators.Has(estimator.ID) {
		return nil
	}

	isUploaded, err := config.AWS.IsS3File(estimator.ImplKey)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(estimator), "upload")
	}

	if !isUploaded {
		err = config.AWS.UploadBytesToS3(impl, estimator.ImplKey)
		if err != nil {
			return errors.Wrap(err, userconfig.Identify(estimator), "upload")
		}
	}

	uploadedEstimators.Add(estimator.ID)
	return nil
}

func getEstimators(
	config *userconfig.Config,
	userEstimators map[string]*context.Estimator,
) (context.Estimators, error) {

	estimators := context.Estimators{}
	for _, modelConfig := range config.Models {
		name := modelConfig.Estimator

		if _, ok := estimators[name]; ok {
			continue
		}

		if estimator, ok := builtinEstimators[name]; ok {
			estimators[name] = estimator
			continue
		}

		if estimator, ok := userEstimators[name]; ok {
			estimators[name] = estimator
			continue
		}

		return nil, errors.Wrap(userconfig.ErrorUndefinedResource(name, resource.EstimatorType), userconfig.Identify(modelConfig), userconfig.EstimatorKey)
	}

	return estimators, nil
}
