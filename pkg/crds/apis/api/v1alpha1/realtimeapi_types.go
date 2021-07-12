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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RealtimeAPISpec defines the desired state of RealtimeAPI
type RealtimeAPISpec struct {
}

// RealtimeAPIStatus defines the observed state of RealtimeAPI
type RealtimeAPIStatus struct {
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
