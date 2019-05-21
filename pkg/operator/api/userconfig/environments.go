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
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type Environments []*Environment

type Environment struct {
	ResourceFields
	LogLevel *LogLevel `json:"log_level" yaml:"log_level"`
	Limit    *Limit    `json:"limit" yaml:"limit"`
	Data     Data      `json:"-" yaml:"-"`
}

var environmentValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required:                   true,
				AlphaNumericDashUnderscore: true,
			},
		},
		{
			StructField:      "LogLevel",
			StructValidation: logLevelValidation,
		},
		{
			StructField:      "Limit",
			StructValidation: limitValidation,
		},
		{
			StructField:               "Data",
			Key:                       "data",
			InterfaceStructValidation: dataValidation,
		},
		typeFieldValidation,
	},
}

type Limit struct {
	NumRows        *int64   `json:"num_rows" yaml:"num_rows"`
	FractionOfRows *float32 `json:"fraction_of_rows" yaml:"fraction_of_rows"`
	Randomize      *bool    `json:"randomize" yaml:"randomize"`
	RandomSeed     *int64   `json:"random_seed" yaml:"random_seed"`
}

var limitValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "NumRows",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: pointer.Int64(0),
			},
		},
		{
			StructField: "FractionOfRows",
			Float32PtrValidation: &cr.Float32PtrValidation{
				GreaterThan: pointer.Float32(0),
				LessThan:    pointer.Float32(1),
			},
		},
		{
			StructField:       "Randomize",
			BoolPtrValidation: &cr.BoolPtrValidation{},
		},
		{
			StructField:        "RandomSeed",
			Int64PtrValidation: &cr.Int64PtrValidation{},
		},
	},
}

type LogLevel struct {
	Tensorflow string `json:"tensorflow" yaml:"tensorflow"`
	Spark      string `json:"spark" yaml:"spark"`
}

var logLevelValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Tensorflow",
			StringValidation: &cr.StringValidation{
				Default:       "DEBUG",
				AllowedValues: []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
			},
		},
		{
			StructField: "Spark",
			StringValidation: &cr.StringValidation{
				Default:       "WARN",
				AllowedValues: []string{"ALL", "TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
			},
		},
	},
}

type Data interface {
	GetIngestedColumns() []string
	GetExternalPath() string
	Validate() error
}

var dataValidation = &cr.InterfaceStructValidation{
	TypeKey:         "type",
	TypeStructField: "Type",
	ParsedInterfaceStructTypes: map[interface{}]*cr.InterfaceStructType{
		CSVEnvironmentDataType: {
			Type:                   (*CSVData)(nil),
			StructFieldValidations: csvDataFieldValidations,
		},
		ParquetEnvironmentDataType: {
			Type:                   (*ParquetData)(nil),
			StructFieldValidations: parquetDataFieldValidations,
		},
	},
	Parser: func(str string) (interface{}, error) {
		return EnvironmentDataTypeFromString(str), nil
	},
}

type CSVData struct {
	Type      EnvironmentDataType `json:"type" yaml:"type"`
	Path      string              `json:"path" yaml:"path"`
	Schema    []string            `json:"schema" yaml:"schema"`
	DropNull  bool                `json:"drop_null" yaml:"drop_null"`
	CSVConfig *CSVConfig          `json:"csv_config" yaml:"csv_config"`
}

// CSVConfig is SPARK_VERSION dependent
type CSVConfig struct {
	Sep                       *string `json:"sep" yaml:"sep"`
	Encoding                  *string `json:"encoding" yaml:"encoding"`
	Quote                     *string `json:"quote" yaml:"quote"`
	Escape                    *string `json:"escape" yaml:"escape"`
	Comment                   *string `json:"comment" yaml:"comment"`
	Header                    *bool   `json:"header" yaml:"header"`
	IgnoreLeadingWhiteSpace   *bool   `json:"ignore_leading_white_space" yaml:"ignore_leading_white_space"`
	IgnoreTrailingWhiteSpace  *bool   `json:"ignore_trailing_white_space" yaml:"ignore_trailing_white_space"`
	NullValue                 *string `json:"null_value" yaml:"null_value"`
	NanValue                  *string `json:"nan_value" yaml:"nan_value"`
	PositiveInf               *string `json:"positive_inf" yaml:"positive_inf"`
	NegativeInf               *string `json:"negative_inf" yaml:"negative_inf"`
	MaxColumns                *int32  `json:"max_columns" yaml:"max_columns"`
	MaxCharsPerColumn         *int32  `json:"max_chars_per_column" yaml:"max_chars_per_column"`
	Multiline                 *bool   `json:"multiline" yaml:"multiline"`
	CharToEscapeQuoteEscaping *string `json:"char_to_escape_quote_escaping" yaml:"char_to_escape_quote_escaping"`
	EmptyValue                *string `json:"empty_value" yaml:"empty_value"`
}

