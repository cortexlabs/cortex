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

type RawFeaturesTypeSplit struct {
	RawIntFeatures    map[string]*RawIntFeature    `json:"raw_int_features"`
	RawStringFeatures map[string]*RawStringFeature `json:"raw_string_features"`
	RawFloatFeatures  map[string]*RawFloatFeature  `json:"raw_float_features"`
}

type DataSplit struct {
	CsvData     *userconfig.CsvData     `json:"csv_data"`
	ParquetData *userconfig.ParquetData `json:"parquet_data"`
}

type ContextSerial struct {
	Context
	RawFeatureSplit *RawFeaturesTypeSplit `json:"raw_features"`
	DataSplit       *DataSplit            `json:"environment_data"`
}

func (ctx Context) splitRawFeatures() *RawFeaturesTypeSplit {
	var rawIntFeatures = make(map[string]*RawIntFeature)
	var rawFloatFeatures = make(map[string]*RawFloatFeature)
	var rawStringFeatures = make(map[string]*RawStringFeature)
	for name, rawFeature := range ctx.RawFeatures {
		switch typedRawFeature := rawFeature.(type) {
		case *RawIntFeature:
			rawIntFeatures[name] = typedRawFeature
		case *RawFloatFeature:
			rawFloatFeatures[name] = typedRawFeature
		case *RawStringFeature:
			rawStringFeatures[name] = typedRawFeature
		}
	}

	return &RawFeaturesTypeSplit{
		RawIntFeatures:    rawIntFeatures,
		RawFloatFeatures:  rawFloatFeatures,
		RawStringFeatures: rawStringFeatures,
	}
}

func (ctx ContextSerial) collectRawFeatures() RawFeatures {
	var rawFeatures = make(map[string]RawFeature)

	for name, rawFeature := range ctx.RawFeatureSplit.RawIntFeatures {
		rawFeatures[name] = rawFeature
	}
	for name, rawFeature := range ctx.RawFeatureSplit.RawFloatFeatures {
		rawFeatures[name] = rawFeature
	}
	for name, rawFeature := range ctx.RawFeatureSplit.RawStringFeatures {
		rawFeatures[name] = rawFeature
	}

	return rawFeatures
}

func (ctx Context) splitEnvironment() *DataSplit {
	var split DataSplit
	switch typedData := ctx.Environment.Data.(type) {
	case *userconfig.CsvData:
		split.CsvData = typedData
	case *userconfig.ParquetData:
		split.ParquetData = typedData
	}

	return &split
}

func (ctxSerial *ContextSerial) collectEnvironment() (*Environment, error) {
	if ctxSerial.DataSplit.ParquetData != nil && ctxSerial.DataSplit.CsvData == nil {
		ctxSerial.Environment.Data = ctxSerial.DataSplit.ParquetData
	} else if ctxSerial.DataSplit.CsvData != nil && ctxSerial.DataSplit.ParquetData == nil {
		ctxSerial.Environment.Data = ctxSerial.DataSplit.CsvData
	} else {
		return nil, errors.Wrap(userconfig.ErrorSpecifyOnlyOne("CSV", "PARQUET"), ctxSerial.App.Name, resource.EnvironmentType.String(), userconfig.DataKey)
	}
	return ctxSerial.Environment, nil
}

func (ctx Context) ToSerial() *ContextSerial {
	ctxSerial := ContextSerial{
		Context:         ctx,
		RawFeatureSplit: ctx.splitRawFeatures(),
		DataSplit:       ctx.splitEnvironment(),
	}

	return &ctxSerial
}

func (ctxSerial ContextSerial) FromSerial() (*Context, error) {
	ctx := ctxSerial.Context
	ctx.RawFeatures = ctxSerial.collectRawFeatures()
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
