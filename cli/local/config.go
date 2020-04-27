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

package local

import (
	"os"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/mitchellh/go-homedir"
)

var _cwd string
var _localDir string
var _localWorkspaceDir string
var _modelCacheDir string

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		err := errors.Wrap(err, "unable to determine current working directory")
		exit.Error(err)
	}
	_cwd = s.EnsureSuffix(cwd, "/")

	homeDir, err := homedir.Dir()
	if err != nil {
		err := errors.Wrap(err, "unable to determine home directory")
		exit.Error(err)
	}

	_localDir = filepath.Join(homeDir, ".cortex")
	err = os.MkdirAll(_localDir, os.ModePerm)
	if err != nil {
		err := errors.Wrap(err, "unable to write to home directory", _localDir)
		exit.Error(err)
	}

	_localWorkspaceDir = filepath.Join(_localDir, "workspace")
	err = os.MkdirAll(_localWorkspaceDir, os.ModePerm)
	if err != nil {
		err := errors.Wrap(err, "unable to write to home directory", _localWorkspaceDir)
		exit.Error(err)
	}

	localAPISWorkspaceDir := filepath.Join(_localWorkspaceDir, "apis")
	err = os.MkdirAll(localAPISWorkspaceDir, os.ModePerm)
	if err != nil {
		err := errors.Wrap(err, "unable to write to home directory", localAPISWorkspaceDir)
		exit.Error(err)
	}

	_modelCacheDir = filepath.Join(_localDir, "model_cache")
	err = os.MkdirAll(_modelCacheDir, os.ModePerm)
	if err != nil {
		err := errors.Wrap(err, "unable to write to home directory", _modelCacheDir)
		exit.Error(err)
	}
}
