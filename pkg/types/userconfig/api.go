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

package userconfig

import (
	"fmt"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/yaml"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type API struct {
	Resource
	APIs             []*TrafficSplit `json:"apis" yaml:"apis"`
	Predictor        *Predictor      `json:"predictor" yaml:"predictor"`
	TaskDefinition   *TaskDefinition `json:"definition" yaml:"definition"`
	Networking       *Networking     `json:"networking" yaml:"networking"`
	Compute          *Compute        `json:"compute" yaml:"compute"`
	Autoscaling      *Autoscaling    `json:"autoscaling" yaml:"autoscaling"`
	UpdateStrategy   *UpdateStrategy `json:"update_strategy" yaml:"update_strategy"`
	Index            int             `json:"index" yaml:"-"`
	FileName         string          `json:"file_name" yaml:"-"`
	SubmittedAPISpec interface{}     `json:"submitted_api_spec" yaml:"submitted_api_spec"`
}

type Predictor struct {
	Type PredictorType `json:"type" yaml:"type"`
	Path string        `json:"path" yaml:"path"`

	MultiModelReloading *MultiModels `json:"multi_model_reloading" yaml:"multi_model_reloading"`
	Models              *MultiModels `json:"models" yaml:"models"`

	ServerSideBatching     *ServerSideBatching    `json:"server_side_batching" yaml:"server_side_batching"`
	ProcessesPerReplica    int32                  `json:"processes_per_replica" yaml:"processes_per_replica"`
	ThreadsPerProcess      int32                  `json:"threads_per_process" yaml:"threads_per_process"`
	ShmSize                *k8s.Quantity          `json:"shm_size" yaml:"shm_size"`
	PythonPath             *string                `json:"python_path" yaml:"python_path"`
	LogLevel               LogLevel               `json:"log_level" yaml:"log_level"`
	Image                  string                 `json:"image" yaml:"image"`
	TensorFlowServingImage string                 `json:"tensorflow_serving_image" yaml:"tensorflow_serving_image"`
	Config                 map[string]interface{} `json:"config" yaml:"config"`
	Env                    map[string]string      `json:"env" yaml:"env"`
	Dependencies           *Dependencies          `json:"dependencies" yaml:"dependencies"`
}

type TaskDefinition struct {
	Path       string                 `json:"path" yaml:"path"`
	PythonPath *string                `json:"python_path" yaml:"python_path"`
	Image      string                 `json:"image" yaml:"image"`
	LogLevel   LogLevel               `json:"log_level" yaml:"log_level"`
	Config     map[string]interface{} `json:"config" yaml:"config"`
	Env        map[string]string      `json:"env" yaml:"env"`
}

type MultiModels struct {
	Path          *string          `json:"path" yaml:"path"`
	Paths         []*ModelResource `json:"paths" yaml:"paths"`
	Dir           *string          `json:"dir" yaml:"dir"`
	CacheSize     *int32           `json:"cache_size" yaml:"cache_size"`
	DiskCacheSize *int32           `json:"disk_cache_size" yaml:"disk_cache_size"`
	SignatureKey  *string          `json:"signature_key" yaml:"signature_key"`
}

type TrafficSplit struct {
	Name   string `json:"name" yaml:"name"`
	Weight int32  `json:"weight" yaml:"weight"`
}

type ModelResource struct {
	Name         string  `json:"name" yaml:"name"`
	Path         string  `json:"path" yaml:"path"`
	SignatureKey *string `json:"signature_key" yaml:"signature_key"`
}

type ServerSideBatching struct {
	MaxBatchSize  int32         `json:"max_batch_size" yaml:"max_batch_size"`
	BatchInterval time.Duration `json:"batch_interval" yaml:"batch_interval"`
}

type Dependencies struct {
	Pip   string `json:"pip" yaml:"pip"`
	Conda string `json:"conda" yaml:"conda"`
	Shell string `json:"shell" yaml:"shell"`
}

type Networking struct {
	Endpoint *string `json:"endpoint" yaml:"endpoint"`
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
	for _, model := range api.Predictor.Models.Paths {
		names = append(names, model.Name)
	}
	return names
}

