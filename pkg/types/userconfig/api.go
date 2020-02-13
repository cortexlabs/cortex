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

package userconfig

import (
	"fmt"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/yaml"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type API struct {
	Name           string          `json:"name" yaml:"name"`
	Endpoint       *string         `json:"endpoint" yaml:"endpoint"`
	Predictor      *Predictor      `json:"predictor" yaml:"predictor"`
	Tracker        *Tracker        `json:"tracker" yaml:"tracker"`
	Compute        *Compute        `json:"compute" yaml:"compute"`
	Autoscaling    *Autoscaling    `json:"autoscaling" yaml:"autoscaling"`
	UpdateStrategy *UpdateStrategy `json:"update_strategy" yaml:"update_strategy"`

	Index    int    `json:"index" yaml:"-"`
	FilePath string `json:"file_path" yaml:"-"`
}

type Predictor struct {
	Type         PredictorType          `json:"type" yaml:"type"`
	Path         string                 `json:"path" yaml:"path"`
	Model        *string                `json:"model" yaml:"model"`
	PythonPath   *string                `json:"python_path" yaml:"python_path"`
	Config       map[string]interface{} `json:"config" yaml:"config"`
	Env          map[string]string      `json:"env" yaml:"env"`
	SignatureKey *string                `json:"signature_key" yaml:"signature_key"`
}

type Tracker struct {
	Key       *string   `json:"key" yaml:"key"`
	ModelType ModelType `json:"model_type" yaml:"model_type"`
}

type Compute struct {
	CPU k8s.Quantity  `json:"cpu" yaml:"cpu"`
	Mem *k8s.Quantity `json:"mem" yaml:"mem"`
	GPU int64         `json:"gpu" yaml:"gpu"`
}

type Autoscaling struct {
	MinReplicas                  int32         `json:"min_replicas" yaml:"min_replicas"`
	MaxReplicas                  int32         `json:"max_replicas" yaml:"max_replicas"`
	InitReplicas                 int32         `json:"init_replicas" yaml:"init_replicas"`
	WorkersPerReplica            int32         `json:"workers_per_replica" yaml:"workers_per_replica"`
	ThreadsPerWorker             int32         `json:"threads_per_worker" yaml:"threads_per_worker"`
	RequestBacklog               int32         `json:"request_backlog" yaml:"request_backlog"`
	TargetQueueLength            float64       `json:"target_queue_length" yaml:"target_queue_length"`
	Window                       time.Duration `json:"window" yaml:"window"`
	DownscaleStabilizationPeriod time.Duration `json:"downscale_stabilization_period" yaml:"downscale_stabilization_period"`
	UpscaleStabilizationPeriod   time.Duration `json:"upscale_stabilization_period" yaml:"upscale_stabilization_period"`
	MaxDownscaleFactor           float64       `json:"max_downscale_factor" yaml:"max_downscale_factor"`
	MaxUpscaleFactor             float64       `json:"max_upscale_factor" yaml:"max_upscale_factor"`
	DownscaleTolerance           float64       `json:"downscale_tolerance" yaml:"downscale_tolerance"`
	UpscaleTolerance             float64       `json:"upscale_tolerance" yaml:"upscale_tolerance"`
}

// InitReplicas was left out deliberately
func (autoscaling *Autoscaling) ToK8sAnnotations() map[string]string {
	return map[string]string{
		MinReplicasAnnotationKey:                  s.Int32(autoscaling.MinReplicas),
		MaxReplicasAnnotationKey:                  s.Int32(autoscaling.MaxReplicas),
		WorkersPerReplicaAnnotationKey:            s.Int32(autoscaling.WorkersPerReplica),
		ThreadsPerWorkerAnnotationKey:             s.Int32(autoscaling.ThreadsPerWorker),
		RequestBacklogAnnotationKey:               s.Int32(autoscaling.RequestBacklog),
		TargetQueueLengthAnnotationKey:            s.Float64(autoscaling.TargetQueueLength),
		WindowAnnotationKey:                       autoscaling.Window.String(),
		DownscaleStabilizationPeriodAnnotationKey: autoscaling.DownscaleStabilizationPeriod.String(),
		UpscaleStabilizationPeriodAnnotationKey:   autoscaling.UpscaleStabilizationPeriod.String(),
		MaxDownscaleFactorAnnotationKey:           s.Float64(autoscaling.MaxDownscaleFactor),
		MaxUpscaleFactorAnnotationKey:             s.Float64(autoscaling.MaxUpscaleFactor),
		DownscaleToleranceAnnotationKey:           s.Float64(autoscaling.DownscaleTolerance),
		UpscaleToleranceAnnotationKey:             s.Float64(autoscaling.UpscaleTolerance),
	}
}

