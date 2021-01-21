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

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/gorilla/mux"
)

func getRequiredPathParam(paramName string, r *http.Request) (string, error) {
	param := mux.Vars(r)[paramName]
	if param == "" {
		return "", ErrorPathParamRequired(paramName)
	}
	return param, nil
}

func getRequiredQueryParam(paramName string, r *http.Request) (string, error) {
	param := r.URL.Query().Get(paramName)
	if param == "" {
		return "", ErrorQueryParamRequired(paramName)
	}
	return param, nil
}

func getOptionalQParam(paramName string, r *http.Request) string {
	return r.URL.Query().Get(paramName)
}

func getOptionalBoolQParam(paramName string, defaultVal bool, r *http.Request) bool {
	param := r.URL.Query().Get(paramName)
	paramBool, ok := s.ParseBool(param)
	if ok {
		return paramBool
	}
	return defaultVal
}