func (api *API) ApplyDefaultDockerPaths() {
	usesGPU := api.Compute.GPU > 0
	usesInf := api.Compute.Inf > 0

	switch api.Kind {
	case RealtimeAPIKind, BatchAPIKind:
		api.applyPredictorDefaultDockerPaths(usesGPU, usesInf)
	case TaskAPIKind:
		api.applyTaskDefaultDockerPaths(usesGPU, usesInf)
	}
}

func (api *API) applyPredictorDefaultDockerPaths(usesGPU, usesInf bool) {
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

func (api *API) applyTaskDefaultDockerPaths(usesGPU, usesInf bool) {
	task := api.TaskDefinition
	if task.Image == "" {
		if usesGPU {
			task.Image = consts.DefaultImagePythonPredictorGPU
		} else if usesInf {
			task.Image = consts.DefaultImagePythonPredictorInf
		} else {
			task.Image = consts.DefaultImagePythonPredictorCPU
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
	annotations := map[string]string{}
	if api.Predictor != nil {
		annotations[ProcessesPerReplicaAnnotationKey] = s.Int32(api.Predictor.ProcessesPerReplica)
		annotations[ThreadsPerProcessAnnotationKey] = s.Int32(api.Predictor.ThreadsPerProcess)
	}

	if api.Networking != nil {
		annotations[EndpointAnnotationKey] = *api.Networking.Endpoint
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

	if api.TaskDefinition != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", TaskDefinitionKey))
		sb.WriteString(s.Indent(api.TaskDefinition.UserStr(), "  "))
	}

	if api.Predictor != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", PredictorKey))
		sb.WriteString(s.Indent(api.Predictor.UserStr(), "  "))
	}

	if api.Networking != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", NetworkingKey))
		sb.WriteString(s.Indent(api.Networking.UserStr(), "  "))
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

func (trafficSplit *TrafficSplit) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", NameKey, trafficSplit.Name))
	sb.WriteString(fmt.Sprintf("%s: %s\n", WeightKey, s.Int32(trafficSplit.Weight)))
	return sb.String()
}

func (task *TaskDefinition) UserStr() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s: %s\n", PathKey, task.Path))
	if task.PythonPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", PythonPathKey, *task.PythonPath))
	}
	sb.WriteString(fmt.Sprintf("%s: %s\n", ImageKey, task.Image))
	sb.WriteString(fmt.Sprintf("%s: %s\n", LogLevelKey, task.LogLevel))
	if len(task.Config) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", ConfigKey))
		d, _ := yaml.Marshal(&task.Config)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	if len(task.Env) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", EnvKey))
		d, _ := yaml.Marshal(&task.Env)
		sb.WriteString(s.Indent(string(d), "  "))
	}

	return sb.String()
}

func (predictor *Predictor) UserStr() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s: %s\n", TypeKey, predictor.Type))
	sb.WriteString(fmt.Sprintf("%s: %s\n", PathKey, predictor.Path))

	if predictor.Models != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ModelsKey))
		sb.WriteString(s.Indent(predictor.Models.UserStr(), "  "))
	}
	if predictor.MultiModelReloading != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", MultiModelReloadingKey))
		sb.WriteString(s.Indent(predictor.MultiModelReloading.UserStr(), "  "))
	}

	if predictor.Type == TensorFlowPredictorType && predictor.ServerSideBatching != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ServerSideBatchingKey))
		sb.WriteString(s.Indent(predictor.ServerSideBatching.UserStr(), "  "))
	}

	sb.WriteString(fmt.Sprintf("%s: %s\n", ProcessesPerReplicaKey, s.Int32(predictor.ProcessesPerReplica)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", ThreadsPerProcessKey, s.Int32(predictor.ThreadsPerProcess)))

	if predictor.ShmSize != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ShmSize, predictor.ShmSize.UserString))
	}

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

	sb.WriteString(fmt.Sprintf("%s: %s\n", LogLevelKey, predictor.LogLevel))

	if len(predictor.Env) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", EnvKey))
		d, _ := yaml.Marshal(&predictor.Env)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	return sb.String()
}

