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
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func mustValidateOutputSchema(yamlStr string) userconfig.OutputSchema {
	outputSchema, err := userconfig.ValidateOutputSchema(cr.MustReadYAMLStr(yamlStr))
	if err != nil {
		panic(err)
	}
	return outputSchema
}

func genConst(id string, outputType string, value string) *context.Constant {
	var outType userconfig.OutputSchema = nil
	if outputType != "" {
		outType = mustValidateOutputSchema(outputType)
	}

	constant := &context.Constant{
		Constant: &userconfig.Constant{
			ResourceFields: userconfig.ResourceFields{
				Name: "c" + id,
			},
			Type:  outType,
			Value: cr.MustReadYAMLStr(value),
		},
		ResourceFields: &context.ResourceFields{
			ID:           "a_c" + id,
			ResourceType: resource.ConstantType,
		},
	}

	err := constant.Validate()
	if err != nil {
		panic(err)
	}

	return constant
}

func genAgg(id string, aggregatorType string) (*context.Aggregate, *context.Aggregator) {
	var outputType userconfig.OutputSchema = nil
	if aggregatorType != "" {
		outputType = mustValidateOutputSchema(aggregatorType)
	}

	aggregator := &context.Aggregator{
		Aggregator: &userconfig.Aggregator{
			ResourceFields: userconfig.ResourceFields{
				Name: "aggregator" + id,
			},
			OutputType: outputType,
		},
		ResourceFields: &context.ResourceFields{
			ID:           "d_aggregator" + id,
			ResourceType: resource.AggregatorType,
		},
	}

	aggregate := &context.Aggregate{
		Aggregate: &userconfig.Aggregate{
			ResourceFields: userconfig.ResourceFields{
				Name: "agg" + id,
			},
			Aggregator: "aggregator" + id,
		},
		Type: aggregator.OutputType,
		ComputedResourceFields: &context.ComputedResourceFields{
			ResourceFields: &context.ResourceFields{
				ID:           "c_agg" + id,
				ResourceType: resource.AggregateType,
			},
		},
	}

	return aggregate, aggregator
}

func genTrans(id string, transformerType userconfig.ColumnType) (*context.TransformedColumn, *context.Transformer) {
	transformer := &context.Transformer{
		Transformer: &userconfig.Transformer{
			ResourceFields: userconfig.ResourceFields{
				Name: "transformer" + id,
			},
			OutputType: transformerType,
		},
		ResourceFields: &context.ResourceFields{
			ID:           "f_transformer" + id,
			ResourceType: resource.TransformerType,
		},
	}

	transformedCol := &context.TransformedColumn{
		TransformedColumn: &userconfig.TransformedColumn{
			ResourceFields: userconfig.ResourceFields{
				Name: "tc" + id,
			},
			Transformer: "transformer" + id,
		},
		Type: transformer.OutputType,
		ComputedResourceFields: &context.ComputedResourceFields{
			ResourceFields: &context.ResourceFields{
				ID:           "e_tc" + id,
				ResourceType: resource.TransformedColumnType,
			},
		},
	}

	return transformedCol, transformer
}

var c1 = genConst("1", `STRING`, `test`)
var c2 = genConst("2", ``, `test`)
var c3 = genConst("3", `
  map: {INT: FLOAT}
  map2: {a: FLOAT, b: FLOAT, c: INT}
  str: STRING
  floats: [FLOAT]
  list:
    - STRING:
        lat: INT
        lon:
          a: [STRING]
  `, `
  map: {2: 2.2, 3: 3}
  map2: {a: 2.2, b: 3, c: 4}
  str: test
  floats: [1.1, 2.2, 3.3]
  list:
    - key_1:
        lat: 17
        lon:
          a: [test1, test2, test3]
      key_2:
        lat: 88
        lon:
          a: [test4, test5, test6]
    - key_a:
        lat: 12
        lon:
          a: [test7, test8, test9]
  `)
var c4 = genConst("4", ``, `
  map: {2: 2.2, 3: 3}
  map2: {a: 2, b: 3, c: 4}
  str: test
  floats: [1.1, 2.2, 3.3]
  list:
    - key_1:
        lat: 17
        lon:
          a: [test1, test2, test3]
      key_2:
        lat: 88
        lon:
          a: [test4, test5, test6]
    - key_a:
        lat: 12
        lon:
          a: [test7, test8, test9]
  `)
var c5 = genConst("5", `FLOAT`, `2`)
var c6 = genConst("6", `INT`, `2`)
var c7 = genConst("7", ``, `2`)
var c8 = genConst("8", `{a: INT, b: INT}`, `{a: 1, b: 2}`)
var c9 = genConst("9", `{STRING: INT}`, `{a: 1, b: 2}`)
var ca = genConst("a", ``, `{a: 1, b: 2}`)
var cb = genConst("b", `[FLOAT]`, `[2]`)
var cc = genConst("c", `[INT]`, `[2]`)
var cd = genConst("d", ``, `[2]`)
var ce = genConst("e", `[STRING]`, `[a, b, c]`)
var cf = genConst("f", ``, `{a: [a, b, c]}`)

var rc1 = &context.RawIntColumn{
	RawIntColumn: &userconfig.RawIntColumn{
		ResourceFields: userconfig.ResourceFields{
			Name: "rc1",
		},
		Type: userconfig.IntegerColumnType,
	},
	ComputedResourceFields: &context.ComputedResourceFields{
		ResourceFields: &context.ResourceFields{
			ID:           "b_rc1",
			ResourceType: resource.RawColumnType,
		},
	},
}

