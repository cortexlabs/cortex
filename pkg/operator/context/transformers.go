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

func loadUserTransformers(
	config *userconfig.Config,
	impls map[string][]byte,
	pythonPackages context.PythonPackages,
) (map[string]*context.Transformer, error) {

	userTransformers := make(map[string]*context.Transformer)
	for _, transConfig := range config.Transformers {
		impl, ok := impls[transConfig.Path]
		if !ok {
			return nil, errors.Wrap(ErrorImplDoesNotExist(transConfig.Path), userconfig.Identify(transConfig))
		}
		transformer, err := newTransformer(*transConfig, impl, nil, pythonPackages)
		if err != nil {
			return nil, err
		}
		userTransformers[transformer.Name] = transformer
	}

	for _, transColConfig := range config.TransformedColumns {
		if transColConfig.TransformerPath == nil {
			continue
		}

		impl, ok := impls[*transColConfig.TransformerPath]
		if !ok {
			return nil, errors.Wrap(ErrorImplDoesNotExist(*transColConfig.TransformerPath), userconfig.Identify(transColConfig))
		}

		implHash := hash.Bytes(impl)
		if _, ok := userTransformers[implHash]; ok {
			continue
		}

		anonTransformerConfig := &userconfig.Transformer{
			ResourceFields: userconfig.ResourceFields{
				Name: implHash,
			},
			Path: *transColConfig.TransformerPath,
		}
		transformer, err := newTransformer(*anonTransformerConfig, impl, nil, pythonPackages)
		if err != nil {
			return nil, err
		}
		transColConfig.Transformer = transformer.Name
		userTransformers[transformer.Name] = transformer
	}
	return userTransformers, nil
}

func newTransformer(
	transConfig userconfig.Transformer,
	impl []byte,
	namespace *string,
	pythonPackages context.PythonPackages,
) (*context.Transformer, error) {

	implID := hash.Bytes(impl)

	var buf bytes.Buffer
	buf.WriteString(context.DataTypeID(transConfig.Inputs))
	buf.WriteString(context.DataTypeID(transConfig.OutputType))
	buf.WriteString(implID)
	for _, pythonPackage := range pythonPackages {
		buf.WriteString(pythonPackage.GetID())
	}

	id := hash.Bytes(buf.Bytes())

	transformer := &context.Transformer{
		ResourceFields: &context.ResourceFields{
			ID:           id,
			IDWithTags:   id,
			ResourceType: resource.TransformerType,
			MetadataKey:  filepath.Join(consts.TransformersDir, id+"_metadata.json"),
		},
		Transformer: &transConfig,
		Namespace:   namespace,
		ImplKey:     filepath.Join(consts.TransformersDir, implID+".py"),
	}
	transformer.Transformer.Path = ""

	if err := uploadTransformer(transformer, impl); err != nil {
		return nil, err
	}

	return transformer, nil
}

func uploadTransformer(transformer *context.Transformer, impl []byte) error {
	if uploadedTransformers.Has(transformer.ID) {
		return nil
	}

	isUploaded, err := config.AWS.IsS3File(transformer.ImplKey)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(transformer), "upload")
	}

	if !isUploaded {
		err = config.AWS.UploadBytesToS3(impl, transformer.ImplKey)
		if err != nil {
			return errors.Wrap(err, userconfig.Identify(transformer), "upload")
		}
	}

	uploadedTransformers.Add(transformer.ID)
	return nil
}

func getTransformer(
	name string,
	userTransformers map[string]*context.Transformer,
) (*context.Transformer, error) {
	if transformer, ok := builtinTransformers[name]; ok {
		return transformer, nil
	}
	if transformer, ok := userTransformers[name]; ok {
		return transformer, nil
	}
	return nil, userconfig.ErrorUndefinedResourceBuiltin(name, resource.TransformerType)
}

func getTransformers(
	config *userconfig.Config,
	userTransformers map[string]*context.Transformer,
) (context.Transformers, error) {

	transformers := context.Transformers{}
	for _, transformedColumnConfig := range config.TransformedColumns {
		if _, ok := transformers[transformedColumnConfig.Transformer]; ok {
			continue
		}

		transformer, err := getTransformer(transformedColumnConfig.Transformer, userTransformers)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(transformedColumnConfig), userconfig.TransformerKey)
		}
		transformers[transformedColumnConfig.Transformer] = transformer
	}

	return transformers, nil
}