func (models *MultiModels) UserStr() string {
	var sb strings.Builder

	if len(models.Paths) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", ModelsPathsKey))
		for _, model := range models.Paths {
			modelUserStr := s.Indent(model.UserStr(), "    ")
			modelUserStr = modelUserStr[:2] + "-" + modelUserStr[3:]
			sb.WriteString(modelUserStr)
		}
	} else if models.Path != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelsPathKey, *models.Path))
	} else {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelsDirKey, *models.Dir))
	}
	if models.SignatureKey != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelsSignatureKeyKey, *models.SignatureKey))
	}
	if models.CacheSize != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelsCacheSizeKey, s.Int32(*models.CacheSize)))
	}
	if models.DiskCacheSize != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelsDiskCacheSizeKey, s.Int32(*models.DiskCacheSize)))
	}
	return sb.String()
}

func (model *ModelResource) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelsNameKey, model.Name))
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelsPathKey, model.Path))
	if model.SignatureKey != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelsSignatureKeyKey, *model.SignatureKey))
	}
	return sb.String()
}

func (batch *ServerSideBatching) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxBatchSizeKey, s.Int32(batch.MaxBatchSize)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", BatchIntervalKey, batch.BatchInterval))
	return sb.String()
}

func (networking *Networking) UserStr() string {
	var sb strings.Builder
	if networking.Endpoint != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", EndpointKey, *networking.Endpoint))
	}
	return sb.String()
}

