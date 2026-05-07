/*
Copyright 2026.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HybridWorkloadSpec defines the desired state of HybridWorkload
type HybridWorkloadSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of HybridWorkload. Edit hybridworkload_types.go to remove/update
	// +optional
	//Foo *string `json:"foo,omitempty"`
	//

	// Priority determines scheduleing preference (high = always AWS)
	// +kubebuilder:validation:Enum=low;medium;high
	// +kubebuilder:default=medium
	// +kubebuilder:validation:Required
	Priority string `json:"priority"`

	// MaxMonthlyCostCents is budget constraint for this workload
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Required
	MaxMonthlyCostCents int64 `json:"maxMonthlyCostCents"`

	// Resource specifies CPU and memory requirements
	// +kubebuilder:validation:Required
	Resources corev1.ResourceRequirements `json:"resources"`

	// WorkloadTemplate is the Pod spec for Deployment/StatefulSet
	// +kubebuilder:validation:Required
	WorkloadTemplate corev1.PodTemplateSpec `json:"workloadTemplate"`

	// CapacityType for AWS nodes: "spot" or "on-demand"
	// +kubebuilder:validation:Enum=spot;on-demand
	// +kubebuilder:validation:Required
	// +kubebuilder:default=spot
	CapacityType string `json:"capacityType"`
}

// HybridWorkloadStatus defines the observed state of HybridWorkload.
type HybridWorkloadStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the HybridWorkload resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//

	// Phase: Pending, Running, Failed
	// +kubebuilder:validation:Enum=Pending;Running;Failed
	Phase string `json:"phase,omitempty"`

	// RecommendedPlatform: "proxmox" or "aws" or "pending"
	// +kubebuilder:validation:Enum=proxmox;aws;pending
	RecommendedPlatform string `json:"recommendedPlatform,omitempty"`

	// EstimatedMonthlyCostCents for current placement
	// +kubebuilder:validation:Minimum=0
	EstimatedMonthlyCostCents int64 `json:"estimatedMonthlyCostCents,omitempty"`

	// KarpenterNodePoolName if AWS burst was created
	KarpenterNodePoolName string `json:"karpenterNodePoolName,omitempty"`

	// VPNHealth - VPN health state (Healthy, Unhealthy, Unknown)
	// +kubebuilder:validation:Enum=Healthy;Unhealthy;Unknown
	VPNHealth string `json:"vpnHealth"`

	// DryRun indicates decision was computed but no resources were created
	DryRun bool `json:"dryRun,omitempty"`

	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	// Conditions for status tracking (SchedulingDecision, NodePoolReady, VPNHealthy)
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastReconcileTime
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Priority",type=string,JSONPath=`.spec.priority`
// +kubebuilder:printcolumn:name="Platform",type=string,JSONPath=`.status.recommendedPlatform`
// +kubebuilder:printcolumn:name="Cost",type=string,JSONPath=`.status.estimatedMonthlyCostCents`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// HybridWorkload is the Schema for the hybridworkloads API
type HybridWorkload struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of HybridWorkload
	// +required
	Spec HybridWorkloadSpec `json:"spec"`

	// status defines the observed state of HybridWorkload
	// +optional
	Status HybridWorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HybridWorkloadList contains a list of HybridWorkload
type HybridWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HybridWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HybridWorkload{}, &HybridWorkloadList{})
}
