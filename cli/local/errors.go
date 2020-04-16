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
	"fmt"
	"runtime"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrConnectToDockerDaemon         = "local.connect_to_docker_daemon"
	ErrDockerPermissions             = "local.docker_permissions"
	ErrTensorFlowDirTooManyFiles     = "local.tensorflow_dir_too_many_files"
	ErrAPINotDeployed                = "local.api_not_deployed"
	ErrAPISpecNotFound               = "local.api_specification_not_found"
	ErrCortexVersionMismatch         = "local.err_cortex_version_mismatch"
	ErrAPIContainerNotFound          = "local.api_container_not_found"
	ErrFoundContainersWithoutAPISpec = "local.found_containers_without_api_spec"
	ErrInvalidTensorFlowZip          = "local.invalid_tensorflow_zip"
	ErrFailedToDeleteAPISpec         = "local.failed_to_delete_api_spec"
)

func ErrorConnectToDockerDaemon() error {
	installMsg := "install it by following the instructions for your operating system: https://docs.docker.com/install"
	if strings.HasPrefix(runtime.GOOS, "darwin") {
		installMsg = "install it here: https://docs.docker.com/docker-for-mac/install"
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrConnectToDockerDaemon,
		Message: fmt.Sprintf("unable to connect to the Docker daemon\n\nplease confirm Docker is running, or if Docker is not installed, %s", installMsg),
	})
}

func ErrorDockerPermissions(err error) error {
	errStr := errors.Message(err)

	var groupAddStr string
	if strings.HasPrefix(runtime.GOOS, "linux") {
		groupAddStr = " (e.g. by running `sudo groupadd docker && sudo gpasswd -a $USER docker`)"
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrDockerPermissions,
		Message: errStr + "\n\nyou can re-run this command with `sudo`, or grant your current user access to docker" + groupAddStr,
	})
}

func ErrorTensorFlowDirTooManyFiles(count int32) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrTensorFlowDirTooManyFiles,
		Message: fmt.Sprintf("more than %d many files found in tensorflow directory", count),
	})
}

func ErrorAPINotDeployed(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPINotDeployed,
		Message: fmt.Sprintf("%s is not deployed", apiName), // note: if modifying this string, search the codebase for it and change all occurrences
	})
}

func ErrorAPISpecNotFound(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPISpecNotFound,
		Message: fmt.Sprintf("unable to find configuration for %s api", apiName),
	})
}

func ErrorCortexVersionMismatch(apiName string, apiVersion string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCortexVersionMismatch,
		Message: fmt.Sprintf("api %s was deployed using CLI version %s but the current CLI version is %s; please run `cortex deploy` to redeploy the api or `cortex delete %s` to delete the api", apiName, apiVersion, consts.CortexVersion, apiName),
	})
}

func ErrorFoundContainersWithoutAPISpec(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFoundContainersWithoutAPISpec,
		Message: fmt.Sprintf("unable to find configuration for %s api; please run `cortex delete %s` to perform cleanup and try deploying again", apiName, apiName),
	})
}

func ErrorAPIContainersNotFound(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPIContainerNotFound,
		Message: fmt.Sprintf("unable to find containers for %s api", apiName),
	})
}

var _tfExpectedStructMessage = `For TensorFlow models, the zipped file must be a directory with the following structure:
  1523423423/ (Version prefix, usually a timestamp)
  ├── saved_model.pb
  └── variables/
      ├── variables.index
      ├── variables.data-00000-of-00003
      ├── variables.data-00001-of-00003
      └── variables.data-00002-of-...`

func ErrorInvalidTensorFlowZip(path string) error {
	message := "invalid TensorFlow zip.\n"
	message += _tfExpectedStructMessage
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidTensorFlowZip,
		Message: message,
	})
}

func ErrorFailedToDeleteAPISpec(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFailedToDeleteAPISpec,
		Message: fmt.Sprintf("failed to delete api specification; `sudo rm -rf %s` can be run to manually to delete api specification", path),
	})
}