var csvDataFieldValidations = []*cr.StructFieldValidation{
	{
		StructField: "Path",
		StringValidation: cr.GetS3aPathValidation(&cr.S3aPathValidation{
			Required: true,
		}),
	},
	{
		StructField: "Schema",
		StringListValidation: &cr.StringListValidation{
			Required: true,
		},
	},
	{
		StructField: "DropNull",
		BoolValidation: &cr.BoolValidation{
			Default: false,
		},
	},
	{
		StructField: "CSVConfig",
		StructValidation: &cr.StructValidation{
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField:         "Sep",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				{
					StructField:         "Encoding",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				{
					StructField:         "Quote",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				{
					StructField:         "Escape",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				{
					StructField:         "Comment",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				{
					StructField:       "Header",
					BoolPtrValidation: &cr.BoolPtrValidation{},
				},
				{
					StructField:       "IgnoreLeadingWhiteSpace",
					BoolPtrValidation: &cr.BoolPtrValidation{},
				},
				{
					StructField:       "IgnoreTrailingWhiteSpace",
					BoolPtrValidation: &cr.BoolPtrValidation{},
				},
				{
					StructField:         "NullValue",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				{
					StructField:         "NanValue",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				{
					StructField:         "PositiveInf",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				{
					StructField:         "NegativeInf",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				{
					StructField: "MaxColumns",
					Int32PtrValidation: &cr.Int32PtrValidation{
						GreaterThan: pointer.Int32(0),
					},
				},
				{
					StructField: "MaxCharsPerColumn",
					Int32PtrValidation: &cr.Int32PtrValidation{
						GreaterThanOrEqualTo: pointer.Int32(-1),
					},
				},
				{
					StructField:       "Multiline",
					BoolPtrValidation: &cr.BoolPtrValidation{},
				},
				{
					StructField:         "CharToEscapeQuoteEscaping",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				{
					StructField:         "EmptyValue",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
			},
		},
	},
}

type ParquetData struct {
	Type     EnvironmentDataType `json:"type" yaml:"type"`
	Path     string              `json:"path" yaml:"path"`
	Schema   []*ParquetColumn    `json:"schema" yaml:"schema"`
	DropNull bool                `json:"drop_null" yaml:"drop_null"`
}

var parquetDataFieldValidations = []*cr.StructFieldValidation{
	{
		StructField: "Path",
		StringValidation: cr.GetS3aPathValidation(&cr.S3aPathValidation{
			Required: true,
		}),
	},
	{
		StructField: "Schema",
		StructListValidation: &cr.StructListValidation{
			StructValidation: parquetColumnValidation,
		},
	},
	{
		StructField: "DropNull",
		BoolValidation: &cr.BoolValidation{
			Default: false,
		},
	},
}

type ParquetColumn struct {
	ParquetColumnName string `json:"parquet_column_name" yaml:"parquet_column_name"`
	RawColumnName     string `json:"raw_column_name" yaml:"raw_column_name"`
}

var parquetColumnValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "ParquetColumnName",
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		{
			StructField: "RawColumnName",
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
	},
}

func (environments Environments) Validate() error {
	for _, env := range environments {
		if err := env.Validate(); err != nil {
			return err
		}
	}

	resources := make([]Resource, len(environments))
	for i, res := range environments {
		resources[i] = res
	}

	dups := FindDuplicateResourceName(resources...)
	if len(dups) > 0 {
		return ErrorDuplicateResourceName(dups...)
	}

	return nil
}

func (env *Environment) Validate() error {
	if err := env.Data.Validate(); err != nil {
		return errors.Wrap(err, Identify(env))
	}

	if env.Limit != nil {
		if env.Limit.NumRows != nil && env.Limit.FractionOfRows != nil {
			return errors.Wrap(ErrorSpecifyOnlyOne(NumRowsKey, FractionOfRowsKey), Identify(env), LimitKey)
		}
		if env.Limit.Randomize != nil && env.Limit.NumRows == nil && env.Limit.FractionOfRows == nil {
			return errors.Wrap(ErrorOneOfPrerequisitesNotDefined(RandomizeKey, LimitKey, FractionOfRowsKey), Identify(env))
		}
		if env.Limit.RandomSeed != nil && env.Limit.Randomize == nil {
			return errors.Wrap(ErrorOneOfPrerequisitesNotDefined(RandomSeedKey, RandomizeKey), Identify(env))
		}
	}

	dups := slices.FindDuplicateStrs(env.Data.GetIngestedColumns())
	if len(dups) > 0 {
		return errors.Wrap(configreader.ErrorDuplicatedValue(dups[0]), Identify(env), DataKey, SchemaKey, "column name")
	}

	return nil
}

func (csvData *CSVData) Validate() error {
	return nil
}

func (parqData *ParquetData) Validate() error {
	return nil
}

func (csvData *CSVData) GetExternalPath() string {
	return csvData.Path
}

func (parqData *ParquetData) GetExternalPath() string {
	return parqData.Path
}

func (csvData *CSVData) GetIngestedColumns() []string {
	return csvData.Schema
}

func (parqData *ParquetData) GetIngestedColumns() []string {
	columnNames := make([]string, len(parqData.Schema))
	for i, parqCol := range parqData.Schema {
		columnNames[i] = parqCol.RawColumnName
	}
	return columnNames
}

func (env *Environment) GetResourceType() resource.Type {
	return resource.EnvironmentType
}

func (environments Environments) Names() []string {
	names := make([]string, len(environments))
	for i, env := range environments {
		names[i] = env.Name
	}
	return names
}
