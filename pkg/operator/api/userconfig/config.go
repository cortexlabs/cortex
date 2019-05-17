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
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type Config struct {
	App                *App               `json:"app" yaml:"app"`
	Environments       Environments       `json:"environments" yaml:"environments"`
	Environment        *Environment       `json:"environment" yaml:"environment"`
	RawColumns         RawColumns         `json:"raw_columns" yaml:"raw_columns"`
	Aggregates         Aggregates         `json:"aggregates" yaml:"aggregates"`
	TransformedColumns TransformedColumns `json:"transformed_columns" yaml:"transformed_columns"`
	Models             Models             `json:"models" yaml:"models"`
	APIs               APIs               `json:"apis" yaml:"apis"`
	Aggregators        Aggregators        `json:"aggregators" yaml:"aggregators"`
	Transformers       Transformers       `json:"transformers" yaml:"transformers"`
	Constants          Constants          `json:"constants" yaml:"constants"`
	Templates          Templates          `json:"templates" yaml:"templates"`
	Embeds             Embeds             `json:"embeds" yaml:"embeds"`
}

var typeFieldValidation = &cr.StructFieldValidation{
	Key: "kind",
	Nil: true,
}

func mergeConfigs(target *Config, source *Config) error {
	target.Environments = append(target.Environments, source.Environments...)
	target.RawColumns = append(target.RawColumns, source.RawColumns...)
	target.Aggregates = append(target.Aggregates, source.Aggregates...)
	target.TransformedColumns = append(target.TransformedColumns, source.TransformedColumns...)
	target.Models = append(target.Models, source.Models...)
	target.APIs = append(target.APIs, source.APIs...)
	target.Aggregators = append(target.Aggregators, source.Aggregators...)
	target.Transformers = append(target.Transformers, source.Transformers...)
	target.Constants = append(target.Constants, source.Constants...)
	target.Templates = append(target.Templates, source.Templates...)
	target.Embeds = append(target.Embeds, source.Embeds...)

	if source.App != nil {
		if target.App != nil {
			return ErrorDuplicateConfig(resource.AppType)
		}
		target.App = source.App
	}

	return nil
}

