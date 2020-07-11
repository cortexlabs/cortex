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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrAPINotDeployed                = "local.api_not_deployed"
	ErrAPISpecNotFound               = "local.api_specification_not_found"
	ErrCortexVersionMismatch         = "local.cortex_version_mismatch"
	ErrAPIContainersNotFound         = "local.api_containers_not_found"
	ErrFoundContainersWithoutAPISpec = "local.found_containers_without_api_spec"
	ErrInvalidTensorFlowZip          = "local.invalid_tensorflow_zip"
	ErrFailedToDeleteAPISpec         = "local.failed_to_delete_api_spec"
	ErrDuplicateLocalPort            = "local.duplicate_local_port"
	ErrPortAlreadyInUse              = "local.port_already_in_use"
	ErrUnableToFindAvailablePorts    = "local.unable_to_find_available_ports"
)

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
		Kind:    ErrAPIContainersNotFound,
		Message: fmt.Sprintf("unable to find container(s) for %s api", apiName),
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

func ErrorInvalidTensorFlowZip() error {
	message := "invalid TensorFlow zip.\n"
	message += _tfExpectedStructMessage
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidTensorFlowZip,
		Message: message,
	})
}

func ErrorFailedToDeleteAPISpec(path string, err error) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFailedToDeleteAPISpec,
		Message: errors.Message(err) + fmt.Sprintf("\n\nfailed to delete api specification; run `sudo rm -rf %s` to clean up", path),
	})
}

func ErrorDuplicateLocalPort(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateLocalPort,
		Message: fmt.Sprintf("port has already been assigned to api %s, please delete the api with `cortex delete %s --env local` or use another port", apiName, apiName),
	})
}

func ErrorPortAlreadyInUse(port int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrPortAlreadyInUse,
		Message: fmt.Sprintf("port %d is being used by a non-cortex process; please specify a different port or make port %d available", port, port),
	})
}

func ErrorUnableToFindAvailablePorts() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrUnableToFindAvailablePorts,
		Message: fmt.Sprintf("unable to find available ports"),
	})
}
