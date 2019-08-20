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
	"fmt"
	"io/ioutil"

	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type Config struct {
	App       *App `json:"app" yaml:"app"`
	APIs      APIs `json:"apis" yaml:"apis"`
	Resources map[string][]Resource
}

var typeFieldValidation = &cr.StructFieldValidation{
	Key: "kind",
	Nil: true,
}

func mergeConfigs(target *Config, source *Config) error {
	target.APIs = append(target.APIs, source.APIs...)

	if source.App != nil {
		if target.App != nil {
			return ErrorDuplicateConfig(resource.AppType)
		}
		target.App = source.App
	}

	if target.Resources == nil {
		target.Resources = make(map[string][]Resource)
	}
	for resourceName, resources := range source.Resources {
		for _, res := range resources {
			target.Resources[resourceName] = append(target.Resources[resourceName], res)
		}
	}

	return nil
}

func (config *Config) ValidatePartial() error {
	if config.App != nil {
		if err := config.App.Validate(); err != nil {
			return err
		}
	}
	if config.APIs != nil {
		if err := config.APIs.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (config *Config) Validate(envName string) error {
	if config.App == nil {
		return ErrorMissingAppDefinition()
	}

	return nil
}

func (config *Config) MergeBytes(configBytes []byte, filePath string) (*Config, error) {
	sliceData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}

	subConfig, err := newPartial(sliceData, filePath)
	if err != nil {
		return nil, err
	}

	err = mergeConfigs(config, subConfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func newPartial(configData interface{}, filePath string) (*Config, error) {
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
			if config.Resources == nil {
				config.Resources = make(map[string][]Resource)
			}
			config.Resources[newResource.GetName()] = append(config.Resources[newResource.GetName()], newResource)
		}
	}

	err := config.ValidatePartial()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func NewPartialPath(filePath string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err, filePath, ErrorReadConfig().Error())
	}

	configData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		return nil, errors.Wrap(err, filePath, ErrorParseConfig().Error())
	}
	return newPartial(configData, filePath)
}

func New(configs map[string][]byte, envName string) (*Config, error) {
	var err error
	config := &Config{}
	for filePath, configBytes := range configs {
		if !files.IsFilePathYAML(filePath) {
			continue
		}
		config, err = config.MergeBytes(configBytes, filePath)
		if err != nil {
			return nil, err
		}
	}

	if err := config.Validate(envName); err != nil {
		return nil, err
	}
	return config, nil
}

func ReadAppName(filePath string, relativePath string) (string, error) {
	configBytes, err := files.ReadFileBytes(filePath)
	if err != nil {
		return "", errors.Wrap(err, ErrorReadConfig().Error(), relativePath)
	}
	configData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		return "", errors.Wrap(err, ErrorParseConfig().Error(), relativePath)
	}
	configDataSlice, ok := cast.InterfaceToStrInterfaceMapSlice(configData)
	if !ok {
		return "", errors.Wrap(ErrorMalformedConfig(), relativePath)
	}

	if len(configDataSlice) == 0 {
		return "", errors.Wrap(ErrorMissingAppDefinition(), relativePath)
	}

	var appName string
	for i, configItem := range configDataSlice {
		kindStr, _ := configItem[KindKey].(string)
		if resource.TypeFromKindString(kindStr) == resource.AppType {
			if appName != "" {
				return "", errors.Wrap(ErrorDuplicateConfig(resource.AppType), relativePath)
			}

			wrapStr := fmt.Sprintf("%s at %s", resource.AppType.String(), s.Index(i))

			appNameInter, ok := configItem[NameKey]
			if !ok {
				return "", errors.Wrap(configreader.ErrorMustBeDefined(), relativePath, wrapStr, NameKey)
			}

			appName, ok = appNameInter.(string)
			if !ok {
				return "", errors.Wrap(configreader.ErrorInvalidPrimitiveType(appNameInter, configreader.PrimTypeString), relativePath, wrapStr)
			}
			if appName == "" {
				return "", errors.Wrap(configreader.ErrorCannotBeEmpty(), relativePath, wrapStr)
			}
		}
	}

	if appName == "" {
		return "", errors.Wrap(ErrorMissingAppDefinition(), relativePath)
	}

	return appName, nil
}
