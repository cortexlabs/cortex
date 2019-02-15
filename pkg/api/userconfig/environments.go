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
	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	cr "github.com/cortexlabs/cortex/pkg/utils/configreader"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type Environments []*Environment

type Environment struct {
	Name     string    `json:"name" yaml:"name"`
	LogLevel *LogLevel `json:"log_level" yaml:"log_level"`
	Data     Data      `json:"-" yaml:"-"`
	FilePath string    `json:"file_path"  yaml:"-"`
}

var environmentValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		&cr.StructFieldValidation{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required:                   true,
				AlphaNumericDashUnderscore: true,
			},
		},
		&cr.StructFieldValidation{
			StructField:      "LogLevel",
			StructValidation: logLevelValidation,
		},
		&cr.StructFieldValidation{
			StructField:               "Data",
			Key:                       "data",
			InterfaceStructValidation: dataValidation,
		},
		typeFieldValidation,
	},
}

type LogLevel struct {
	Tensorflow string `json:"tensorflow" yaml:"tensorflow"`
	Spark      string `json:"spark" yaml:"spark"`
}

var logLevelValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		&cr.StructFieldValidation{
			StructField: "Tensorflow",
			StringValidation: &cr.StringValidation{
				Default:       "INFO",
				AllowedValues: []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
			},
		},
		&cr.StructFieldValidation{
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
	InterfaceStructTypes: map[string]*cr.InterfaceStructType{
		"csv": &cr.InterfaceStructType{
			Type:                   (*CSVData)(nil),
			StructFieldValidations: csvDataFieldValidations,
		},
		"parquet": &cr.InterfaceStructType{
			Type:                   (*ParquetData)(nil),
			StructFieldValidations: parquetDataFieldValidations,
		},
	},
}

type CSVData struct {
	Type      string     `json:"type" yaml:"type"`
	Path      string     `json:"path" yaml:"path"`
	Schema    []string   `json:"schema" yaml:"schema"`
	DropNull  bool       `json:"drop_null" yaml:"drop_null"`
	CSVConfig *CSVConfig `json:"csv_config" yaml:"csv_config"`
}

// SPARK_VERSION dependent
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
	&cr.StructFieldValidation{
		StructField: "Path",
		StringValidation: cr.GetS3aPathValidation(&cr.S3aPathValidation{
			Required: true,
		}),
	},
	&cr.StructFieldValidation{
		StructField: "Schema",
		StringListValidation: &cr.StringListValidation{
			Required: true,
		},
	},
	&cr.StructFieldValidation{
		StructField: "DropNull",
		BoolValidation: &cr.BoolValidation{
			Default: false,
		},
	},
	&cr.StructFieldValidation{
		StructField: "CSVConfig",
		StructValidation: &cr.StructValidation{
			StructFieldValidations: []*cr.StructFieldValidation{
				&cr.StructFieldValidation{
					StructField:         "Sep",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:         "Encoding",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:         "Quote",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:         "Escape",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:         "Comment",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:       "Header",
					BoolPtrValidation: &cr.BoolPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:       "IgnoreLeadingWhiteSpace",
					BoolPtrValidation: &cr.BoolPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:       "IgnoreTrailingWhiteSpace",
					BoolPtrValidation: &cr.BoolPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:         "NullValue",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:         "NanValue",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:         "PositiveInf",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:         "NegativeInf",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField: "MaxColumns",
					Int32PtrValidation: &cr.Int32PtrValidation{
						GreaterThan: util.Int32Ptr(0),
					},
				},
				&cr.StructFieldValidation{
					StructField: "MaxCharsPerColumn",
					Int32PtrValidation: &cr.Int32PtrValidation{
						GreaterThanOrEqualTo: util.Int32Ptr(-1),
					},
				},
				&cr.StructFieldValidation{
					StructField:       "Multiline",
					BoolPtrValidation: &cr.BoolPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:         "CharToEscapeQuoteEscaping",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				&cr.StructFieldValidation{
					StructField:         "EmptyValue",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
			},
		},
	},
}

type ParquetData struct {
	Type     string           `json:"type" yaml:"type"`
	Path     string           `json:"path" yaml:"path"`
	Schema   []*ParquetColumn `json:"schema" yaml:"schema"`
	DropNull bool             `json:"drop_null" yaml:"drop_null"`
}

var parquetDataFieldValidations = []*cr.StructFieldValidation{
	&cr.StructFieldValidation{
		StructField: "Path",
		StringValidation: cr.GetS3aPathValidation(&cr.S3aPathValidation{
			Required: true,
		}),
	},
	&cr.StructFieldValidation{
		StructField: "Schema",
		StructListValidation: &cr.StructListValidation{
			StructValidation: parquetColumnValidation,
		},
	},
	&cr.StructFieldValidation{
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
		&cr.StructFieldValidation{
			StructField: "ParquetColumnName",
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		&cr.StructFieldValidation{
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

	dups := util.FindDuplicateStrs(env.Data.GetIngestedColumns())
	if len(dups) > 0 {
		return errors.New(Identify(env), DataKey, SchemaKey, "column name", s.ErrDuplicatedValue(dups[0]))
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
	column_names := make([]string, len(parqData.Schema))
	for i, parqCol := range parqData.Schema {
		column_names[i] = parqCol.RawColumnName
	}
	return column_names
}

func (env *Environment) GetName() string {
	return env.Name
}

func (env *Environment) GetResourceType() resource.Type {
	return resource.EnvironmentType
}

func (env *Environment) GetFilePath() string {
	return env.FilePath
}

func (environments Environments) Names() []string {
	names := make([]string, len(environments))
	for i, env := range environments {
		names[i] = env.Name
	}
	return names
}
