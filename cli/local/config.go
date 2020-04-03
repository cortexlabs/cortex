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
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	dockerclient "github.com/docker/docker/client"
	"github.com/mitchellh/go-homedir"
)

var _cachedDockerClient *dockerclient.Client

var CWD string
var LocalDir string
var LocalWorkspace string
var ModelCacheDir string

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		err := errors.Wrap(err, "unable to determine current working directory")
		exit.Error(err)
	}
	CWD = s.EnsureSuffix(cwd, "/")

	homeDir, err := homedir.Dir()
	if err != nil {
		err := errors.Wrap(err, "unable to determine home directory")
		exit.Error(err)
	}

	LocalDir = filepath.Join(homeDir, ".cortex")
	err = os.MkdirAll(LocalDir, os.ModePerm)
	if err != nil {
		err := errors.Wrap(err, "unable to write to home directory", LocalDir)
		exit.Error(err)
	}

	LocalWorkspace = filepath.Join(LocalDir, "local_workspace")
	err = os.MkdirAll(LocalWorkspace, os.ModePerm)
	if err != nil {
		err := errors.Wrap(err, "unable to write to home directory", LocalWorkspace)
		exit.Error(err)
	}

	ModelCacheDir = filepath.Join(LocalDir, "model_cache")
	err = os.MkdirAll(ModelCacheDir, os.ModePerm)
	if err != nil {
		err := errors.Wrap(err, "unable to write to home directory", ModelCacheDir)
		exit.Error(err)
	}
}

func DockerClient() *dockerclient.Client {
	if _cachedDockerClient != nil {
		return _cachedDockerClient
	}

	var err error
	_cachedDockerClient, err = dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		exit.Error(wrapDockerError(err))
	}

	_cachedDockerClient.NegotiateAPIVersion(context.Background())
	return _cachedDockerClient
}

func wrapDockerError(err error) error {
	if dockerclient.IsErrConnectionFailed(err) {
		return ErrorConnectToDockerDaemon()
	}

	if strings.Contains(strings.ToLower(err.Error()), "permission denied") {
		return ErrorDockerPermissions(err)
	}

	return errors.WithStack(err)
}
