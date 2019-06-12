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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

type RawColumnsTypeSplit struct {
	RawIntColumns      map[string]*RawIntColumn      `json:"raw_int_columns"`
	RawStringColumns   map[string]*RawStringColumn   `json:"raw_string_columns"`
	RawFloatColumns    map[string]*RawFloatColumn    `json:"raw_float_columns"`
	RawInferredColumns map[string]*RawInferredColumn `json:"raw_inferred_columns"`
}

type DataSplit struct {
	CSVData     *userconfig.CSVData     `json:"csv_data"`
	ParquetData *userconfig.ParquetData `json:"parquet_data"`
}

type Serial struct {
	*Context
	RawColumnSplit *RawColumnsTypeSplit `json:"raw_columns"`
	DataSplit      *DataSplit           `json:"environment_data"`
}

func (ctx *Context) splitRawColumns() *RawColumnsTypeSplit {
	var rawIntColumns = make(map[string]*RawIntColumn)
	var rawFloatColumns = make(map[string]*RawFloatColumn)
	var rawStringColumns = make(map[string]*RawStringColumn)
	var rawInferredColumns = make(map[string]*RawInferredColumn)
	for name, rawColumn := range ctx.RawColumns {
		switch typedRawColumn := rawColumn.(type) {
		case *RawIntColumn:
			rawIntColumns[name] = typedRawColumn
		case *RawFloatColumn:
			rawFloatColumns[name] = typedRawColumn
		case *RawStringColumn:
			rawStringColumns[name] = typedRawColumn
		case *RawInferredColumn:
			rawInferredColumns[name] = typedRawColumn
		}
	}

	return &RawColumnsTypeSplit{
		RawIntColumns:      rawIntColumns,
		RawFloatColumns:    rawFloatColumns,
		RawStringColumns:   rawStringColumns,
		RawInferredColumns: rawInferredColumns,
	}
}

func (serial *Serial) collectRawColumns() RawColumns {
	var rawColumns = make(map[string]RawColumn)

	for name, rawColumn := range serial.RawColumnSplit.RawIntColumns {
		rawColumns[name] = rawColumn
	}
	for name, rawColumn := range serial.RawColumnSplit.RawFloatColumns {
		rawColumns[name] = rawColumn
	}
	for name, rawColumn := range serial.RawColumnSplit.RawStringColumns {
		rawColumns[name] = rawColumn
	}
	for name, rawColumn := range serial.RawColumnSplit.RawInferredColumns {
		rawColumns[name] = rawColumn
	}

	return rawColumns
}

func (ctx *Context) splitEnvironment() *DataSplit {
	var split DataSplit
	switch typedData := ctx.Environment.Data.(type) {
	case *userconfig.CSVData:
		split.CSVData = typedData
	case *userconfig.ParquetData:
		split.ParquetData = typedData
	}

	return &split
}

func (serial *Serial) collectEnvironment() (*Environment, error) {
	if serial.DataSplit.ParquetData != nil && serial.DataSplit.CSVData == nil {
		serial.Environment.Data = serial.DataSplit.ParquetData
	} else if serial.DataSplit.CSVData != nil && serial.DataSplit.ParquetData == nil {
		serial.Environment.Data = serial.DataSplit.CSVData
	} else {
		return nil, errors.Wrap(userconfig.ErrorSpecifyOnlyOne("CSV", "PARQUET"), serial.App.Name, resource.EnvironmentType.String(), userconfig.DataKey)
	}
	return serial.Environment, nil
}

func (ctx *Context) castSchemaTypes() error {
	for _, constant := range ctx.Constants {
		if constant.Type != nil {
			castedType, err := userconfig.ValidateOutputSchema(constant.Type)
			if err != nil {
				return err
			}
			constant.Constant.Type = castedType
		}
	}

	for _, aggregator := range ctx.Aggregators {
		if aggregator.OutputType != nil {
			casted, err := userconfig.ValidateOutputSchema(aggregator.OutputType)
			if err != nil {
				return err
			}
			aggregator.Aggregator.OutputType = casted
		}

		if aggregator.Input != nil {
			casted, err := userconfig.ValidateInputTypeSchema(aggregator.Input.Type, false, true)
			if err != nil {
				return err
			}
			aggregator.Aggregator.Input.Type = casted
		}
	}

	for _, aggregate := range ctx.Aggregates {
		if aggregate.Type != nil {
			casted, err := userconfig.ValidateOutputSchema(aggregate.Type)
			if err != nil {
				return err
			}
			aggregate.Type = casted
		}
	}

	for _, transformer := range ctx.Transformers {
		if transformer.Input != nil {
			casted, err := userconfig.ValidateInputTypeSchema(transformer.Input.Type, false, true)
			if err != nil {
				return err
			}
			transformer.Transformer.Input.Type = casted
		}
	}

	for _, estimator := range ctx.Estimators {
		if estimator.Input != nil {
			casted, err := userconfig.ValidateInputTypeSchema(estimator.Input.Type, false, true)
			if err != nil {
				return err
			}
			estimator.Estimator.Input.Type = casted
		}

		if estimator.TrainingInput != nil {
			casted, err := userconfig.ValidateInputTypeSchema(estimator.TrainingInput.Type, false, true)
			if err != nil {
				return err
			}
			estimator.Estimator.TrainingInput.Type = casted
		}

		if estimator.Hparams != nil {
			casted, err := userconfig.ValidateInputTypeSchema(estimator.Hparams.Type, true, true)
			if err != nil {
				return err
			}
			estimator.Estimator.Hparams.Type = casted
		}
	}

	return nil
}

func (ctx *Context) ToSerial() *Serial {
	serial := Serial{
		Context:        ctx,
		RawColumnSplit: ctx.splitRawColumns(),
		DataSplit:      ctx.splitEnvironment(),
	}

	return &serial
}

func (serial *Serial) ContextFromSerial() (*Context, error) {
	ctx := serial.Context

	if ctx.Environment != nil {
		ctx.RawColumns = serial.collectRawColumns()

		environment, err := serial.collectEnvironment()
		if err != nil {
			return nil, err
		}
		ctx.Environment = environment

		err = ctx.castSchemaTypes()
		if err != nil {
			return nil, err
		}
	}

	return ctx, nil
}

func (ctx Context) ToMsgpackBytes() ([]byte, error) {
	return msgpack.Marshal(ctx.ToSerial())
}

func FromMsgpackBytes(b []byte) (*Context, error) {
	var serial Serial
	err := msgpack.Unmarshal(b, &serial)
	if err != nil {
		return nil, err
	}
	return serial.ContextFromSerial()
}

func (ctx Context) MarshalJSON() ([]byte, error) {
	msgpackBytes, err := ctx.ToMsgpackBytes()
	if err != nil {
		return nil, err
	}
	msgpackJSONBytes, err := json.Marshal(&msgpackBytes)
	if err != nil {
		return nil, err
	}
	return msgpackJSONBytes, nil
}

func (ctx *Context) UnmarshalJSON(b []byte) error {
	var msgpackBytes []byte
	if err := json.Unmarshal(b, &msgpackBytes); err != nil {
		return err
	}
	ctxPtr, err := FromMsgpackBytes(msgpackBytes)
	if err != nil {
		return err
	}
	*ctx = *ctxPtr
	return nil
}
