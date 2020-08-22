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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/yaml"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type API struct {
	Resource
	APIs           []*TrafficSplit `json:"apis" yaml:"apis"`
	Predictor      *Predictor      `json:"predictor" yaml:"predictor"`
	Monitoring     *Monitoring     `json:"monitoring" yaml:"monitoring"`
	Networking     *Networking     `json:"networking" yaml:"networking"`
	Compute        *Compute        `json:"compute" yaml:"compute"`
	Autoscaling    *Autoscaling    `json:"autoscaling" yaml:"autoscaling"`
	UpdateStrategy *UpdateStrategy `json:"update_strategy" yaml:"update_strategy"`
	Index          int             `json:"index" yaml:"-"`
	FileName       string          `json:"file_name" yaml:"-"`
}

type Predictor struct {
	Type                   PredictorType          `json:"type" yaml:"type"`
	Path                   string                 `json:"path" yaml:"path"`
	ModelPath              *string                `json:"model_path" yaml:"model_path"`
	Models                 []*ModelResource       `json:"models" yaml:"models"`
	ServerSideBatching     *ServerSideBatching    `json:"server_side_batching" yaml:"server_side_batching"`
	ProcessesPerReplica    int32                  `json:"processes_per_replica" yaml:"processes_per_replica"`
	ThreadsPerProcess      int32                  `json:"threads_per_process" yaml:"threads_per_process"`
	PythonPath             *string                `json:"python_path" yaml:"python_path"`
	Image                  string                 `json:"image" yaml:"image"`
	TensorFlowServingImage string                 `json:"tensorflow_serving_image" yaml:"tensorflow_serving_image"`
	Config                 map[string]interface{} `json:"config" yaml:"config"`
	Env                    map[string]string      `json:"env" yaml:"env"`
	SignatureKey           *string                `json:"signature_key" yaml:"signature_key"`
}

type TrafficSplit struct {
	Name   string `json:"name" yaml:"name"`
	Weight int32  `json:"weight" yaml:"weight"`
}

type ModelResource struct {
	Name         string  `json:"name" yaml:"name"`
	ModelPath    string  `json:"model_path" yaml:"model_path"`
	SignatureKey *string `json:"signature_key" yaml:"signature_key"`
}

type Monitoring struct {
	Key       *string   `json:"key" yaml:"key"`
	ModelType ModelType `json:"model_type" yaml:"model_type"`
}

type ServerSideBatching struct {
	MaxBatchSize  int32         `json:"max_batch_size" yaml:"max_batch_size"`
	BatchInterval time.Duration `json:"batch_interval" yaml:"batch_interval"`
}

type Networking struct {
	Endpoint   *string        `json:"endpoint" yaml:"endpoint"`
	LocalPort  *int           `json:"local_port" yaml:"local_port"`
	APIGateway APIGatewayType `json:"api_gateway" yaml:"api_gateway"`
}

type Compute struct {
	CPU *k8s.Quantity `json:"cpu" yaml:"cpu"`
	Mem *k8s.Quantity `json:"mem" yaml:"mem"`
	GPU int64         `json:"gpu" yaml:"gpu"`
	Inf int64         `json:"inf" yaml:"inf"`
}

type Autoscaling struct {
	MinReplicas                  int32         `json:"min_replicas" yaml:"min_replicas"`
	MaxReplicas                  int32         `json:"max_replicas" yaml:"max_replicas"`
	InitReplicas                 int32         `json:"init_replicas" yaml:"init_replicas"`
	TargetReplicaConcurrency     *float64      `json:"target_replica_concurrency" yaml:"target_replica_concurrency"`
	MaxReplicaConcurrency        int64         `json:"max_replica_concurrency" yaml:"max_replica_concurrency"`
	Window                       time.Duration `json:"window" yaml:"window"`
	DownscaleStabilizationPeriod time.Duration `json:"downscale_stabilization_period" yaml:"downscale_stabilization_period"`
	UpscaleStabilizationPeriod   time.Duration `json:"upscale_stabilization_period" yaml:"upscale_stabilization_period"`
	MaxDownscaleFactor           float64       `json:"max_downscale_factor" yaml:"max_downscale_factor"`
	MaxUpscaleFactor             float64       `json:"max_upscale_factor" yaml:"max_upscale_factor"`
	DownscaleTolerance           float64       `json:"downscale_tolerance" yaml:"downscale_tolerance"`
	UpscaleTolerance             float64       `json:"upscale_tolerance" yaml:"upscale_tolerance"`
}

