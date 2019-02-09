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

	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/utils/cast"
	cr "github.com/cortexlabs/cortex/pkg/utils/configreader"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type Config struct {
	App                 *App                `json:"app" yaml:"app"`
	Environments        Environments        `json:"environments" yaml:"environments"`
	Environment         *Environment        `json:"environment" yaml:"environment"`
	RawFeatures         RawFeatures         `json:"raw_features" yaml:"raw_features"`
	Aggregates          Aggregates          `json:"aggregates" yaml:"aggregates"`
	TransformedFeatures TransformedFeatures `json:"transformed_features" yaml:"transformed_features"`
	Models              Models              `json:"models" yaml:"models"`
	APIs                APIs                `json:"apis" yaml:"apis"`
	Aggregators         Aggregators         `json:"aggregators" yaml:"aggregators"`
	Transformers        Transformers        `json:"transformers" yaml:"transformers"`
	Constants           Constants           `json:"constants" yaml:"constants"`
	Templates           Templates           `json:"templates" yaml:"templates"`
	Embeds              Embeds              `json:"embeds" yaml:"embeds"`
}

type Resource interface {
	GetName() string
	GetResourceType() resource.Type
}

func Identify(r Resource) string {
	return fmt.Sprintf("%s: %s", r.GetResourceType().String(), r.GetName())
}

var typeFieldValidation = &cr.StructFieldValidation{
	Key: "kind",
	Nil: true,
}

