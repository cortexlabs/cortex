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
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BatchJobSpec defines the desired state of BatchJob
type BatchJobSpec struct {
	// +kubebuilder:validation:Required
	// Reference to a cortex BatchAPI name
	APIName string `json:"api_name,omitempty"`

	// +kubebuilder:validation:Required
	// Reference to a cortex BatchAPI apiID
	APIId string `json:"api_id,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// Number of workers for the batch job
	Workers int32 `json:"workers,omitempty"`

	// +kubebuilder:validation:Optional
	// Duration until a batch job times out
	Timeout *kmeta.Duration `json:"timeout,omitempty"`

	// +kubebuilder:validation:Optional
	// Configuration for the dead letter queue
	DeadLetterQueue *DeadLetterQueueSpec `json:"dead_letter_queue,omitempty"`

	// +kubebuilder:validation:Optional
	// Compute resource requirements
	Resources *kcore.ResourceRequirements `json:"resources,omitempty"`
}

// DeadLetterQueueSpec defines the desired state for the dead letter queue in a BatchJob
type DeadLetterQueueSpec struct {
	// +kubebuilder:validation:Required
	// arn of the dead letter queue e.g. arn:aws:sqs:us-west-2:123456789:failed.fifo
	ARN string `json:"arn,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// Number of times a batch is allowed to be handled by a worker before it is considered to be failed
	// and transferred to the dead letter queue (must be >= 1)
	MaxReceiveCount int64 `json:"max_receive_count,omitempty"`
}

// BatchJobStatus defines the observed state of BatchJob
type BatchJobStatus struct {
	// Job ID
	ID string `json:"id,omitempty"`

	// Processing start timestamp
	StartTime *kmeta.Time `json:"start_time,omitempty"`

	// Processing ending timestamp
	EndTime *kmeta.Time `json:"end_time,omitempty"`

	// URL for the used SQS queue
	QueueURL string `json:"queue_url,omitempty"`

	// Status of the batch job
	Status string `json:"status,omitempty"`

	// Detailed worker counts with respective status
	WorkerCounts *status.WorkerCounts `json:"worker_counts,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BatchJob is the Schema for the batchjobs API
type BatchJob struct {
	kmeta.TypeMeta   `json:",inline"`
	kmeta.ObjectMeta `json:"metadata,omitempty"`

	Spec   BatchJobSpec   `json:"spec,omitempty"`
	Status BatchJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BatchJobList contains a list of BatchJob
type BatchJobList struct {
	kmeta.TypeMeta `json:",inline"`
	kmeta.ListMeta `json:"metadata,omitempty"`
	Items          []BatchJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BatchJob{}, &BatchJobList{})
}
