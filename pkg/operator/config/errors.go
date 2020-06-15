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

package config

import (
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrNoAPIGateway         = "config.no_api_gateway"
	ErrNoVPCLink            = "config.no_vpc_link"
	ErrNoVPCLinkIntegration = "config.no_vpc_link_integration"
)

func ErrorNoAPIGateway() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoAPIGateway,
		Message: "unable to locate cortex's api gateway",
	})
}

func ErrorNoVPCLink() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoVPCLink,
		Message: "unable to locate cortex's vpc link",
	})
}

func ErrorNoVPCLinkIntegration() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoVPCLinkIntegration,
		Message: "unable to locate cortex's api gateway vpc link integration",
	})
}