type UpdateStrategy struct {
	MaxSurge       string `json:"max_surge" yaml:"max_surge"`
	MaxUnavailable string `json:"max_unavailable" yaml:"max_unavailable"`
}

func (api *API) Identify() string {
	return IdentifyAPI(api.FileName, api.Name, api.Kind, api.Index)
}

func (api *API) ModelNames() []string {
	names := []string{}
	if api != nil && len(api.Predictor.Models) > 0 {
		for _, model := range api.Predictor.Models {
			names = append(names, model.Name)
		}
	}

	return names
}

func (api *API) ApplyDefaultDockerPaths() {
	usesGPU := api.Compute.GPU > 0
	usesInf := api.Compute.Inf > 0

	predictor := api.Predictor
	switch predictor.Type {
	case PythonPredictorType:
		if predictor.Image == "" {
			if usesGPU {
				predictor.Image = consts.DefaultImagePythonPredictorGPU
			} else if usesInf {
				predictor.Image = consts.DefaultImagePythonPredictorInf
			} else {
				predictor.Image = consts.DefaultImagePythonPredictorCPU
			}
		}
	case TensorFlowPredictorType:
		if predictor.Image == "" {
			predictor.Image = consts.DefaultImageTensorFlowPredictor
		}
		if predictor.TensorFlowServingImage == "" {
			if usesGPU {
				predictor.TensorFlowServingImage = consts.DefaultImageTensorFlowServingGPU
			} else if usesInf {
				predictor.TensorFlowServingImage = consts.DefaultImageTensorFlowServingInf
			} else {
				predictor.TensorFlowServingImage = consts.DefaultImageTensorFlowServingCPU
			}
		}
	case ONNXPredictorType:
		if predictor.Image == "" {
			if usesGPU {
				predictor.Image = consts.DefaultImageONNXPredictorGPU
			} else {
				predictor.Image = consts.DefaultImageONNXPredictorCPU
			}
		}
	}
}

func IdentifyAPI(filePath string, name string, kind Kind, index int) string {
	str := ""

	if filePath != "" {
		str += filePath + ": "
	}

	if name != "" {
		str += name
		if kind != UnknownKind {
			str += " (" + kind.String() + ")"
		}
		return str
	} else if index >= 0 {
		return str + "resource at " + s.Index(index)
	}
	return str + "resource"
}

// InitReplicas was left out deliberately
func (api *API) ToK8sAnnotations() map[string]string {
	annotations := map[string]string{
		ProcessesPerReplicaAnnotationKey: s.Int32(api.Predictor.ProcessesPerReplica),
		ThreadsPerProcessAnnotationKey:   s.Int32(api.Predictor.ThreadsPerProcess),
	}

	if api.Networking != nil {
		annotations[EndpointAnnotationKey] = *api.Networking.Endpoint
		annotations[APIGatewayAnnotationKey] = api.Networking.APIGateway.String()
	}

	if api.Autoscaling != nil {
		annotations[MinReplicasAnnotationKey] = s.Int32(api.Autoscaling.MinReplicas)
		annotations[MaxReplicasAnnotationKey] = s.Int32(api.Autoscaling.MaxReplicas)
		annotations[TargetReplicaConcurrencyAnnotationKey] = s.Float64(*api.Autoscaling.TargetReplicaConcurrency)
		annotations[MaxReplicaConcurrencyAnnotationKey] = s.Int64(api.Autoscaling.MaxReplicaConcurrency)
		annotations[WindowAnnotationKey] = api.Autoscaling.Window.String()
		annotations[DownscaleStabilizationPeriodAnnotationKey] = api.Autoscaling.DownscaleStabilizationPeriod.String()
		annotations[UpscaleStabilizationPeriodAnnotationKey] = api.Autoscaling.UpscaleStabilizationPeriod.String()
		annotations[MaxDownscaleFactorAnnotationKey] = s.Float64(api.Autoscaling.MaxDownscaleFactor)
		annotations[MaxUpscaleFactorAnnotationKey] = s.Float64(api.Autoscaling.MaxUpscaleFactor)
		annotations[DownscaleToleranceAnnotationKey] = s.Float64(api.Autoscaling.DownscaleTolerance)
		annotations[UpscaleToleranceAnnotationKey] = s.Float64(api.Autoscaling.UpscaleTolerance)
	}
	return annotations
}

func APIGatewayFromAnnotations(k8sObj kmeta.Object) (APIGatewayType, error) {
	apiGatewayType := APIGatewayTypeFromString(k8sObj.GetAnnotations()[APIGatewayAnnotationKey])
	if apiGatewayType == UnknownAPIGatewayType {
		return UnknownAPIGatewayType, ErrorUnknownAPIGatewayType()
	}
	return apiGatewayType, nil
}

