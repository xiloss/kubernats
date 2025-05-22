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

// KeyValueStoreSpec defines the desired state of KeyValueStore
type KeyValueStoreSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Endpoint string         `json:"endpoint"`
	Buckets  []BucketConfig `json:"buckets"`
}

// BucketConfig defines the configuration of a Bucket used by a KeyValueStore
type BucketConfig struct {
	Name         string          `json:"name"`
	Replicas     int             `json:"replicas"`
	MaxValueSize int             `json:"maxValueSize"`
	History      int             `json:"history"`
	TTL          metav1.Duration `json:"ttl"`
}

// KeyValueStoreStatus defines the observed state of KeyValueStore
type KeyValueStoreStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Applied        bool           `json:"applied"`
	ErrorMessage   string         `json:"error-message,omitempty"`
	LastUpdated    metav1.Time    `json:"last-updated,omitempty"`
	BucketStatuses []BucketStatus `json:"bucket-status-list,omitempty"`
}

// BucketStatus defines the status of a Bucket used by a KeyValueStore
type BucketStatus struct {
	Name         string `json:"name"`
	Created      bool   `json:"created"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KeyValueStore is the Schema for the keyvaluestores API
type KeyValueStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeyValueStoreSpec   `json:"spec,omitempty"`
	Status KeyValueStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeyValueStoreList contains a list of KeyValueStore
type KeyValueStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeyValueStore `json:"items"`
}

// init initializes the schema
func init() {
	SchemeBuilder.Register(&KeyValueStore{}, &KeyValueStoreList{})
}
