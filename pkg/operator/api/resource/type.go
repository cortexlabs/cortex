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

package resource

import (
	"strings"
)

type Type int
type Types []Type

const (
	UnknownType           Type = iota // 0
	AppType                           // 1
	RawColumnType                     // 2
	TransformedColumnType             // 3
	AggregateType                     // 4
	APIType                           // 5
	ModelType                         // 6
	EnvironmentType                   // 7
	AggregatorType                    // 8
	TransformerType                   // 9
	EstimatorType                     // 10
	TemplateType                      // 11
	EmbedType                         // 12
	TrainingDatasetType               // 13
	ConstantType                      // 14
	PythonPackageType                 // 15
)

var (
	types = []string{
		"unknown",
		"deployment",
		"raw_column",
		"transformed_column",
		"aggregate",
		"api",
		"model",
		"environment",
		"aggregator",
		"transformer",
		"estimator",
		"template",
		"embed",
		"training_dataset",
		"constant",
		"python_package",
	}

	typePlurals = []string{
		"unknown",
		"deployments",
		"raw_columns",
		"transformed_columns",
		"aggregates",
		"apis",
		"models",
		"environments",
		"aggregators",
		"transformers",
		"estimators",
		"templates",
		"embeds",
		"training_datasets",
		"constants",
		"python_packages",
	}

	typeAcronyms = map[string]Type{
		"rc":   RawColumnType,
		"rcs":  RawColumnType,
		"tc":   TransformedColumnType,
		"tcs":  TransformedColumnType,
		"agg":  AggregateType,
		"aggs": AggregateType,
		"m":    ModelType,
		"ms":   ModelType,
		"env":  EnvironmentType,
		"envs": EnvironmentType,
		"tr":   TransformerType,
		"trs":  TransformerType,
		"td":   TrainingDatasetType,
		"tds":  TrainingDatasetType,
		"ct":   ConstantType,
		"cts":  ConstantType,
		"pp":   PythonPackageType,
		"pps":  PythonPackageType,
	}

	VisibleTypes = Types{
		RawColumnType,
		AggregateType,
		TransformedColumnType,
		TrainingDatasetType,
		ModelType,
		APIType,
	}
)

func TypeFromString(s string) Type {
	for i := 0; i < len(types); i++ {
		if s == types[i] {
			return Type(i)
		}

		if s == typePlurals[i] {
			return Type(i)
		}

		if t, ok := typeAcronyms[s]; ok {
			return t
		}
	}
	return UnknownType
}

func TypeFromKindString(s string) Type {
	for i := 0; i < len(types); i++ {
		if s == types[i] {
			return Type(i)
		}
	}
	return UnknownType
}

func (ts Types) String() string {
	return strings.Join(ts.StringList(), ", ")
}

func (ts Types) Plural() string {
	return strings.Join(ts.PluralList(), ", ")
}

func (ts Types) StringList() []string {
	strs := make([]string, len(ts))
	for i, t := range ts {
		strs[i] = t.String()
	}
	return strs
}

func (ts Types) PluralList() []string {
	strs := make([]string, len(ts))
	for i, t := range ts {
		strs[i] = t.Plural()
	}
	return strs
}

func (t Type) String() string {
	return types[t]
}

func (t Type) Plural() string {
	return typePlurals[t]
}

// MarshalText satisfies TextMarshaler
func (t Type) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *Type) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(types); i++ {
		if enum == types[i] {
			*t = Type(i)
			return nil
		}
	}

	*t = UnknownType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *Type) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t Type) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

func VisibleResourceTypeFromPrefix(prefix string) (Type, error) {
	prefix = strings.ToLower(prefix)

	if resourceType := TypeFromString(prefix); resourceType != UnknownType {
		return resourceType, nil
	}

	resourceTypesMap := make(map[Type]struct{})
	for _, resourceType := range VisibleTypes {
		if strings.HasPrefix(resourceType.String(), prefix) {
			resourceTypesMap[resourceType] = struct{}{}
		}

		if strings.HasPrefix(resourceType.Plural(), prefix) {
			resourceTypesMap[resourceType] = struct{}{}
		}
	}

	i := 0
	resourceTypes := make(Types, len(resourceTypesMap))
	for resourceType := range resourceTypesMap {
		resourceTypes[i] = resourceType
		i++
	}

	if len(resourceTypes) > 1 {
		return UnknownType, ErrorBeMoreSpecific(resourceTypes.PluralList()...)
	}

	if len(resourceTypes) == 0 {
		return UnknownType, ErrorInvalidType(prefix)
	}

	return resourceTypes[0], nil
}
