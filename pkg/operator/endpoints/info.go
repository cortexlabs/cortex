/*
Copyright 2020 Cortex Labs, Inc.

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
	"os"

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

func Info(w http.ResponseWriter, r *http.Request) {
	response := schema.InfoResponse{
		MaskedAWSAccessKeyID: s.MaskString(os.Getenv("AWS_ACCESS_KEY_ID"), 4),
		ClusterConfig:        *config.Cluster,
	}
	respond(w, response)
}
