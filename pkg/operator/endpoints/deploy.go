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
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/operator/resources"
)

func Deploy(w http.ResponseWriter, r *http.Request) {
	force := getOptionalBoolQParam("force", false, r)

	configFileName, err := getRequiredQueryParam("configFileName", r)
	if err != nil {
		respondError(w, r, errors.WithStack(err))
		return
	}

	configBytes, err := files.ReadReqFile(r, "config")
	if err != nil {
		respondError(w, r, errors.WithStack(err))
		return
	} else if len(configBytes) == 0 {
		respondError(w, r, ErrorFormFileMustBeProvided("config"))
		return
	}

	projectBytes, err := files.ReadReqFile(r, "project.zip")
	if err != nil {
		respondError(w, r, err)
		return
	}

	response, err := resources.Deploy(projectBytes, configFileName, configBytes, force)
	if err != nil {
		respondError(w, r, err)
		return
	}

	respond(w, response)
}