func AutoscalingFromAnnotations(deployment kmeta.Object) (*Autoscaling, error) {
	a := Autoscaling{}

	minReplicas, err := k8s.ParseInt32Annotation(deployment, MinReplicasAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MinReplicas = minReplicas

	maxReplicas, err := k8s.ParseInt32Annotation(deployment, MaxReplicasAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxReplicas = maxReplicas

	workersPerReplica, err := k8s.ParseInt32Annotation(deployment, WorkersPerReplicaAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.WorkersPerReplica = workersPerReplica

	threadsPerWorker, err := k8s.ParseInt32Annotation(deployment, ThreadsPerWorkerAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.ThreadsPerWorker = threadsPerWorker

	requestBacklog, err := k8s.ParseInt32Annotation(deployment, RequestBacklogAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.RequestBacklog = requestBacklog

	targetQueueLength, err := k8s.ParseFloat64Annotation(deployment, TargetQueueLengthAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.TargetQueueLength = targetQueueLength

	window, err := k8s.ParseDurationAnnotation(deployment, WindowAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.Window = window

	downscaleStabilizationPeriod, err := k8s.ParseDurationAnnotation(deployment, DownscaleStabilizationPeriodAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.DownscaleStabilizationPeriod = downscaleStabilizationPeriod

	upscaleStabilizationPeriod, err := k8s.ParseDurationAnnotation(deployment, UpscaleStabilizationPeriodAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.UpscaleStabilizationPeriod = upscaleStabilizationPeriod

	maxDownscaleFactor, err := k8s.ParseFloat64Annotation(deployment, MaxDownscaleFactorAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxDownscaleFactor = maxDownscaleFactor

	maxUpscaleFactor, err := k8s.ParseFloat64Annotation(deployment, MaxUpscaleFactorAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxUpscaleFactor = maxUpscaleFactor

	downscaleTolerance, err := k8s.ParseFloat64Annotation(deployment, DownscaleToleranceAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.DownscaleTolerance = downscaleTolerance

	upscaleTolerance, err := k8s.ParseFloat64Annotation(deployment, UpscaleToleranceAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.UpscaleTolerance = upscaleTolerance

	return &a, nil
}

type UpdateStrategy struct {
	MaxSurge       string `json:"max_surge" yaml:"max_surge"`
	MaxUnavailable string `json:"max_unavailable" yaml:"max_unavailable"`
}

func (api *API) Identify() string {
	return IdentifyAPI(api.FilePath, api.Name, api.Index)
}

func IdentifyAPI(filePath string, name string, index int) string {
	str := ""

	if filePath != "" {
		str += filePath + ": "
	}

	if name != "" {
		return str + name
	} else if index >= 0 {
		return str + "api at " + s.Index(index)
	}
	return str + "api"
}

func (api *API) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", NameKey, api.Name))
	sb.WriteString(fmt.Sprintf("%s: %s\n", EndpointKey, *api.Endpoint))

	sb.WriteString(fmt.Sprintf("%s:\n", PredictorKey))
	sb.WriteString(s.Indent(api.Predictor.UserStr(), "  "))

	if api.Tracker != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", TrackerKey))
		sb.WriteString(s.Indent(api.Tracker.UserStr(), "  "))
	}

	if api.Compute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
		sb.WriteString(s.Indent(api.Compute.UserStr(), "  "))
	}

	if api.Autoscaling != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", AutoscalingKey))
		sb.WriteString(s.Indent(api.Autoscaling.UserStr(), "  "))
	}

	if api.UpdateStrategy != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", UpdateStrategyKey))
		sb.WriteString(s.Indent(api.UpdateStrategy.UserStr(), "  "))
	}

	return sb.String()
}

func (predictor *Predictor) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", TypeKey, predictor.Type))
	sb.WriteString(fmt.Sprintf("%s: %s\n", PathKey, predictor.Path))
	if predictor.Model != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelKey, *predictor.Model))
	}
	if predictor.SignatureKey != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", SignatureKeyKey, *predictor.SignatureKey))
	}
	if predictor.PythonPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", PythonPathKey, *predictor.PythonPath))
	}
	if len(predictor.Config) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", ConfigKey))
		d, _ := yaml.Marshal(&predictor.Config)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	if len(predictor.Env) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", EnvKey))
		d, _ := yaml.Marshal(&predictor.Env)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	return sb.String()
}

func (tracker *Tracker) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelTypeKey, tracker.ModelType.String()))
	if tracker.Key != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", KeyKey, *tracker.Key))
	}
	return sb.String()
}

func (compute *Compute) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", CPUKey, compute.CPU.UserString))
	if compute.GPU > 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", GPUKey, s.Int64(compute.GPU)))
	}
	if compute.Mem != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MemKey, compute.Mem.UserString))
	}
	return sb.String()
}

func (autoscaling *Autoscaling) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", MinReplicasKey, s.Int32(autoscaling.MinReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxReplicasKey, s.Int32(autoscaling.MaxReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", InitReplicasKey, s.Int32(autoscaling.InitReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", WorkersPerReplicaKey, s.Int32(autoscaling.WorkersPerReplica)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", ThreadsPerWorkerKey, s.Int32(autoscaling.ThreadsPerWorker)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", RequestBacklogKey, s.Int32(autoscaling.RequestBacklog)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", TargetQueueLengthKey, s.Float64(autoscaling.TargetQueueLength)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", WindowKey, autoscaling.Window.String()))
	sb.WriteString(fmt.Sprintf("%s: %s\n", DownscaleStabilizationPeriodKey, autoscaling.DownscaleStabilizationPeriod.String()))
	sb.WriteString(fmt.Sprintf("%s: %s\n", UpscaleStabilizationPeriodKey, autoscaling.UpscaleStabilizationPeriod.String()))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxDownscaleFactorKey, s.Float64(autoscaling.MaxDownscaleFactor)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxUpscaleFactorKey, s.Float64(autoscaling.MaxUpscaleFactor)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", DownscaleToleranceKey, s.Float64(autoscaling.DownscaleTolerance)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", UpscaleToleranceKey, s.Float64(autoscaling.UpscaleTolerance)))
	return sb.String()
}

func (updateStrategy *UpdateStrategy) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxSurgeKey, updateStrategy.MaxSurge))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxUnavailableKey, updateStrategy.MaxUnavailable))
	return sb.String()
}
