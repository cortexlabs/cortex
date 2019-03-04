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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	"github.com/cortexlabs/cortex/pkg/operator/telemetry"
)

var builtinTransformers = make(map[string]*context.Transformer)
var uploadedTransformers = strset.New()

func init() {
	configPath := filepath.Join(OperatorTransformersDir, "transformers.yaml")

	config, err := userconfig.NewPartialPath(configPath)
	if err != nil {
		telemetry.ReportErrorBlocking(err)
		errors.Exit(err)
	}

	for _, transConfig := range config.Transformers {
		implPath := filepath.Join(OperatorTransformersDir, transConfig.Path)
		impl, err := ioutil.ReadFile(implPath)
		if err != nil {
			err = errors.Wrap(err, userconfig.Identify(transConfig), s.ErrReadFile(implPath))
			telemetry.ReportErrorBlocking(err)
			errors.Exit(err)
		}
		transformer, err := newTransformer(*transConfig, impl, pointer.String("cortex"), nil)
		if err != nil {
			telemetry.ReportErrorBlocking(err)
			errors.Exit(err)
		}
		builtinTransformers["cortex."+transConfig.Name] = transformer
	}
}

func loadUserTransformers(
	transConfigs userconfig.Transformers,
	impls map[string][]byte,
	pythonPackages context.PythonPackages,
) (map[string]*context.Transformer, error) {

	userTransformers := make(map[string]*context.Transformer)
	for _, transConfig := range transConfigs {
		impl, ok := impls[transConfig.Path]
		if !ok {
			return nil, errors.New(userconfig.Identify(transConfig), s.ErrFileDoesNotExist(transConfig.Path))
		}
		transformer, err := newTransformer(*transConfig, impl, nil, pythonPackages)
		if err != nil {
			return nil, err
		}
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

	isUploaded, err := aws.IsS3File(transformer.ImplKey)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(transformer), "upload")
	}

	if !isUploaded {
		err = aws.UploadBytesToS3(impl, transformer.ImplKey)
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
		transformerName := transformedColumnConfig.Transformer
		if _, ok := transformers[transformerName]; ok {
			continue
		}
		transformer, err := getTransformer(transformerName, userTransformers)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(transformedColumnConfig), userconfig.TransformerKey)
		}
		transformers[transformerName] = transformer
	}

	return transformers, nil
}
