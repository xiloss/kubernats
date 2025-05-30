/*
Copyright 2024 xiloss.

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

type JetStreamConfig struct {
	Subjects  []string `json:"subjects,omitempty"`
	Replicas  int      `json:"replicas,omitempty"`
	Retention string   `json:"retention,omitempty"`
}

// JetStreamSpec defines the desired state of JetStream
type JetStreamSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Credentials string          `json:"credentials,omitempty"`
	Account     string          `json:"account,omitempty"`
	Domain      string          `json:"domain,omitempty"`
	Config      JetStreamConfig `json:"config,omitempty"`
}

// JetStreamStatus defines the observed state of JetStream
type JetStreamStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ConfigApplied bool        `json:"config-applied,omitempty"`
	ErrorMessage  string      `json:"error-message,omitempty"`
	LastUpdated   metav1.Time `json:"last-updated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// JetStream is the Schema for the jetstreams API
type JetStream struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JetStreamSpec   `json:"spec,omitempty"`
	Status JetStreamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JetStreamList contains a list of JetStream
type JetStreamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JetStream `json:"items"`
}

// init initializes the schema
func init() {
	SchemeBuilder.Register(&JetStream{}, &JetStreamList{})
}
