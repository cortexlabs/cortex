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
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

var uploadedConstants = strset.New()

func loadConstants(constantConfigs userconfig.Constants) (context.Constants, error) {
	constants := context.Constants{}
	for _, constantConfig := range constantConfigs {
		constant, err := newConstant(*constantConfig)
		if err != nil {
			return nil, err
		}
		constants[constant.Name] = constant
	}

	return constants, nil
}

func newConstant(constantConfig userconfig.Constant) (*context.Constant, error) {
	var buf bytes.Buffer
	buf.WriteString(context.DataTypeID(constantConfig.Type))
	buf.WriteString(s.Obj(constantConfig.Value))
	id := hash.Bytes(buf.Bytes())
	idWithTags := hash.String(id + constantConfig.Tags.ID())

	constant := &context.Constant{
		ResourceFields: &context.ResourceFields{
			ID:           id,
			IDWithTags:   idWithTags,
			ResourceType: resource.ConstantType,
			MetadataKey:  filepath.Join(consts.ConstantsDir, id+"_metadata.json"),
		},
		Constant: &constantConfig,
		Key:      filepath.Join(consts.ConstantsDir, id+".msgpack"),
	}

	if err := uploadConstant(constant); err != nil {
		return nil, err
	}

	constant.Constant.Value = nil
	return constant, nil
}

func uploadConstant(constant *context.Constant) error {
	if uploadedConstants.Has(constant.ID) {
		return nil
	}

	isUploaded, err := config.AWS.IsS3File(constant.Key)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(constant), "upload")
	}

	if !isUploaded {
		serializedConstant := msgpack.MustMarshal(constant.Value)
		err = config.AWS.UploadBytesToS3(serializedConstant, constant.Key)
		if err != nil {
			return errors.Wrap(err, userconfig.Identify(constant), "upload")
		}
	}

	uploadedConstants.Add(constant.ID)
	return nil
}
