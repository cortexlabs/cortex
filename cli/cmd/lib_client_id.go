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

package cmd

import (
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/google/uuid"
)

var _cachedClientID string

func clientID() string {
	if _cachedClientID != "" {
		return _cachedClientID
	}

	var err error
	_cachedClientID, err = files.ReadFile(_clientIDPath)
	if err != nil || _cachedClientID == "" {
		_cachedClientID = uuid.New().String()
		files.WriteFile([]byte(_cachedClientID), _clientIDPath)
	}

	return _cachedClientID
}
