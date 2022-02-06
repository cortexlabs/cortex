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

package dequeuer

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrUserContainerResponseStatusCode        = "dequeuer.user_container_response_status_code"
	ErrUserContainerResponseMissingJSONHeader = "dequeuer.user_container_response_missing_json_header"
	ErrUserContainerResponseNotJSONDecodable  = "dequeuer.user_container_response_not_json_decodable"
	ErrUserContainerNotReachable              = "dequeuer.user_container_not_reachable"
)

func ErrorUserContainerResponseStatusCode(statusCode int) error {
	return &errors.Error{
		Kind:        ErrUserContainerResponseStatusCode,
		Message:     fmt.Sprintf("invalid response from user container; got status code %d, expected status code 200", statusCode),
		NoTelemetry: true,
	}
}

func ErrorUserContainerResponseMissingJSONHeader() error {
	return &errors.Error{
		Kind:        ErrUserContainerResponseMissingJSONHeader,
		Message:     "invalid response from user container; response content type header is not 'application/json'",
		NoTelemetry: true,
	}
}

func ErrorUserContainerResponseNotJSONDecodable() error {
	return &errors.Error{
		Kind:        ErrUserContainerResponseNotJSONDecodable,
		Message:     "invalid response from user container; response is not json decodable",
		NoTelemetry: true,
	}
}

func ErrorUserContainerNotReachable(err error) error {
	return &errors.Error{
		Kind:        ErrUserContainerNotReachable,
		Message:     fmt.Sprintf("user container not reachable: %v", err),
		NoTelemetry: true,
	}
}
