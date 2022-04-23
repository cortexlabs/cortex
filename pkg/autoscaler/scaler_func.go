/*
Copyright 2022 Cortex Labs, Inc.

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

package autoscaler

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type ScalerFunc struct {
	ScaleFunc                    func(apiName string, request int32) error
	GetInFlightRequestsFunc      func(apiName string, window time.Duration) (*float64, error)
	GetAutoscalingSpecFunc       func(apiName string) (*userconfig.Autoscaling, error)
	CurrentRequestedReplicasFunc func(apiName string) (int32, error)
}

func (s *ScalerFunc) Scale(apiName string, request int32) error {
	if s.ScaleFunc == nil {
		return nil
	}

	return s.ScaleFunc(apiName, request)
}

func (s *ScalerFunc) GetInFlightRequests(apiName string, window time.Duration) (*float64, error) {
	if s.GetInFlightRequestsFunc == nil {
		return nil, nil
	}

	return s.GetInFlightRequestsFunc(apiName, window)
}

func (s *ScalerFunc) GetAutoscalingSpec(apiName string) (*userconfig.Autoscaling, error) {
	if s.GetAutoscalingSpecFunc == nil {
		return nil, nil
	}

	return s.GetAutoscalingSpecFunc(apiName)
}

func (s *ScalerFunc) CurrentRequestedReplicas(apiName string) (int32, error) {
	if s.CurrentRequestedReplicasFunc == nil {
		return 0, nil
	}

	return s.CurrentRequestedReplicasFunc(apiName)
}
