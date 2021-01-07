/*
Copyright 2021 Cortex Labs, Inc.

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

package debug

import (
	"encoding/json"
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/yaml"
	"github.com/davecgh/go-spew/spew"
)

func Pp(obj interface{}) {
	fmt.Println(s.Obj(obj))
}

func Ppg(obj interface{}) {
	fmt.Print(Sppg(obj))
}

func Sppg(obj interface{}) string {
	spew.Config.SortKeys = true
	spew.Config.SpewKeys = true
	spew.Config.Indent = "  "
	spew.Config.ContinueOnMethod = true
	spew.Config.DisablePointerAddresses = true
	spew.Config.DisableCapacities = true
	return spew.Sdump(obj)
}

func Ppj(obj interface{}) {
	b, err := json.MarshalIndent(obj, "", " ")
	if err != nil {
		errors.PrintError(err)
	}
	fmt.Println(string(b))
}

func Ppy(obj interface{}) {
	b, err := yaml.Marshal(obj)
	if err != nil {
		errors.PrintError(err)
	}
	fmt.Println(string(b))
}
