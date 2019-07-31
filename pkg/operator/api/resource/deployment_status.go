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

package resource

type DeploymentStatus int

const (
	UnknownDeploymentStatus DeploymentStatus = iota
	UpdatedDeploymentStatus
	UpdatingDeploymentStatus
	ErrorDeploymentStatus
)

var deploymentStatuses = []string{
	"unknown",
	"updated",
	"updating",
	"error",
}

func DeploymentStatusFromString(s string) DeploymentStatus {
	for i := 0; i < len(deploymentStatuses); i++ {
		if s == deploymentStatuses[i] {
			return DeploymentStatus(i)
		}
	}
	return UnknownDeploymentStatus
}

func DeploymentStatusStrings() []string {
	return deploymentStatuses[1:]
}

func (t DeploymentStatus) String() string {
	return deploymentStatuses[t]
}

// MarshalText satisfies TextMarshaler
func (t DeploymentStatus) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *DeploymentStatus) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(deploymentStatuses); i++ {
		if enum == deploymentStatuses[i] {
			*t = DeploymentStatus(i)
			return nil
		}
	}

	*t = UnknownDeploymentStatus
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *DeploymentStatus) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t DeploymentStatus) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
