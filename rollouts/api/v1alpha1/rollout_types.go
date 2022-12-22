/*
Copyright 2022.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RolloutSpec defines the desired state of Rollout
type RolloutSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Description is a user friendly description of this Rollout.
	Description string `json:"description,omitempty"`

	// Packages source for this Rollout.
	Packages PackagesConfig `json:"packages"`

	// Targets specifies the clusters that will receive the KRM config packages.
	Targets ClusterTargetSelector `json:"targets,omitempty"`

	// PackageToTargetMatcher specifies the clusters that will receive a specific package.
	PackageToTargetMatcher PackageToClusterMatcher `json:"packageToTargetMatcher"`
}

type ClusterTargetSelector struct {
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// ClusterReference contains the identify information
// need to refer a cluster.
type ClusterRef struct {
	Name string `json:"name"`
}

// +kubebuilder:validation:Enum=git
type PackageSourceType string

// PackagesConfig defines the packages the Rollout should deploy.
type PackagesConfig struct {
	SourceType PackageSourceType `json:"sourceType"`

	Git GitSource `json:"git"`
}

// GitSource defines the packages source in Git.
type GitSource struct {
	GitRepoSelector GitSelector `json:"selector"`
}

// GitSelector defines the selector to apply to Git.
type GitSelector struct {
	Org       string          `json:"org"`
	Repo      string          `json:"repo"`
	Directory string          `json:"directory"`
	Revision  string          `json:"revision"`
	SecretRef SecretReference `json:"secretRef,omitempty"`
}

// SecretReference contains the reference to the secret
type SecretReference struct {
	// Name represents the secret name
	Name string `json:"name,omitempty"`
}

// +kubebuilder:validation:Enum=CEL
type MatcherType string

type PackageToClusterMatcher struct {
	Type            MatcherType `json:"type"`
	MatchExpression string      `json:"matchExpression"`
}

// RolloutStatus defines the observed state of Rollout
type RolloutStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Rollout is the Schema for the rollouts API
type Rollout struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RolloutSpec   `json:"spec,omitempty"`
	Status RolloutStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RolloutList contains a list of Rollout
type RolloutList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rollout `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Rollout{}, &RolloutList{})
}