func AutoscalingFromAnnotations(k8sObj kmeta.Object) (*Autoscaling, error) {
	a := Autoscaling{}

	minReplicas, err := k8s.ParseInt32Annotation(k8sObj, MinReplicasAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MinReplicas = minReplicas

	maxReplicas, err := k8s.ParseInt32Annotation(k8sObj, MaxReplicasAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxReplicas = maxReplicas

	targetReplicaConcurrency, err := k8s.ParseFloat64Annotation(k8sObj, TargetReplicaConcurrencyAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.TargetReplicaConcurrency = &targetReplicaConcurrency

	maxReplicaConcurrency, err := k8s.ParseInt64Annotation(k8sObj, MaxReplicaConcurrencyAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxReplicaConcurrency = maxReplicaConcurrency

	window, err := k8s.ParseDurationAnnotation(k8sObj, WindowAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.Window = window

	downscaleStabilizationPeriod, err := k8s.ParseDurationAnnotation(k8sObj, DownscaleStabilizationPeriodAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.DownscaleStabilizationPeriod = downscaleStabilizationPeriod

	upscaleStabilizationPeriod, err := k8s.ParseDurationAnnotation(k8sObj, UpscaleStabilizationPeriodAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.UpscaleStabilizationPeriod = upscaleStabilizationPeriod

	maxDownscaleFactor, err := k8s.ParseFloat64Annotation(k8sObj, MaxDownscaleFactorAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxDownscaleFactor = maxDownscaleFactor

	maxUpscaleFactor, err := k8s.ParseFloat64Annotation(k8sObj, MaxUpscaleFactorAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxUpscaleFactor = maxUpscaleFactor

	downscaleTolerance, err := k8s.ParseFloat64Annotation(k8sObj, DownscaleToleranceAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.DownscaleTolerance = downscaleTolerance

	upscaleTolerance, err := k8s.ParseFloat64Annotation(k8sObj, UpscaleToleranceAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.UpscaleTolerance = upscaleTolerance

	return &a, nil
}

func (api *API) UserStr(provider types.ProviderType) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", NameKey, api.Name))
	sb.WriteString(fmt.Sprintf("%s: %s\n", KindKey, api.Kind.String()))

	if api.Kind == TrafficSplitterKind {
		sb.WriteString(fmt.Sprintf("%s:\n", APIsKey))
		for _, api := range api.APIs {
			sb.WriteString(s.Indent(api.UserStr(), "  "))
		}
	}

	if api.Predictor != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", PredictorKey))
		sb.WriteString(s.Indent(api.Predictor.UserStr(), "  "))
	}

	if api.Networking != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", NetworkingKey))
		sb.WriteString(s.Indent(api.Networking.UserStr(provider), "  "))
	}

	if api.Compute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
		sb.WriteString(s.Indent(api.Compute.UserStr(), "  "))
	}

	if provider != types.LocalProviderType {
		if api.Monitoring != nil {
			sb.WriteString(fmt.Sprintf("%s:\n", MonitoringKey))
			sb.WriteString(s.Indent(api.Monitoring.UserStr(), "  "))
		}

		if api.Autoscaling != nil {
			sb.WriteString(fmt.Sprintf("%s:\n", AutoscalingKey))
			sb.WriteString(s.Indent(api.Autoscaling.UserStr(), "  "))
		}

		if api.UpdateStrategy != nil {
			sb.WriteString(fmt.Sprintf("%s:\n", UpdateStrategyKey))
			sb.WriteString(s.Indent(api.UpdateStrategy.UserStr(), "  "))
		}
	}
	return sb.String()
}

func (trafficSplit *TrafficSplit) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", NameKey, trafficSplit.Name))
	sb.WriteString(fmt.Sprintf("%s: %s\n", WeightKey, s.Int32(trafficSplit.Weight)))
	return sb.String()
}

func (predictor *Predictor) UserStr() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s: %s\n", TypeKey, predictor.Type))
	sb.WriteString(fmt.Sprintf("%s: %s\n", PathKey, predictor.Path))

	if predictor.ModelPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelPathKey, *predictor.ModelPath))
	}
	if predictor.ModelPath == nil && len(predictor.Models) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", ModelsKey))
		for _, model := range predictor.Models {
			sb.WriteString(fmt.Sprintf(s.Indent(model.UserStr(), "  ")))
		}
	}
	if predictor.SignatureKey != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", SignatureKeyKey, *predictor.SignatureKey))
	}

	if predictor.Type == TensorFlowPredictorType && predictor.ServerSideBatching != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ServerSideBatchingKey))
		sb.WriteString(s.Indent(predictor.ServerSideBatching.UserStr(), "  "))
	}

	sb.WriteString(fmt.Sprintf("%s: %s\n", ProcessesPerReplicaKey, s.Int32(predictor.ProcessesPerReplica)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", ThreadsPerProcessKey, s.Int32(predictor.ThreadsPerProcess)))

	if len(predictor.Config) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", ConfigKey))
		d, _ := yaml.Marshal(&predictor.Config)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	sb.WriteString(fmt.Sprintf("%s: %s\n", ImageKey, predictor.Image))
	if predictor.TensorFlowServingImage != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", TensorFlowServingImageKey, predictor.TensorFlowServingImage))
	}
	if predictor.PythonPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", PythonPathKey, *predictor.PythonPath))
	}
	if len(predictor.Env) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", EnvKey))
		d, _ := yaml.Marshal(&predictor.Env)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	return sb.String()
}

