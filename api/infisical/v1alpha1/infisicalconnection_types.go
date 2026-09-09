/*
Copyright 2026 winrarr.

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

// InfisicalConnectionSpec defines the desired state of InfisicalConnection
type InfisicalConnectionSpec struct {
	// HostAPI is the Infisical API base URL, including the /api path.
	// It defaults to Infisical Cloud.
	// +optional
	// +kubebuilder:default="https://app.infisical.com/api"
	// +kubebuilder:validation:Pattern=`^https?://`
	HostAPI string `json:"hostAPI,omitempty"`

	// AuthSecretRef references a Secret containing a bearer token under Key.
	// The Secret must be in the same namespace as this connection.
	AuthSecretRef SecretKeyReference `json:"authSecretRef"`

	// RequestTimeout bounds each request made to Infisical.
	// +optional
	// +kubebuilder:default="30s"
	RequestTimeout *metav1.Duration `json:"requestTimeout,omitempty"`
}

// InfisicalConnectionStatus defines the observed state of InfisicalConnection.
type InfisicalConnectionStatus struct {
	// ObservedGeneration is the most recent generation reflected in status.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the connection.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InfisicalConnection is the Schema for the infisicalconnections API
type InfisicalConnection struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of InfisicalConnection
	// +required
	Spec InfisicalConnectionSpec `json:"spec"`

	// status defines the observed state of InfisicalConnection
	// +optional
	Status InfisicalConnectionStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InfisicalConnectionList contains a list of InfisicalConnection
type InfisicalConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InfisicalConnection `json:"items"`
}