// Type not specified
var rc2 = &context.RawInferredColumn{
	RawInferredColumn: &userconfig.RawInferredColumn{
		ResourceFields: userconfig.ResourceFields{
			Name: "rc2",
		},
		Type: userconfig.InferredColumnType,
	},
	ComputedResourceFields: &context.ComputedResourceFields{
		ResourceFields: &context.ResourceFields{
			ID:           "b_rc2",
			ResourceType: resource.RawColumnType,
		},
	},
}

var rc3 = &context.RawStringColumn{
	RawStringColumn: &userconfig.RawStringColumn{
		ResourceFields: userconfig.ResourceFields{
			Name: "rc3",
		},
		Type: userconfig.StringColumnType,
	},
	ComputedResourceFields: &context.ComputedResourceFields{
		ResourceFields: &context.ResourceFields{
			ID:           "b_rc3",
			ResourceType: resource.RawColumnType,
		},
	},
}

var rc4 = &context.RawFloatColumn{
	RawFloatColumn: &userconfig.RawFloatColumn{
		ResourceFields: userconfig.ResourceFields{
			Name: "rc4",
		},
		Type: userconfig.FloatColumnType,
	},
	ComputedResourceFields: &context.ComputedResourceFields{
		ResourceFields: &context.ResourceFields{
			ID:           "b_rc4",
			ResourceType: resource.RawColumnType,
		},
	},
}

var agg1, aggregator1 = genAgg("1", `STRING`)
var agg2, aggregator2 = genAgg("2", ``) // Nil output type
var agg3, aggregator3 = genAgg("3", `
  map: {INT: FLOAT}
  map2: {a: FLOAT, b: FLOAT, c: INT}
  str: STRING
  floats: [FLOAT]
  list:
    - STRING:
        lat: INT
        lon:
          a: [STRING]
  `)
var agg4, aggregator4 = genAgg("4", `INT`)
var agg5, aggregator5 = genAgg("5", `FLOAT`)
var agg6, aggregator6 = genAgg("6", `{a: INT, b: INT}`)
var agg7, aggregator7 = genAgg("7", `{STRING: INT}`)
var agg8, aggregator8 = genAgg("8", `[INT]`)
var agg9, aggregator9 = genAgg("9", `[FLOAT]`)
var agga, aggregatora = genAgg("a", `[STRING]`)
var aggb, aggregatorb = genAgg("b", `{a: [STRING]}`)

var tc1, transformer1 = genTrans("1", userconfig.StringColumnType)
var tc2, transformer2 = genTrans("2", userconfig.InferredColumnType)
var tc3, transformer3 = genTrans("3", userconfig.IntegerColumnType)
var tc4, transformer4 = genTrans("4", userconfig.FloatColumnType)
var tc5, transformer5 = genTrans("5", userconfig.StringListColumnType)
var tc6, transformer6 = genTrans("6", userconfig.IntegerListColumnType)
var tc7, transformer7 = genTrans("7", userconfig.FloatListColumnType)

var constants = []context.Resource{c1, c2, c3, c4, c5, c6, c7, c8, c9, ca, cb, cc, cd, ce, cf}
var rawCols = []context.Resource{rc1, rc2, rc3, rc4}
var aggregates = []context.Resource{agg1, agg2, agg3, agg4, agg5, agg6, agg7, agg8, agg9, agga, aggb}
var transformedCols = []context.Resource{tc1, tc2, tc3, tc4, tc5, tc6, tc7}

var aggregators = context.Aggregators{
	aggregator1.GetName(): aggregator1,
	aggregator2.GetName(): aggregator2,
	aggregator3.GetName(): aggregator3,
	aggregator4.GetName(): aggregator4,
	aggregator5.GetName(): aggregator5,
	aggregator6.GetName(): aggregator6,
	aggregator7.GetName(): aggregator7,
	aggregator8.GetName(): aggregator8,
	aggregator9.GetName(): aggregator9,
	aggregatora.GetName(): aggregatora,
	aggregatorb.GetName(): aggregatorb,
}
var transformers = context.Transformers{
	transformer1.GetName(): transformer1,
	transformer2.GetName(): transformer2,
	transformer3.GetName(): transformer3,
	transformer4.GetName(): transformer4,
	transformer5.GetName(): transformer5,
	transformer6.GetName(): transformer6,
	transformer7.GetName(): transformer7,
}

var allResources = append(append(append(constants, rawCols...), aggregates...), transformedCols...)

var constantsMap = make(map[string]context.Resource)
var rawColsMap = make(map[string]context.Resource)
var aggregatesMap = make(map[string]context.Resource)
var transformedColsMap = make(map[string]context.Resource)
var allResourcesMap = make(map[string]context.Resource)

var allResourceConfigsMap = make(map[string][]userconfig.Resource)

func init() {
	for _, res := range constants {
		constantsMap[res.GetName()] = res
	}
	for _, res := range rawCols {
		rawColsMap[res.GetName()] = res
	}
	for _, res := range aggregates {
		aggregatesMap[res.GetName()] = res
	}
	for _, res := range transformedCols {
		transformedColsMap[res.GetName()] = res
	}
	for _, res := range allResources {
		allResourcesMap[res.GetName()] = res
	}

	for _, res := range allResources {
		allResourceConfigsMap[res.GetName()] = []userconfig.Resource{res}
	}
}
