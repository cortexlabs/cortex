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
	"encoding/json"
	"math"

	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	s "github.com/cortexlabs/cortex/pkg/operator/api/strings"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
)

type Quantity struct {
	k8sresource.Quantity
	UserString string
}

func MustNewQuantity(str string) Quantity {
	k8sQuantity := k8sresource.MustParse(str)

	return Quantity{
		Quantity:   k8sQuantity,
		UserString: str,
	}
}

type QuantityValidation struct {
	Min k8sresource.Quantity
	Int bool
}

func QuantityParser(v *QuantityValidation) func(string) (interface{}, error) {
	return func(str string) (interface{}, error) {
		k8sQuantity, err := k8sresource.ParseQuantity(str)
		if err != nil {
			return Quantity{}, err
		}

		if k8sQuantity.Cmp(v.Min) < 0 {
			return nil, configreader.ErrorMustBeGreaterThanOrEqualTo(str, v.Min)
		}

		if v.Int {
			if k8sQuantity.MilliValue() != k8sQuantity.Value()*1000 {
				return nil, ErrorK8sQuantityMustBeInt(str)
			}
		}

		return Quantity{
			Quantity:   k8sQuantity,
			UserString: str,
		}, nil
	}
}

func (quantity *Quantity) ToFloat32() float32 {
	return float32(quantity.Quantity.MilliValue()) / float32(1000)
}

func (quantity *Quantity) ToKi() int64 {
	kiFloat := float64(quantity.Quantity.Value()) / float64(1024)
	return int64(math.Round(kiFloat))
}

// SplitInTwo divides the quantity in two and return both halves (ensuring they add up to the original value)
func (quantity *Quantity) SplitInTwo() (*k8sresource.Quantity, *k8sresource.Quantity) {
	milliValue := quantity.MilliValue()
	halfMilliValue := milliValue / 2
	q1 := k8sresource.NewMilliQuantity(milliValue-halfMilliValue, k8sresource.DecimalSI)
	q2 := k8sresource.NewMilliQuantity(halfMilliValue, k8sresource.DecimalSI)
	return q1, q2
}

func (quantity *Quantity) String() string {
	if quantity.UserString != "" {
		return quantity.UserString
	}
	return quantity.Quantity.String()
}

func (quantity *Quantity) Equal(quantity2 Quantity) bool {
	return quantity.Quantity.Cmp(quantity2.Quantity) == 0
}

func (quantity *Quantity) ID() string {
	return s.Int64(quantity.MilliValue())
}

func QuantityPtrID(quantity *Quantity) string {
	if quantity == nil {
		return "nil"
	}
	return quantity.ID()
}

func QuantityPtrsEqual(quantity *Quantity, quantity2 *Quantity) bool {
	if quantity == nil && quantity2 == nil {
		return true
	}
	if quantity == nil || quantity2 == nil {
		return false
	}
	return quantity.Equal(*quantity2)
}

type quantityMarshalable struct {
	Quantity   k8sresource.Quantity
	UserString string
}

func (quantity Quantity) MarshalJSON() ([]byte, error) {
	marshalable := quantityMarshalable{
		Quantity:   quantity.Quantity,
		UserString: quantity.UserString,
	}
	return json.Marshal(marshalable)
}

func (quantity *Quantity) UnmarshalJSON(data []byte) error {
	var unmarshaled quantityMarshalable
	err := json.Unmarshal(data, &unmarshaled)
	if err != nil {
		return err
	}
	quantity.Quantity = unmarshaled.Quantity
	quantity.UserString = unmarshaled.UserString
	return nil
}

func (quantity Quantity) MarshalBinary() ([]byte, error) {
	return json.Marshal(quantity)
}

func (quantity *Quantity) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, quantity)
}

func (quantity Quantity) MarshalText() ([]byte, error) {
	return json.Marshal(quantity)
}

func (quantity *Quantity) UnmarshalText(data []byte) error {
	return json.Unmarshal(data, quantity)
}
