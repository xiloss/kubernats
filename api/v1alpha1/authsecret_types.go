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

type AuthSecretCredentials struct {
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

// AuthSecretSpec defines the desired state of AuthSecret
type AuthSecretSpec struct {
	TLS         string                `json:"tls,omitempty"`
	Token       string                `json:"token,omitempty"`
	Credentials AuthSecretCredentials `json:"credentials,omitempty"`
}

// AuthSecretStatus defines the observed state of AuthSecret
type AuthSecretStatus struct {
	SecretCreated bool   `json:"secret-created,omitempty"`
	ErrorMessage  string `json:"error-message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AuthSecret is the Schema for the authsecrets API
type AuthSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthSecretSpec   `json:"spec,omitempty"`
	Status AuthSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AuthSecretList contains a list of AuthSecret
type AuthSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthSecret `json:"items"`
}

// init initializes the schema
func init() {
	SchemeBuilder.Register(&AuthSecret{}, &AuthSecretList{})
}
