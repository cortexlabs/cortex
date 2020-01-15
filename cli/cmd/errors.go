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

package cmd

import (
	"fmt"
	"net/url"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

const (
	_errStrCantMakeRequest = "unable to make request"
	_errStrRead            = "unable to read"
)

func errStrFailedToConnect(u url.URL) string {
	return "failed to connect to " + urls.TrimQueryParamsURL(u)
}

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrAPINotReady
	ErrFailedToConnectOperator
	ErrConfigCannotBeChangedOnUpdate
	ErrDuplicateCLIEnvNames
)

var _errorKinds = []string{
	"err_unknown",
	"err_api_not_ready",
	"err_failed_to_connect_operator",
	"err_config_cannot_be_changed_on_update",
	"err_duplicate_cli_env_names",
}

var _ = [1]int{}[int(ErrDuplicateCLIEnvNames)-(len(_errorKinds)-1)] // Ensure list length matches

func (t ErrorKind) String() string {
	return _errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_errorKinds); i++ {
		if enum == _errorKinds[i] {
			*t = ErrorKind(i)
			return nil
		}
	}

	*t = ErrUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ErrorKind) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ErrorKind) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

type Error struct {
	Kind    ErrorKind
	message string
}

func (e Error) Error() string {
	return e.message
}

func ErrorAPINotReady(apiName string, status string) error {
	return errors.WithStack(Error{
		Kind:    ErrAPINotReady,
		message: fmt.Sprintf("%s api is %s", s.UserStr(apiName), status),
	})
}

func ErrorFailedToConnectOperator(originalError error, operatorURL string) error {
	operatorURLMsg := ""
	if operatorURL != "" {
		operatorURLMsg = fmt.Sprintf(" (%s)", operatorURL)
	}

	originalErrMsg := ""
	if originalError != nil {
		originalErrMsg = urls.TrimQueryParamsStr(originalError.Error()) + "\n\n"
	}

	return errors.WithStack(Error{
		Kind:    ErrFailedToConnectOperator,
		message: fmt.Sprintf("%sfailed to connect to the operator%s, run `cortex configure` if you need to update the operator endpoint", originalErrMsg, operatorURLMsg),
	})
}

func ErrorConfigCannotBeChangedOnUpdate(configKey string, prevVal interface{}) error {
	return errors.WithStack(Error{
		Kind:    ErrConfigCannotBeChangedOnUpdate,
		message: fmt.Sprintf("modifying %s in a running cluster is not supported, please set %s to its previous value: %s", configKey, configKey, s.UserStr(prevVal)),
	})
}

func ErrorDuplicateCLIEnvNames(environment string) error {
	return errors.WithStack(Error{
		Kind:    ErrDuplicateCLIEnvNames,
		message: fmt.Sprintf("duplicate environment names: %s is defined more than once", s.UserStr(environment)),
	})
}
