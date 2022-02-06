/*
Copyright 2022 Cortex Labs, Inc.

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

package k8s

import (
	"encoding/json"
	"math"

	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

type Quantity struct {
	kresource.Quantity
	UserString string
}

type QuantityValidation struct {
	GreaterThan          *kresource.Quantity
	GreaterThanOrEqualTo *kresource.Quantity
	LessThan             *kresource.Quantity
	LessThanOrEqualTo    *kresource.Quantity
}

func QuantityParser(v *QuantityValidation) func(string) (interface{}, error) {
	return func(str string) (interface{}, error) {
		k8sQuantity, err := kresource.ParseQuantity(str)
		if err != nil {
			return Quantity{}, ErrorParseQuantity(str)
		}

		if v.GreaterThan != nil {
			if k8sQuantity.Cmp(*v.GreaterThan) <= 0 {
				return nil, configreader.ErrorMustBeGreaterThan(str, *v.GreaterThan)
			}
		}
		if v.GreaterThanOrEqualTo != nil {
			if k8sQuantity.Cmp(*v.GreaterThanOrEqualTo) < 0 {
				return nil, configreader.ErrorMustBeGreaterThanOrEqualTo(str, *v.GreaterThanOrEqualTo)
			}
		}
		if v.LessThan != nil {
			if k8sQuantity.Cmp(*v.LessThan) >= 0 {
				return nil, configreader.ErrorMustBeLessThan(str, *v.LessThan)
			}
		}
		if v.LessThanOrEqualTo != nil {
			if k8sQuantity.Cmp(*v.LessThanOrEqualTo) > 0 {
				return nil, configreader.ErrorMustBeLessThanOrEqualTo(str, *v.LessThanOrEqualTo)
			}
		}

		return Quantity{
			Quantity:   k8sQuantity,
			UserString: str,
		}, nil
	}
}

func WrapQuantity(k8sQuantity kresource.Quantity) *Quantity {
	return &Quantity{
		Quantity: k8sQuantity,
	}
}

func NewQuantity(value int64) *Quantity {
	k8sQuantity := kresource.NewQuantity(value, kresource.DecimalSI)

	return &Quantity{
		Quantity: *k8sQuantity,
	}
}

func NewMilliQuantity(milliValue int64) *Quantity {
	k8sQuantity := kresource.NewMilliQuantity(milliValue, kresource.DecimalSI)

	return &Quantity{
		Quantity:   *k8sQuantity,
		UserString: s.Int64(milliValue) + "m",
	}
}

// Returns nil if no quantities are passed in
func NewSummed(quantities ...kresource.Quantity) *Quantity {
	if len(quantities) == 0 {
		return nil
	}

	k8sQuantity := kresource.Quantity{}
	for _, q := range quantities {
		k8sQuantity.Add(q)
	}

	return &Quantity{
		Quantity: k8sQuantity,
	}
}

func (quantity *Quantity) MilliString() string {
	return s.Int64(quantity.Quantity.MilliValue()) + "m"
}

func (quantity *Quantity) ToFloat32() float32 {
	return float32(quantity.Quantity.MilliValue()) / float32(1000)
}

func ToKiRounded(k8sQuantity kresource.Quantity) int64 {
	kiFloat := float64(k8sQuantity.Value()) / float64(1024)
	return int64(math.Round(kiFloat))
}
func ToKiRoundedStr(k8sQuantity kresource.Quantity) string {
	return s.Int64(ToKiRounded(k8sQuantity)) + "Ki"
}
func (quantity *Quantity) ToKiRounded() int64 {
	return ToKiRounded(quantity.Quantity)
}
func (quantity *Quantity) ToKiRoundedStr() string {
	return ToKiRoundedStr(quantity.Quantity)
}

func ToKiCeil(k8sQuantity kresource.Quantity) int64 {
	kiFloat := float64(k8sQuantity.Value()) / float64(1024)
	return int64(math.Ceil(kiFloat))
}
func ToKiCeilStr(k8sQuantity kresource.Quantity) string {
	return s.Int64(ToKiCeil(k8sQuantity)) + "Ki"
}
func (quantity *Quantity) ToKiCeil() int64 {
	return ToKiCeil(quantity.Quantity)
}
func (quantity *Quantity) ToKiCeilStr() string {
	return ToKiCeilStr(quantity.Quantity)
}

func ToKiFloor(k8sQuantity kresource.Quantity) int64 {
	kiFloat := float64(k8sQuantity.Value()) / float64(1024)
	return int64(math.Floor(kiFloat))
}
func ToKiFloorStr(k8sQuantity kresource.Quantity) string {
	return s.Int64(ToKiFloor(k8sQuantity)) + "Ki"
}
func (quantity *Quantity) ToKiFloor() int64 {
	return ToKiFloor(quantity.Quantity)
}
func (quantity *Quantity) ToKiFloorStr() string {
	return ToKiFloorStr(quantity.Quantity)
}

func ToMiRounded(k8sQuantity kresource.Quantity) int64 {
	miFloat := float64(k8sQuantity.Value()) / float64(1024*1024)
	return int64(math.Round(miFloat))
}
func ToMiRoundedStr(k8sQuantity kresource.Quantity) string {
	return s.Int64(ToMiRounded(k8sQuantity)) + "Mi"
}
func (quantity *Quantity) ToMiRounded() int64 {
	return ToMiRounded(quantity.Quantity)
}
func (quantity *Quantity) ToMiRoundedStr() string {
	return ToMiRoundedStr(quantity.Quantity)
}

func ToMiCeil(k8sQuantity kresource.Quantity) int64 {
	miFloat := float64(k8sQuantity.Value()) / float64(1024*1024)
	return int64(math.Ceil(miFloat))
}
func ToMiCeilStr(k8sQuantity kresource.Quantity) string {
	return s.Int64(ToMiCeil(k8sQuantity)) + "Mi"
}
func (quantity *Quantity) ToMiCeil() int64 {
	return ToMiCeil(quantity.Quantity)
}
func (quantity *Quantity) ToMiCeilStr() string {
	return ToMiCeilStr(quantity.Quantity)
}

func ToMiFloor(k8sQuantity kresource.Quantity) int64 {
	miFloat := float64(k8sQuantity.Value()) / float64(1024*1024)
	return int64(math.Floor(miFloat))
}
func ToMiFloorStr(k8sQuantity kresource.Quantity) string {
	return s.Int64(ToMiFloor(k8sQuantity)) + "Mi"
}
func (quantity *Quantity) ToMiFloor() int64 {
	return ToMiFloor(quantity.Quantity)
}
func (quantity *Quantity) ToMiFloorStr() string {
	return ToMiFloorStr(quantity.Quantity)
}

func (quantity *Quantity) Sub(q2 kresource.Quantity) {
	quantity.Quantity.Sub(q2)
	quantity.UserString = ""
}

func (quantity *Quantity) SubQty(q2 Quantity) {
	quantity.Quantity.Sub(q2.Quantity)
	quantity.UserString = ""
}

func (quantity *Quantity) Add(q2 kresource.Quantity) {
	quantity.Quantity.Add(q2)
	quantity.UserString = ""
}

func (quantity *Quantity) AddQty(q2 Quantity) {
	quantity.Quantity.Add(q2.Quantity)
	quantity.UserString = ""
}

func (quantity *Quantity) String() string {
	if quantity == nil {
		return "<nil>"
	}
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

func (quantity *Quantity) DeepCopy() Quantity {
	return Quantity{
		Quantity:   quantity.Quantity.DeepCopy(),
		UserString: quantity.UserString,
	}
}

func QuantityPtr(k8sQuantity kresource.Quantity) *kresource.Quantity {
	return &k8sQuantity
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
	Quantity   kresource.Quantity
	UserString string
}

func (quantity Quantity) MarshalYAML() (interface{}, error) {
	return quantity.String(), nil
}

func (quantity *Quantity) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var userString string
	err := unmarshal(&userString)
	if err != nil {
		return err
	}
	err = quantity.UnmarshalJSON([]byte(userString))
	if err != nil {
		return err
	}
	return nil
}

func (quantity Quantity) MarshalJSON() ([]byte, error) {
	return json.Marshal(quantity.String())
}

func (quantity *Quantity) UnmarshalJSON(data []byte) error {
	var userString string
	err := json.Unmarshal(data, &userString)
	quantity.UserString = userString

	parsedQuantity, err := kresource.ParseQuantity(userString)
	if err != nil {
		return err
	}

	quantity.Quantity = parsedQuantity
	quantity.UserString = userString
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
