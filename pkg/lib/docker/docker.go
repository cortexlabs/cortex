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
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
)

var (
	NoAuth string
)

func init() {
	NoAuth, _ = EncodeAuthConfig(dockertypes.AuthConfig{})
}

func EncodeAuthConfig(authConfig dockertypes.AuthConfig) (string, error) {
	encoded, err := json.Marshal(authConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to encode docker login credentials")
	}
	registryAuth := base64.URLEncoding.EncodeToString(encoded)
	return registryAuth, nil
}

func CheckImageAccessible(c *dockerclient.Client, dockerImage, registryAuth string) error {
	if _, err := c.DistributionInspect(context.Background(), dockerImage, registryAuth); err != nil {
		return ErrorImageInaccessible(dockerImage, err)
	}
	return nil
}
