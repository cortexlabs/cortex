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

package userconfig

import (
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type Config struct {
	App  *App `json:"app" yaml:"app"`
	APIs APIs `json:"apis" yaml:"apis"`
}

var typeFieldValidation = &cr.StructFieldValidation{
	Key: "kind",
	Nil: true,
}

func (config *Config) Validate(projectBytes []byte) error {
	err := config.App.Validate()

	if err != nil {
		return err
	}

	projectFileMap := make(map[string][]byte)

	if config.AreProjectFilesRequired() {
		projectFileMap, err = zip.UnzipMemToMem(projectBytes)
		if err != nil {
			return err
		}
	}

	if config.APIs != nil {
		if err := config.APIs.Validate(projectFileMap); err != nil {
			return err
		}
	}

	return nil
}

func (config *Config) AreProjectFilesRequired() bool {
	return config.APIs.AreProjectFilesRequired()
}

func New(filePath string, configBytes []byte) (*Config, error) {
	var err error

	configData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		return nil, errors.Wrap(err, filePath, ErrorParseConfig().Error())
	}

	configDataSlice, ok := cast.InterfaceToStrInterfaceMapSlice(configData)
	if !ok {
		return nil, errors.Wrap(ErrorMalformedConfig(), filePath)
	}

	config := &Config{}
	for i, data := range configDataSlice {
		kindInterface, ok := data[KindKey]
		if !ok {
			return nil, errors.Wrap(configreader.ErrorMustBeDefined(), identify(filePath, resource.UnknownType, "", i), KindKey)
		}
		kindStr, ok := kindInterface.(string)
		if !ok {
			return nil, errors.Wrap(configreader.ErrorInvalidPrimitiveType(kindInterface, configreader.PrimTypeString), identify(filePath, resource.UnknownType, "", i), KindKey)
		}

		var errs []error
		resourceType := resource.TypeFromKindString(kindStr)
		var newResource Resource
		switch resourceType {
		case resource.AppType:
			if config.App != nil {
				return nil, errors.Wrap(ErrorDuplicateConfig(resource.AppType), filePath)
			}
			app := &App{}
			errs = cr.Struct(app, data, appValidation)
			config.App = app
		case resource.APIType:
			newResource = &API{}
			errs = cr.Struct(newResource, data, apiValidation)
			if !errors.HasErrors(errs) {
				config.APIs = append(config.APIs, newResource.(*API))
			}
		default:
			return nil, errors.Wrap(resource.ErrorUnknownKind(kindStr), identify(filePath, resource.UnknownType, "", i))
		}

		if errors.HasErrors(errs) {
			name, _ := data[NameKey].(string)
			return nil, errors.Wrap(errors.FirstError(errs...), identify(filePath, resourceType, name, i))
		}

		if newResource != nil {
			newResource.SetIndex(i)
			newResource.SetFilePath(filePath)
		}
	}

	if config.App == nil {
		return nil, ErrorMissingAppDefinition()
	}

	return config, nil
}

func ReadConfigFile(filePath string, relativePath string) (*Config, error) {
	configBytes, err := files.ReadFileBytes(filePath)
	if err != nil {
		return nil, errors.Wrap(err, relativePath, ErrorReadConfig().Error())
	}

	config, err := New(relativePath, configBytes)
	if err != nil {
		return nil, err
	}

	return config, nil
}