// Represent compute using the smallest base units e.g. bytes for Mem, milli for CPU
func (compute *Compute) Normalized() string {
	var sb strings.Builder
	if compute.CPU == nil {
		sb.WriteString(fmt.Sprintf("%s: null\n", CPUKey))
	} else {
		sb.WriteString(fmt.Sprintf("%s: %d\n", CPUKey, compute.CPU.MilliValue()))
	}
	if compute.GPU > 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", GPUKey, s.Int64(compute.GPU)))
	}
	if compute.Inf > 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", InfKey, s.Int64(compute.Inf)))
	}
	if compute.Mem == nil {
		sb.WriteString(fmt.Sprintf("%s: null\n", MemKey))
	} else {
		sb.WriteString(fmt.Sprintf("%s: %d\n", MemKey, compute.Mem.Value()))
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
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxReplicaConcurrencyKey, s.Int64(autoscaling.MaxReplicaConcurrency)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", TargetReplicaConcurrencyKey, s.Float64(*autoscaling.TargetReplicaConcurrency)))
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

func (api *API) TelemetryEvent(provider types.ProviderType) map[string]interface{} {
	event := map[string]interface{}{
		"provider": provider,
		"kind":     api.Kind,
	}

	if len(api.APIs) > 0 {
		event["apis._is_defined"] = true
		event["apis._len"] = len(api.APIs)
	}

	if api.Networking != nil {
		event["networking._is_defined"] = true
		if api.Networking.Endpoint != nil {
			event["networking.endpoint._is_defined"] = true
			if urls.CanonicalizeEndpoint(api.Name) != *api.Networking.Endpoint {
				event["networking.endpoint._is_custom"] = true
			}
		}
	}

	if api.Compute != nil {
		event["compute._is_defined"] = true
		if api.Compute.CPU != nil {
			event["compute.cpu._is_defined"] = true
			event["compute.cpu"] = float64(api.Compute.CPU.MilliValue()) / 1000
		}
		if api.Compute.Mem != nil {
			event["compute.mem._is_defined"] = true
			event["compute.mem"] = api.Compute.Mem.Value()
		}
		event["compute.gpu"] = api.Compute.GPU
		event["compute.inf"] = api.Compute.Inf
	}

	if api.Predictor != nil {
		event["predictor._is_defined"] = true
		event["predictor.type"] = api.Predictor.Type
		event["predictor.processes_per_replica"] = api.Predictor.ProcessesPerReplica
		event["predictor.threads_per_process"] = api.Predictor.ThreadsPerProcess

		if api.Predictor.ShmSize != nil {
			event["predictor.shm_size"] = api.Predictor.ShmSize.String()
		}

		event["predictor.log_level"] = api.Predictor.LogLevel

		if api.Predictor.PythonPath != nil {
			event["predictor.python_path._is_defined"] = true
		}
		if !strings.HasPrefix(api.Predictor.Image, "cortexlabs/") {
			event["predictor.image._is_custom"] = true
		}
		if !strings.HasPrefix(api.Predictor.TensorFlowServingImage, "cortexlabs/") {
			event["predictor.tensorflow_serving_image._is_custom"] = true
		}
		if len(api.Predictor.Config) > 0 {
			event["predictor.config._is_defined"] = true
			event["predictor.config._len"] = len(api.Predictor.Config)
		}
		if len(api.Predictor.Env) > 0 {
			event["predictor.env._is_defined"] = true
			event["predictor.env._len"] = len(api.Predictor.Env)
		}

		var models *MultiModels
		if api.Predictor.Models != nil {
			models = api.Predictor.Models
		}
		if api.Predictor.MultiModelReloading != nil {
			models = api.Predictor.MultiModelReloading
		}

		if models != nil {
			event["predictor.models._is_defined"] = true
			if models.Path != nil {
				event["predictor.models.path._is_defined"] = true
			}
			if len(models.Paths) > 0 {
				event["predictor.models.paths._is_defined"] = true
				event["predictor.models.paths._len"] = len(models.Paths)
				var numSignatureKeysDefined int
				for _, mmPath := range models.Paths {
					if mmPath.SignatureKey != nil {
						numSignatureKeysDefined++
					}
				}
				event["predictor.models.paths._num_signature_keys_defined"] = numSignatureKeysDefined
			}
			if models.Dir != nil {
				event["predictor.models.dir._is_defined"] = true
			}
			if models.CacheSize != nil {
				event["predictor.models.cache_size._is_defined"] = true
				event["predictor.models.cache_size"] = *models.CacheSize
			}
			if models.DiskCacheSize != nil {
				event["predictor.models.disk_cache_size._is_defined"] = true
				event["predictor.models.disk_cache_size"] = *models.DiskCacheSize
			}
			if models.SignatureKey != nil {
				event["predictor.models.signature_key._is_defined"] = true
			}
		}

		if api.Predictor.ServerSideBatching != nil {
			event["predictor.server_side_batching._is_defined"] = true
			event["predictor.server_side_batching.max_batch_size"] = api.Predictor.ServerSideBatching.MaxBatchSize
			event["predictor.server_side_batching.batch_interval"] = api.Predictor.ServerSideBatching.BatchInterval.Seconds()
		}
	}

	if api.UpdateStrategy != nil {
		event["update_strategy._is_defined"] = true
		event["update_strategy.max_surge"] = api.UpdateStrategy.MaxSurge
		event["update_strategy.max_unavailable"] = api.UpdateStrategy.MaxUnavailable
	}

	if api.Autoscaling != nil {
		event["autoscaling._is_defined"] = true
		event["autoscaling.min_replicas"] = api.Autoscaling.MinReplicas
		event["autoscaling.max_replicas"] = api.Autoscaling.MaxReplicas
		event["autoscaling.init_replicas"] = api.Autoscaling.InitReplicas
		if api.Autoscaling.TargetReplicaConcurrency != nil {
			event["autoscaling.target_replica_concurrency._is_defined"] = true
			event["autoscaling.target_replica_concurrency"] = *api.Autoscaling.TargetReplicaConcurrency
		}
		event["autoscaling.max_replica_concurrency"] = api.Autoscaling.MaxReplicaConcurrency
		event["autoscaling.window"] = api.Autoscaling.Window.Seconds()
		event["autoscaling.downscale_stabilization_period"] = api.Autoscaling.DownscaleStabilizationPeriod.Seconds()
		event["autoscaling.upscale_stabilization_period"] = api.Autoscaling.UpscaleStabilizationPeriod.Seconds()
		event["autoscaling.max_downscale_factor"] = api.Autoscaling.MaxDownscaleFactor
		event["autoscaling.max_upscale_factor"] = api.Autoscaling.MaxUpscaleFactor
		event["autoscaling.downscale_tolerance"] = api.Autoscaling.DownscaleTolerance
		event["autoscaling.upscale_tolerance"] = api.Autoscaling.UpscaleTolerance
	}

	return event
}
