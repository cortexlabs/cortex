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
	"encoding/json"

	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type RawColumnsTypeSplit struct {
	RawIntColumns    map[string]*RawIntColumn    `json:"raw_int_columns"`
	RawStringColumns map[string]*RawStringColumn `json:"raw_string_columns"`
	RawFloatColumns  map[string]*RawFloatColumn  `json:"raw_float_columns"`
}

type DataSplit struct {
	CSVData     *userconfig.CSVData     `json:"csv_data"`
	ParquetData *userconfig.ParquetData `json:"parquet_data"`
}

type ContextSerial struct {
	Context
	RawColumnSplit *RawColumnsTypeSplit `json:"raw_columns"`
	DataSplit      *DataSplit           `json:"environment_data"`
}

func (ctx Context) splitRawColumns() *RawColumnsTypeSplit {
	var rawIntColumns = make(map[string]*RawIntColumn)
	var rawFloatColumns = make(map[string]*RawFloatColumn)
	var rawStringColumns = make(map[string]*RawStringColumn)
	for name, rawColumn := range ctx.RawColumns {
		switch typedRawColumn := rawColumn.(type) {
		case *RawIntColumn:
			rawIntColumns[name] = typedRawColumn
		case *RawFloatColumn:
			rawFloatColumns[name] = typedRawColumn
		case *RawStringColumn:
			rawStringColumns[name] = typedRawColumn
		}
	}

	return &RawColumnsTypeSplit{
		RawIntColumns:    rawIntColumns,
		RawFloatColumns:  rawFloatColumns,
		RawStringColumns: rawStringColumns,
	}
}

func (ctx ContextSerial) collectRawColumns() RawColumns {
	var rawColumns = make(map[string]RawColumn)

	for name, rawColumn := range ctx.RawColumnSplit.RawIntColumns {
		rawColumns[name] = rawColumn
	}
	for name, rawColumn := range ctx.RawColumnSplit.RawFloatColumns {
		rawColumns[name] = rawColumn
	}
	for name, rawColumn := range ctx.RawColumnSplit.RawStringColumns {
		rawColumns[name] = rawColumn
	}

	return rawColumns
}

func (ctx Context) splitEnvironment() *DataSplit {
	var split DataSplit
	switch typedData := ctx.Environment.Data.(type) {
	case *userconfig.CSVData:
		split.CSVData = typedData
	case *userconfig.ParquetData:
		split.ParquetData = typedData
	}

	return &split
}

func (ctxSerial *ContextSerial) collectEnvironment() (*Environment, error) {
	if ctxSerial.DataSplit.ParquetData != nil && ctxSerial.DataSplit.CSVData == nil {
		ctxSerial.Environment.Data = ctxSerial.DataSplit.ParquetData
	} else if ctxSerial.DataSplit.CSVData != nil && ctxSerial.DataSplit.ParquetData == nil {
		ctxSerial.Environment.Data = ctxSerial.DataSplit.CSVData
	} else {
		return nil, errors.Wrap(userconfig.ErrorSpecifyOnlyOne("CSV", "PARQUET"), ctxSerial.App.Name, resource.EnvironmentType.String(), userconfig.DataKey)
	}
	return ctxSerial.Environment, nil
}

func (ctx Context) ToSerial() *ContextSerial {
	ctxSerial := ContextSerial{
		Context:        ctx,
		RawColumnSplit: ctx.splitRawColumns(),
		DataSplit:      ctx.splitEnvironment(),
	}

	return &ctxSerial
}

func (ctxSerial ContextSerial) FromSerial() (*Context, error) {
	ctx := ctxSerial.Context
	ctx.RawColumns = ctxSerial.collectRawColumns()
	environment, err := ctxSerial.collectEnvironment()
	if err != nil {
		return nil, err
	}
	ctx.Environment = environment
	return &ctx, nil
}

func (ctx Context) ToMsgpackBytes() ([]byte, error) {
	return util.MarshalMsgpack(ctx.ToSerial())
}

func FromMsgpackBytes(b []byte) (*Context, error) {
	var ctxSerial ContextSerial
	err := util.UnmarshalMsgpack(b, &ctxSerial)
	if err != nil {
		return nil, err
	}
	return ctxSerial.FromSerial()
}

func (ctx Context) MarshalJSON() ([]byte, error) {
	msgpackBytes, err := ctx.ToMsgpackBytes()
	if err != nil {
		return nil, err
	}
	msgpackJSONBytes, err := json.Marshal(&msgpackBytes)
	if err != nil {
		return nil, errors.Wrap(err, s.ErrMarshalJson)
	}
	return msgpackJSONBytes, nil
}

func (ctx *Context) UnmarshalJSON(b []byte) error {
	var msgpackBytes []byte
	if err := json.Unmarshal(b, &msgpackBytes); err != nil {
		return errors.Wrap(err, s.ErrUnmarshalJson)
	}
	ctxPtr, err := FromMsgpackBytes(msgpackBytes)
	if err != nil {
		return err
	}
	*ctx = *ctxPtr
	return nil
}
