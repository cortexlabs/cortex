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
	"strings"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type App struct {
	Name       string `json:"name" yaml:"name"`
	PythonRoot string `json:"python_root" yaml:"python_root"`
}

var appValidation = &cr.StructValidation{
	DefaultNil: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required:                   true,
				AlphaNumericDashUnderscore: true,
				DNS1123:                    true,
			},
		},
		{
			StructField: "PythonRoot",
			StringValidation: &cr.StringValidation{
				AllowEmpty: true,
			},
		},
		typeFieldValidation,
	},
}

func (app *App) Validate(projectFileMap map[string][]byte) error {
	if len(app.PythonRoot) > 0 {
		validPythonRoot := false
		app.PythonRoot = s.EnsureSuffix(app.PythonRoot, "/")
		for fileKey := range projectFileMap {
			if strings.HasPrefix(fileKey, app.PythonRoot) {
				validPythonRoot = true
				break
			}
		}

		if !validPythonRoot {
			return errors.Wrap(ErrorImplDoesNotExist(app.PythonRoot), "app", PythonRootKey)
		}

	}
	return nil
}