func mergeConfigs(target *Config, source *Config) error {
	target.Environments = append(target.Environments, source.Environments...)
	target.RawFeatures = append(target.RawFeatures, source.RawFeatures...)
	target.Aggregates = append(target.Aggregates, source.Aggregates...)
	target.TransformedFeatures = append(target.TransformedFeatures, source.TransformedFeatures...)
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
	if config.RawFeatures != nil {
		if err := config.RawFeatures.Validate(); err != nil {
			return err
		}
	}
	if config.Aggregates != nil {
		if err := config.Aggregates.Validate(); err != nil {
			return err
		}
	}
	if config.TransformedFeatures != nil {
		if err := config.TransformedFeatures.Validate(); err != nil {
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

	err = config.ValidateFeatures()
	if err != nil {
		return err
	}

	// Check ingested features match raw features
	rawFeatureNames := config.RawFeatures.Names()
	for _, env := range config.Environments {
		ingestedFeatureNames := env.Data.GetIngestedFeatures()
		missingFeatures := util.SubtractStrSlice(rawFeatureNames, ingestedFeatureNames)
		if len(missingFeatures) > 0 {
			return errors.Wrap(ErrorMissingRawFeatures(missingFeatures), Identify(env), DataKey, SchemaKey)
		}
		extraFeatures := util.SubtractStrSlice(rawFeatureNames, ingestedFeatureNames)
		if len(extraFeatures) > 0 {
			return errors.Wrap(ErrorUndefinedResource(extraFeatures[0], resource.RawFeatureType), Identify(env), DataKey, SchemaKey)
		}
	}

	// Check model features exist
	featureNames := config.FeatureNames()
	for _, model := range config.Models {
		if !util.IsStrInSlice(model.Target, featureNames) {
			return errors.Wrap(ErrorUndefinedResource(model.Target, resource.RawFeatureType, resource.TransformedFeatureType),
				Identify(model), TargetKey)
		}
		missingFeatureNames := util.SubtractStrSlice(model.Features, featureNames)
		if len(missingFeatureNames) > 0 {
			return errors.Wrap(ErrorUndefinedResource(missingFeatureNames[0], resource.RawFeatureType, resource.TransformedFeatureType),
				Identify(model), FeaturesKey)
		}

		missingAggregateNames := util.SubtractStrSlice(model.Aggregates, config.Aggregates.Names())
		if len(missingAggregateNames) > 0 {
			return errors.Wrap(ErrorUndefinedResource(missingAggregateNames[0], resource.AggregateType),
				Identify(model), AggregatesKey)
		}

		// check training features
		missingTrainingFeatureNames := util.SubtractStrSlice(model.TrainingFeatures, featureNames)
		if len(missingTrainingFeatureNames) > 0 {
			return errors.Wrap(ErrorUndefinedResource(missingTrainingFeatureNames[0], resource.RawFeatureType, resource.TransformedFeatureType),
				Identify(model), TrainingFeaturesKey)
		}
	}

	// Check api models exist
	modelNames := config.Models.Names()
	for _, api := range config.APIs {
		if !util.IsStrInSlice(api.ModelName, modelNames) {
			return errors.Wrap(ErrorUndefinedResource(api.ModelName, resource.ModelType),
				Identify(api), ModelNameKey)
		}
	}

	// Check local aggregators exist
	aggregatorNames := config.Aggregators.Names()
	for _, aggregate := range config.Aggregates {
		if !strings.Contains(aggregate.Aggregator, ".") && !util.IsStrInSlice(aggregate.Aggregator, aggregatorNames) {
			return errors.Wrap(ErrorUndefinedResource(aggregate.Aggregator, resource.AggregatorType), Identify(aggregate), AggregatorKey)
		}
	}

	// Check local transformers exist
	transformerNames := config.Transformers.Names()
	for _, transformedFeature := range config.TransformedFeatures {
		if !strings.Contains(transformedFeature.Transformer, ".") && !util.IsStrInSlice(transformedFeature.Transformer, transformerNames) {
			return errors.Wrap(ErrorUndefinedResource(transformedFeature.Transformer, resource.TransformerType), Identify(transformedFeature), TransformerKey)
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

func (config *Config) MergeBytes(configBytes []byte) (*Config, error) {
	sliceData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		return nil, err
	}

	subConfig, err := newPartial(sliceData)
	if err != nil {
		return nil, err
	}

	err = mergeConfigs(config, subConfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func newPartial(configData interface{}) (*Config, error) {
	configDataSlice, ok := cast.InterfaceToStrInterfaceMapSlice(configData)
	if !ok {
		return nil, ErrorMalformedConfig()
	}

	config := &Config{}
	for i, data := range configDataSlice {
		kindStr, ok := data[KindKey].(string)
		if !ok {
			return nil, errors.New("resource at "+s.Index(i), KindKey, s.ErrMustBeDefined)
		}

		var errs []error
		switch resource.TypeFromKindString(kindStr) {
		case resource.AppType:
			app := &App{}
			errs = cr.Struct(app, data, appValidation)
			config.App = app
		case resource.RawFeatureType:
			var rawFeatureInter interface{}
			rawFeatureInter, errs = cr.InterfaceStruct(data, rawFeatureValidation)
			if rawFeatureInter != nil {
				config.RawFeatures = append(config.RawFeatures, rawFeatureInter.(RawFeature))
			}
		case resource.TransformedFeatureType:
			transformedFeature := &TransformedFeature{}
			errs = cr.Struct(transformedFeature, data, transformedFeatureValidation)
			config.TransformedFeatures = append(config.TransformedFeatures, transformedFeature)
		case resource.AggregateType:
			aggregate := &Aggregate{}
			errs = cr.Struct(aggregate, data, aggregateValidation)
			config.Aggregates = append(config.Aggregates, aggregate)
		case resource.ConstantType:
			constant := &Constant{}
			errs = cr.Struct(constant, data, constantValidation)
			config.Constants = append(config.Constants, constant)
		case resource.APIType:
			api := &API{}
			errs = cr.Struct(api, data, apiValidation)
			config.APIs = append(config.APIs, api)
		case resource.ModelType:
			model := &Model{}
			errs = cr.Struct(model, data, modelValidation)
			config.Models = append(config.Models, model)
		case resource.EnvironmentType:
			environment := &Environment{}
			errs = cr.Struct(environment, data, environmentValidation)
			config.Environments = append(config.Environments, environment)
		case resource.AggregatorType:
			aggregator := &Aggregator{}
			errs = cr.Struct(aggregator, data, aggregatorValidation)
			config.Aggregators = append(config.Aggregators, aggregator)
		case resource.TransformerType:
			transformer := &Transformer{}
			errs = cr.Struct(transformer, data, transformerValidation)
			config.Transformers = append(config.Transformers, transformer)
		case resource.TemplateType:
			template := &Template{}
			errs = cr.Struct(template, data, templateValidation)
			config.Templates = append(config.Templates, template)
		case resource.EmbedType:
			embed := &Embed{}
			errs = cr.Struct(embed, data, embedValidation)
			config.Embeds = append(config.Embeds, embed)
		default:
			return nil, errors.Wrap(resource.ErrorUnknownKind(kindStr), "resource at "+s.Index(i))
		}

		if errors.HasErrors(errs) {
			wrapStr := fmt.Sprintf("%s at %s", kindStr, s.Index(i))
			if resourceName, ok := data[NameKey].(string); ok {
				wrapStr = fmt.Sprintf("%s: %s", kindStr, resourceName)
			}
			return nil, errors.Wrap(errors.FirstError(errs...), wrapStr)
		}
	}

	err := config.ValidatePartial()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func NewPartialPath(configPath string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, configPath, ErrorReadConfig().Error())
	}

	configData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		return nil, errors.Wrap(err, configPath, ErrorParseConfig().Error())
	}
	return newPartial(configData)
}

func New(configs map[string][]byte, envName string) (*Config, error) {
	var err error
	config := &Config{}
	for configPath, configBytes := range configs {
		if !util.IsFilePathYAML(configPath) {
			continue
		}
		config, err = config.MergeBytes(configBytes)
		if err != nil {
			return nil, errors.Wrap(err, configPath)
		}
	}

	templates := config.Templates.Map()
	for _, emb := range config.Embeds {
		template, ok := templates[emb.Template]
		if !ok {
			return nil, errors.Wrap(ErrorUndefinedResource(emb.Template, resource.TemplateType), resource.EmbedType.String())
		}

		populatedTemplate, err := template.Populate(emb)
		if err != nil {
			return nil, errors.Wrap(err, resource.EmbedType.String())
		}

		config, err = config.MergeBytes([]byte(populatedTemplate))
		if err != nil {
			return nil, errors.Wrap(err, resource.EmbedType.String(), fmt.Sprintf("%s %s", resource.TemplateType.String(), s.UserStr(template.Name)))
		}
	}

	if err := config.Validate(envName); err != nil {
		return nil, err
	}
	return config, nil
}

func ReadAppName(configPath string) (string, error) {
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return "", errors.Wrap(err, ErrorReadConfig().Error(), configPath)
	}
	configData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		return "", errors.Wrap(err, ErrorParseConfig().Error(), configPath)
	}
	configDataSlice, ok := cast.InterfaceToStrInterfaceMapSlice(configData)
	if !ok {
		return "", errors.Wrap(ErrorMalformedConfig(), configPath)
	}

	if len(configDataSlice) == 0 {
		return "", errors.Wrap(ErrorMissingAppDefinition(), configPath)
	}

	var appName string
	for i, configItem := range configDataSlice {
		kindStr, _ := configItem[KindKey].(string)
		if resource.TypeFromString(kindStr) == resource.AppType {
			if appName != "" {
				return "", errors.Wrap(ErrorDuplicateConfig(resource.AppType), configPath)
			}

			wrapStr := fmt.Sprintf("%s at %s", resource.AppType.String(), s.Index(i))

			appNameInter, ok := configItem[NameKey]
			if !ok {
				return "", errors.New(configPath, wrapStr, NameKey, s.ErrMustBeDefined)
			}

			appName, ok = appNameInter.(string)
			if !ok {
				return "", errors.New(configPath, wrapStr, s.ErrInvalidPrimitiveType(appNameInter, s.PrimTypeString))
			}
			if appName == "" {
				return "", errors.New(configPath, wrapStr, s.ErrCannotBeEmpty)
			}
		}
	}

	if appName == "" {
		return "", errors.Wrap(ErrorMissingAppDefinition(), configPath)
	}

	return appName, nil
}
