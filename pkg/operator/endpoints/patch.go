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

package endpoints

import (
	"io/ioutil"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/resources"
)

func Patch(w http.ResponseWriter, r *http.Request) {
	force := getOptionalBoolQParam("force", false, r)

	configFileName, err := getRequiredQueryParam("configFileName", r)
	if err != nil {
		respondError(w, r, errors.WithStack(err))
		return
	}

	rw := http.MaxBytesReader(w, r.Body, 10<<20)

	bodyBytes, err := ioutil.ReadAll(rw)
	if err != nil {
		respondError(w, r, err)
		return
	}

	response, err := resources.Patch(bodyBytes, configFileName, force)
	if err != nil {
		respondError(w, r, err)
		return
	}

	respond(w, response)
}
