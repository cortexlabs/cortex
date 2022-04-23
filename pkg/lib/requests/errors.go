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

package requests

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

const (
	_errStrCantMakeRequest = "unable to make request"
	_errStrRead            = "unable to read"
)

func errStrFailedToConnect(u url.URL) string {
	return "failed to connect to " + urls.TrimQueryParamsURL(u)
}

const (
	ErrResponseUnknown = "requests.response_unknown"
)

func ErrorResponseUnknown(body string, statusCode int) error {
	msg := body
	if strings.TrimSpace(body) == "" {
		msg = fmt.Sprintf("empty response (status code %d)", statusCode)
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrResponseUnknown,
		Message: msg,
	})
}
