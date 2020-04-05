package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
)

func EncodeAuthConfig(authConfig dockertypes.AuthConfig) (string, error) {
	encoded, err := json.Marshal(authConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to encode docker login credentials")
	}
	registryAuth := base64.URLEncoding.EncodeToString(encoded)
	return registryAuth, nil
}

func IsImageAccessible(client *dockerclient.Client, dockerImage, registryAuth string) bool {
	if _, err := client.DistributionInspect(context.Background(), dockerImage, registryAuth); err != nil {
		return false
	}
	return true
}
