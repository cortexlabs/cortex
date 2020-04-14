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

package docker

import (
	goerrs "errors"
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrImageInaccessible = "docker.image_inaccessible"
)

func ErrorImageInaccessible(image string, cause error) error {
	// the concrete type of docker client errors (cause) is string
	// it's just better to put them on a new line
	// because they are quite verbosy
	return errors.WithStack(&errors.Error{
		Kind:    ErrImageInaccessible,
		Message: fmt.Sprintf("%s is not accessible", image),
		Cause:   goerrs.New("docker client:\n" + cause.Error()),
	})
}
