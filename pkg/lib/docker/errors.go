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

package docker

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrConnectToDockerDaemon   = "docker.connect_to_docker_daemon"
	ErrDockerPermissions       = "docker.docker_permissions"
	ErrImageDoesntExistLocally = "docker.image_doesnt_exist_locally"
	ErrImageInaccessible       = "docker.image_inaccessible"
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
		groupAddStr = " (e.g. by running `sudo groupadd docker; sudo gpasswd -a $USER docker` and then restarting your terminal)"
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrDockerPermissions,
		Message: errStr + "\n\nyou can re-run this command with `sudo`, or grant your current user access to docker" + groupAddStr,
	})
}

func ErrorImageDoesntExistLocally(image string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrImageDoesntExistLocally,
		Message: fmt.Sprintf("%s does not exist locally; download it with `docker pull %s` (if your registry is private, run `docker login` first)", image, image),
	})
}

func ErrorImageInaccessible(image string, cause error) error {
	message := fmt.Sprintf("%s is not accessible", image)

	if cause != nil {
		message += "\n" + errors.Message(cause) // add \n because docker client errors are verbose but useful
	}

	if strings.Contains(cause.Error(), "auth") {
		message += fmt.Sprintf("\n\nif you would like to use a private docker registry, see https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor)
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrImageInaccessible,
		Message: message,
		Cause:   cause,
	})
}