func (batch *ServerSideBatching) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxBatchSizeKey, s.Int32(batch.MaxBatchSize)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", BatchIntervalKey, batch.BatchInterval))
	return sb.String()
}

func (model *ModelResource) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- %s: %s\n", ModelsNameKey, model.Name))
	sb.WriteString(fmt.Sprintf(s.Indent("%s: %s\n", "  "), ModelPathKey, model.ModelPath))
	if model.SignatureKey != nil {
		sb.WriteString(fmt.Sprintf(s.Indent("%s: %s\n", "  "), SignatureKeyKey, *model.SignatureKey))
	}
	return sb.String()
}

func (monitoring *Monitoring) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelTypeKey, monitoring.ModelType.String()))
	if monitoring.Key != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", KeyKey, *monitoring.Key))
	}
	return sb.String()
}

func (networking *Networking) UserStr(provider types.ProviderType) string {
	var sb strings.Builder
	if provider == types.LocalProviderType && networking.LocalPort != nil {
		sb.WriteString(fmt.Sprintf("%s: %d\n", LocalPortKey, *networking.LocalPort))
	}
	if provider == types.AWSProviderType && networking.Endpoint != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", EndpointKey, *networking.Endpoint))
	}
	if provider == types.AWSProviderType {
		sb.WriteString(fmt.Sprintf("%s: %s\n", APIGatewayKey, networking.APIGateway))
	}
	return sb.String()
}

func (compute *Compute) UserStr() string {
	var sb strings.Builder
	if compute.CPU == nil {
		sb.WriteString(fmt.Sprintf("%s: null  # no limit\n", CPUKey))
	} else {
		sb.WriteString(fmt.Sprintf("%s: %s\n", CPUKey, compute.CPU.UserString))
	}
	if compute.GPU > 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", GPUKey, s.Int64(compute.GPU)))
	}
	if compute.Inf > 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", InfKey, s.Int64(compute.Inf)))
	}
	if compute.Mem == nil {
		sb.WriteString(fmt.Sprintf("%s: null  # no limit\n", MemKey))
	} else {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MemKey, compute.Mem.UserString))
	}
	return sb.String()
}

func (compute Compute) Equals(c2 *Compute) bool {
	if c2 == nil {
		return false
	}

	if compute.CPU == nil && c2.CPU != nil || compute.CPU != nil && c2.CPU == nil {
		return false
	}

	if compute.CPU != nil && c2.CPU != nil && !compute.CPU.Equal(*c2.CPU) {
		return false
	}

	if compute.Mem == nil && c2.Mem != nil || compute.Mem != nil && c2.Mem == nil {
		return false
	}

	if compute.Mem != nil && c2.Mem != nil && !compute.Mem.Equal(*c2.Mem) {
		return false
	}

	if compute.GPU != c2.GPU {
		return false
	}

	return true
}

func (autoscaling *Autoscaling) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", MinReplicasKey, s.Int32(autoscaling.MinReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxReplicasKey, s.Int32(autoscaling.MaxReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", InitReplicasKey, s.Int32(autoscaling.InitReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", TargetReplicaConcurrencyKey, s.Float64(*autoscaling.TargetReplicaConcurrency)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxReplicaConcurrencyKey, s.Int64(autoscaling.MaxReplicaConcurrency)))
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

func ZeroCompute() Compute {
	return Compute{
		CPU: &k8s.Quantity{},
		Mem: &k8s.Quantity{},
		GPU: 0,
	}
}