func (config *Config) ValidatePartial() error {
	if config.App != nil {
		if err := config.App.Validate(); err != nil {
			return err
		}
	}
	if config.Environments != nil {
		if err := config.Environments.Validate(); err != nil {
			return err
		}
	}
	if config.RawColumns != nil {
		if err := config.RawColumns.Validate(); err != nil {
			return err
		}
	}
	if config.Aggregates != nil {
		if err := config.Aggregates.Validate(); err != nil {
			return err
		}
	}
	if config.TransformedColumns != nil {
		if err := config.TransformedColumns.Validate(); err != nil {
			return err
		}
	}
	if config.Models != nil {
		if err := config.Models.Validate(); err != nil {
			return err
		}
	}
	if config.APIs != nil {
		if err := config.APIs.Validate(); err != nil {
			return err
		}
	}
	if config.Aggregators != nil {
		if err := config.Aggregators.Validate(); err != nil {
			return err
		}
	}
	if config.Transformers != nil {
		if err := config.Transformers.Validate(); err != nil {
			return err
		}
	}
	if config.Constants != nil {
		if err := config.Constants.Validate(); err != nil {
			return err
		}
	}
	if config.Templates != nil {
		if err := config.Templates.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (config *Config) Validate(envName string) error {
	err := config.ValidatePartial()
	if err != nil {
		return err
	}

	if config.App == nil {
		return ErrorUndefinedConfig(resource.AppType)
	}

	err = config.ValidateColumns()
	if err != nil {
		return err
	}

	// Check ingested columns match raw columns
	rawColumnNames := config.RawColumns.Names()
	for _, env := range config.Environments {
		ingestedColumnNames := env.Data.GetIngestedColumns()
		missingColumns := slices.SubtractStrSlice(rawColumnNames, ingestedColumnNames)
		if len(missingColumns) > 0 {
			return errors.Wrap(ErrorRawColumnNotInEnv(env.Name), Identify(config.RawColumns.Get(missingColumns[0])))
		}
		extraColumns := slices.SubtractStrSlice(rawColumnNames, ingestedColumnNames)
		if len(extraColumns) > 0 {
			return errors.Wrap(ErrorUndefinedResource(extraColumns[0], resource.RawColumnType), Identify(env), DataKey, SchemaKey)
		}
	}

	// Check model columns exist
	columnNames := config.ColumnNames()
	for _, model := range config.Models {
		if !slices.HasString(columnNames, model.TargetColumn) {
			return errors.Wrap(ErrorUndefinedResource(model.TargetColumn, resource.RawColumnType, resource.TransformedColumnType),
				Identify(model), TargetColumnKey)
		}
		missingColumnNames := slices.SubtractStrSlice(model.FeatureColumns, columnNames)
		if len(missingColumnNames) > 0 {
			return errors.Wrap(ErrorUndefinedResource(missingColumnNames[0], resource.RawColumnType, resource.TransformedColumnType),
				Identify(model), FeatureColumnsKey)
		}

		missingAggregateNames := slices.SubtractStrSlice(model.Aggregates, config.Aggregates.Names())
		if len(missingAggregateNames) > 0 {
			return errors.Wrap(ErrorUndefinedResource(missingAggregateNames[0], resource.AggregateType),
				Identify(model), AggregatesKey)
		}

		// check training columns
		missingTrainingColumnNames := slices.SubtractStrSlice(model.TrainingColumns, columnNames)
		if len(missingTrainingColumnNames) > 0 {
			return errors.Wrap(ErrorUndefinedResource(missingTrainingColumnNames[0], resource.RawColumnType, resource.TransformedColumnType),
				Identify(model), TrainingColumnsKey)
		}
	}

	// Check api models exist
	modelNames := config.Models.Names()
	for _, api := range config.APIs {
		if !slices.HasString(modelNames, api.ModelName) {
			return errors.Wrap(ErrorUndefinedResource(api.ModelName, resource.ModelType),
				Identify(api), ModelNameKey)
		}
	}

	// Check local aggregators exist or a path to one is defined
	aggregatorNames := config.Aggregators.Names()
	for _, aggregate := range config.Aggregates {
		if aggregate.AggregatorPath == nil && aggregate.Aggregator == "" {
			return errors.Wrap(ErrorSpecifyOnlyOneMissing("aggregator", "aggregator_path"), Identify(aggregate))
		}

		if aggregate.AggregatorPath != nil && aggregate.Aggregator != "" {
			return errors.Wrap(ErrorSpecifyOnlyOne("aggregator", "aggregator_path"), Identify(aggregate))
		}

		if aggregate.Aggregator != "" &&
			!strings.Contains(aggregate.Aggregator, ".") &&
			!slices.HasString(aggregatorNames, aggregate.Aggregator) {
			return errors.Wrap(ErrorUndefinedResource(aggregate.Aggregator, resource.AggregatorType), Identify(aggregate), AggregatorKey)
		}
	}

	// Check local transformers exist or a path to one is defined
	transformerNames := config.Transformers.Names()
	for _, transformedColumn := range config.TransformedColumns {
		if transformedColumn.TransformerPath == nil && transformedColumn.Transformer == "" {
			return errors.Wrap(ErrorSpecifyOnlyOneMissing("transformer", "transformer_path"), Identify(transformedColumn))
		}

		if transformedColumn.TransformerPath != nil && transformedColumn.Transformer != "" {
			return errors.Wrap(ErrorSpecifyOnlyOne("transformer", "transformer_path"), Identify(transformedColumn))
		}

		if transformedColumn.Transformer != "" &&
			!strings.Contains(transformedColumn.Transformer, ".") &&
			!slices.HasString(transformerNames, transformedColumn.Transformer) {
			return errors.Wrap(ErrorUndefinedResource(transformedColumn.Transformer, resource.TransformerType), Identify(transformedColumn), TransformerKey)
		}
	}

	for _, env := range config.Environments {
		if env.Name == envName {
			config.Environment = env
		}
	}
	if config.Environment == nil {
		return ErrorUndefinedResource(envName, resource.EnvironmentType)
	}

	return nil
}

func (config *Config) MergeBytes(configBytes []byte, filePath string, emb *Embed, template *Template) (*Config, error) {
	sliceData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		if emb == nil {
			return nil, errors.Wrap(err, filePath)
		}
		return nil, errors.Wrap(err, Identify(template), YAMLKey)
	}

	subConfig, err := newPartial(sliceData, filePath, emb, template)
	if err != nil {
		return nil, err
	}

	err = mergeConfigs(config, subConfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func newPartial(configData interface{}, filePath string, emb *Embed, template *Template) (*Config, error) {
	configDataSlice, ok := cast.InterfaceToStrInterfaceMapSlice(configData)
	if !ok {
		if emb == nil {
			return nil, errors.Wrap(ErrorMalformedConfig(), filePath)
		}
		return nil, errors.Wrap(ErrorMalformedConfig(), Identify(template), YAMLKey)
	}

	config := &Config{}
	for i, data := range configDataSlice {
		kindInterface, ok := data[KindKey]
		if !ok {
			return nil, errors.Wrap(configreader.ErrorMustBeDefined(), identify(filePath, resource.UnknownType, "", i, emb), KindKey)
		}
		kindStr, ok := kindInterface.(string)
		if !ok {
			return nil, errors.Wrap(configreader.ErrorInvalidPrimitiveType(kindInterface, configreader.PrimTypeString), identify(filePath, resource.UnknownType, "", i, emb), KindKey)
		}

		var errs []error
		resourceType := resource.TypeFromKindString(kindStr)
		var newResource Resource
		switch resourceType {
		case resource.AppType:
			app := &App{}
			errs = cr.Struct(app, data, appValidation)
			config.App = app
		case resource.RawColumnType:
			var rawColumnInter interface{}
			rawColumnInter, errs = cr.InterfaceStruct(data, rawColumnValidation)
			if !errors.HasErrors(errs) && rawColumnInter != nil {
				newResource = rawColumnInter.(RawColumn)
				config.RawColumns = append(config.RawColumns, newResource.(RawColumn))
			}
		case resource.TransformedColumnType:
			newResource = &TransformedColumn{}
			errs = cr.Struct(newResource, data, transformedColumnValidation)
			if !errors.HasErrors(errs) {
				config.TransformedColumns = append(config.TransformedColumns, newResource.(*TransformedColumn))
			}
		case resource.AggregateType:
			newResource = &Aggregate{}
			errs = cr.Struct(newResource, data, aggregateValidation)
			if !errors.HasErrors(errs) {
				config.Aggregates = append(config.Aggregates, newResource.(*Aggregate))
			}
		case resource.ConstantType:
			newResource = &Constant{}
			errs = cr.Struct(newResource, data, constantValidation)
			if !errors.HasErrors(errs) {
				config.Constants = append(config.Constants, newResource.(*Constant))
			}
		case resource.APIType:
			newResource = &API{}
			errs = cr.Struct(newResource, data, apiValidation)
			if !errors.HasErrors(errs) {
				config.APIs = append(config.APIs, newResource.(*API))
			}
		case resource.ModelType:
			newResource = &Model{}
			errs = cr.Struct(newResource, data, modelValidation)
			if !errors.HasErrors(errs) {
				config.Models = append(config.Models, newResource.(*Model))
			}
		case resource.EnvironmentType:
			newResource = &Environment{}
			errs = cr.Struct(newResource, data, environmentValidation)
			if !errors.HasErrors(errs) {
				config.Environments = append(config.Environments, newResource.(*Environment))
			}
		case resource.AggregatorType:
			newResource = &Aggregator{}
			errs = cr.Struct(newResource, data, aggregatorValidation)
			if !errors.HasErrors(errs) {
				config.Aggregators = append(config.Aggregators, newResource.(*Aggregator))
			}
		case resource.TransformerType:
			newResource = &Transformer{}
			errs = cr.Struct(newResource, data, transformerValidation)
			if !errors.HasErrors(errs) {
				config.Transformers = append(config.Transformers, newResource.(*Transformer))
			}
		case resource.TemplateType:
			if emb != nil {
				errs = []error{resource.ErrorTemplateInTemplate()}
			} else {
				newResource = &Template{}
				errs = cr.Struct(newResource, data, templateValidation)
				if !errors.HasErrors(errs) {
					config.Templates = append(config.Templates, newResource.(*Template))
				}
			}
		case resource.EmbedType:
			if emb != nil {
				errs = []error{resource.ErrorEmbedInTemplate()}
			} else {
				newResource = &Embed{}
				errs = cr.Struct(newResource, data, embedValidation)
				if !errors.HasErrors(errs) {
					config.Embeds = append(config.Embeds, newResource.(*Embed))
				}
			}
		default:
			return nil, errors.Wrap(resource.ErrorUnknownKind(kindStr), identify(filePath, resource.UnknownType, "", i, emb))
		}

		if errors.HasErrors(errs) {
			name, _ := data[NameKey].(string)
			return nil, errors.Wrap(errors.FirstError(errs...), identify(filePath, resourceType, name, i, emb))
		}

		if newResource != nil {
			newResource.SetIndex(i)
			newResource.SetFilePath(filePath)
			newResource.SetEmbed(emb)
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
	return newPartial(configData, filePath, nil, nil)
}

func New(configs map[string][]byte, envName string) (*Config, error) {
	var err error
	config := &Config{}
	for filePath, configBytes := range configs {
		if !files.IsFilePathYAML(filePath) {
			continue
		}
		config, err = config.MergeBytes(configBytes, filePath, nil, nil)
		if err != nil {
			return nil, err
		}
	}

	templates := config.Templates.Map()
	for _, emb := range config.Embeds {
		template, ok := templates[emb.Template]
		if !ok {
			return nil, errors.Wrap(ErrorUndefinedResource(emb.Template, resource.TemplateType), Identify(emb))
		}

		populatedTemplate, err := template.Populate(emb)
		if err != nil {
			return nil, errors.Wrap(err, Identify(emb))
		}

		config, err = config.MergeBytes([]byte(populatedTemplate), emb.FilePath, emb, template)
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
