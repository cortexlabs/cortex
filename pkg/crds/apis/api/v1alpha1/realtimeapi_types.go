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

package v1alpha1

import (
	"github.com/cortexlabs/cortex/pkg/types/status"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// RealtimeAPISpec defines the desired state of RealtimeAPI
type RealtimeAPISpec struct {
	// Pod configuration
	// +kubebuilder:validation:Required
	Pod PodSpec `json:"pod"`

	// +kubebuilder:validation:Optional
	// Autoscaling configuration
	Autoscaling AutoscalingSpec `json:"autoscaling"`

	// +kubebuilder:validation:Optional
	// List of node groups on which this API can run (default: all node groups are eligible)
	NodeGroups []string `json:"node_groups,omitempty"`

	// +kubebuilder:validation:Optional
	// Deployment strategy to use when replacing existing replicas with new ones
	UpdateStrategy UpdateStratagySpec `json:"update_strategy"`

	// +kubebuilder:validation:Optional
	// Networking configuration
	Networking NetworkingSpec `json:"networking"`
}

type PodSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:default=8080
	// Port to which requests will be sent to
	Port int32 `json:"port"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=1
	// Maximum number of requests that will be concurrently sent into the container
	MaxConcurrency int32 `json:"max_concurrency"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=100
	// Maximum number of requests per replica which will be queued
	// (beyond max_concurrency) before requests are rejected with error code 503
	MaxQueueLength int32 `json:"max_queue_length"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=1
	// Number of desired replicas
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:Required
	// Configurations for the containers to run
	Containers []ContainerSpec `json:"containers"`
}

type ContainerSpec struct {
	// +kubebuilder:validation:Required
	// Name of the container
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// Docker image to use for the container
	Image string `json:"image"`

	// +kubebuilder:validation:Optional
	// Entrypoint (not executed within a shell)
	Command []string `json:"command,omitempty"`

	// +kubebuilder:validation:Optional
	// Arguments to the entrypoint
	Args []string `json:"args,omitempty"`

	// +kubebuilder:validation:Optional
	// Environment variables to set in the container
	Env []kcore.EnvVar `json:"env,omitempty"`

	// +kubebuilder:validation:Optional
	// Compute resource requests
	Compute *ComputeSpec `json:"compute,omitempty"`

	// +kubebuilder:validation:Optional
	// Periodic probe of container readiness;
	// traffic will not be sent into the pod unless all containers' readiness probes are succeeding
	ReadinessProbe *kcore.Probe `json:"readiness_probe,omitempty"`

	// +kubebuilder:validation:Optional
	// Periodic probe of container liveness; container will be restarted if the probe fails
	LivenessProbe *kcore.Probe `json:"liveness_probe,omitempty"`
}

type ComputeSpec struct {
	// +kubebuilder:validation:Optional
	// CPU request for the container; one unit of CPU corresponds to one virtual CPU;
	// fractional requests are allowed, and can be specified as a floating point number or via the "m" suffix
	CPU *resource.Quantity `json:"cpu,omitempty"`

	// +kubebuilder:validation:Optional
	// GPU request for the container; one unit of GPU corresponds to one virtual GPU
	GPU int64 `json:"gpu,omitempty"`

	// +kubebuilder:validation:Optional
	// Inferentia request for the container; one unit of Inf corresponds to one virtual Inf chip
	Inf int64 `json:"inf,omitempty"`

	// +kubebuilder:validation:Optional
	// Memory request for the container;
	// one unit of memory is one byte and can be expressed as an integer or by using one of these suffixes: K, M, G, T
	// (or their power-of two counterparts: Ki, Mi, Gi, Ti)
	Mem *resource.Quantity `json:"mem,omitempty"`

	// +kubebuilder:validation:Optional
	// Size of shared memory (/dev/shm) for sharing data between multiple processes
	Shm *resource.Quantity `json:"shm,omitempty"`
}

type AutoscalingSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// Minimum number of replicas
	MinReplicas int32 `json:"min_replicas,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=100
	// Maximum number of replicas
	MaxReplicas int32 `json:"max_replicas,omitempty"`

	// +kubebuilder:validation:Optional
	// Desired number of in-flight requests per replica (including requests actively being processed as well as queued),
	// which the autoscaler tries to maintain
	TargetInFlight int32 `json:"target_in_flight,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="60s"
	// Duration over which to average the API's in-flight requests per replica
	Window kmeta.Duration `json:"window,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// The API will not scale below the highest recommendation made during this period
	DownscaleStabilizationPeriod kmeta.Duration `json:"downscale_stabilization_period,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// The API will not scale above the lowest recommendation made during this period
	UpscaleStabilizationPeriod kmeta.Duration `json:"upscale_stabilization_period,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="750m"
	// Maximum factor by which to scale down the API on a single scaling event
	MaxDownscaleFactor resource.Quantity `json:"max_downscale_factor,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1500m"
	// Maximum factor by which to scale up the API on a single scaling event
	MaxUpscaleFactor resource.Quantity `json:"max_upscale_factor,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="50m"
	// Any recommendation falling within this factor below the current number of replicas will not trigger a
	// scale down event
	DownscaleTolerance resource.Quantity `json:"downscale_tolerance,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="50m"
	// Any recommendation falling within this factor above the current number of replicas will not trigger a scale up event
	UpscaleTolerance resource.Quantity `json:"upscale_tolerance,omitempty"`
}

type UpdateStratagySpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="25%"
	// Maximum number of replicas that can be scheduled above the desired number of replicas during an update;
	// can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
	// (set to 0 to disable rolling updates)
	MaxSurge intstr.IntOrString `json:"max_surge"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="25%"
	// maximum number of replicas that can be unavailable during an update; can be an absolute number,
	// e.g. 5, or a percentage of desired replicas, e.g. 10%
	MaxUnavailable intstr.IntOrString `json:"max_unavailable"`
}

type NetworkingSpec struct {
	// +kubebuilder:validation:Optional
	// Endpoint for the API
	Endpoint string `json:"endpoint,omitempty"`
}

// RealtimeAPIStatus defines the observed state of RealtimeAPI
type RealtimeAPIStatus struct {
	Status          status.Code `json:"status"`
	DesiredReplicas int32       `json:"desired_replicas"`
	CurrentReplicas int32       `json:"current_replicas"`
	ReadyReplicas   int32       `json:"ready_replicas"`
	Endpoint        string      `json:"endpoint,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RealtimeAPI is the Schema for the realtimeapis API
type RealtimeAPI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RealtimeAPISpec   `json:"spec,omitempty"`
	Status RealtimeAPIStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RealtimeAPIList contains a list of RealtimeAPI
type RealtimeAPIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RealtimeAPI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RealtimeAPI{}, &RealtimeAPIList{})
}
