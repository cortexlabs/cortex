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

package autoscaler

import (
	serverless "github.com/cortexlabs/cortex/pkg/crds/apis/serverless/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libstrings "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func generateAutoscalingFromServerlessRealtimeAPI(realtimeAPI serverless.RealtimeAPI) (*userconfig.Autoscaling, error) {
	targetInFlight, ok := libstrings.ParseFloat64(realtimeAPI.Spec.Autoscaling.TargetInFlight)
	if !ok {
		return nil, errors.ErrorUnexpected("failed to parse target-in-flight requests from autoscaling spec")
	}

	maxDownscaleFactor, ok := libstrings.ParseFloat64(realtimeAPI.Spec.Autoscaling.MaxDownscaleFactor)
	if !ok {
		return nil, errors.ErrorUnexpected("failed to parse max downscale factor from autoscaling spec")
	}

	maxUpscaleFactor, ok := libstrings.ParseFloat64(realtimeAPI.Spec.Autoscaling.MaxUpscaleFactor)
	if !ok {
		return nil, errors.ErrorUnexpected("failed to parse max upscale factor from autoscaling spec")
	}

	downscaleTolerance, ok := libstrings.ParseFloat64(realtimeAPI.Spec.Autoscaling.DownscaleTolerance)
	if !ok {
		return nil, errors.ErrorUnexpected("failed to parse downscale tolerance from autoscaling spec")
	}

	upscaleTolerance, ok := libstrings.ParseFloat64(realtimeAPI.Spec.Autoscaling.UpscaleTolerance)
	if !ok {
		return nil, errors.ErrorUnexpected("failed to parse upscale tolerance from autoscaling spec")
	}

	return &userconfig.Autoscaling{
		MinReplicas:                  realtimeAPI.Spec.Autoscaling.MinReplicas,
		MaxReplicas:                  realtimeAPI.Spec.Autoscaling.MaxReplicas,
		InitReplicas:                 realtimeAPI.Spec.Autoscaling.InitReplicas,
		TargetInFlight:               &targetInFlight,
		Window:                       realtimeAPI.Spec.Autoscaling.Window.Duration,
		DownscaleStabilizationPeriod: realtimeAPI.Spec.Autoscaling.DownscaleStabilizationPeriod.Duration,
		UpscaleStabilizationPeriod:   realtimeAPI.Spec.Autoscaling.UpscaleStabilizationPeriod.Duration,
		MaxDownscaleFactor:           maxDownscaleFactor,
		MaxUpscaleFactor:             maxUpscaleFactor,
		DownscaleTolerance:           downscaleTolerance,
		UpscaleTolerance:             upscaleTolerance,
	}, nil
}
