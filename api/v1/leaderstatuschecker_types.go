/*
Copyright 2025.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LeaderStatusCheckerSpec defines the desired state of LeaderStatusChecker.
type LeaderStatusCheckerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// StatefulSetName is the name of the StatefulSet to monitor pods from.
	StatefulSetName string `json:"statefulSetName"`

	// Namespace is the namespace where the StatefulSet is deployed.
	Namespace string `json:"namespace"`

	// RPCPort is the port number for gRPC service to check leader status.
	RPCPort int32 `json:"rpcPort"`

	// LocalDev is a flag to indicate if the operator is running locally.
	LocalDev bool `json:"localDev"`
}

// LeaderStatusCheckerStatus defines the observed state of LeaderStatusChecker.
type LeaderStatusCheckerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// LastUpdated is the time the labels were last updated
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// LeaderNode is the name of the current leader node
	LeaderNode string `json:"leaderNode,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// LeaderStatusChecker is the Schema for the leaderstatuscheckers API.
// +kubebuilder:printcolumn:JSONPath=".spec.statefulSetName",name=StatefulSetName,type=string
// +kubebuilder:printcolumn:JSONPath=".status.leaderNode",name=LeaderNode,type=string
// +kubebuilder:printcolumn:JSONPath=".status.lastUpdated",name=LastUpdated,type=date
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date
type LeaderStatusChecker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LeaderStatusCheckerSpec   `json:"spec,omitempty"`
	Status LeaderStatusCheckerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LeaderStatusCheckerList contains a list of LeaderStatusChecker.
type LeaderStatusCheckerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LeaderStatusChecker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LeaderStatusChecker{}, &LeaderStatusCheckerList{})
}
