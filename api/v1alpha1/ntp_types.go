/*
Copyright 2020 Wim Henderickx.

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

// NtpServer defines the NTP server
type NtpServer struct {
	// +kubebuilder:validation:Required
	Address string `json:"address"`
	// +kubebuilder:validation:Default:false
	IBurst bool `json:"iburst,omitempty"`
	// +kubebuilder:validation:Default:false
	Prefer bool `json:"prefer,omitempty"`
}

// NtpSpec defines the desired state of Ntp
type NtpSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Enum=enable;disable
	AdminState string `json:"admin-state,omitempty"`
	// +kubebuilder:validation:Required
	NetworkInstance string      `json:"network-instance"`
	Server          []NtpServer `json:"server,omitempty"`
}

// NtpServerState defines the NTP server state
type NtpServerState struct {
	// +kubebuilder:validation:Required
	Address string `json:"address"`
	// +kubebuilder:validation:Default:false
	IBurst bool `json:"iBurst,omitempty"`
	// +kubebuilder:validation:Default:false
	Prefer       bool   `json:"prefer,omitempty"`
	Stratum      uint8  `json:"stratum,omitempty"`
	Jitter       string `json:"jitter,omitempty"`
	Offset       string `json:"offset,omitempty"`
	PollInterval uint16 `json:"pollInterval,omitempty"`
}

// NtpStatus defines the observed state of Ntp
type NtpStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Enum=enable;disable
	AdminState string `json:"adminState,omitempty"`
	// +kubebuilder:validation:Enum=up;down;empty;downloading;booting;starting;failed;synchronizing;upgrading
	OperState    string `json:"operState,omitempty"`
	Synchronized string `json:"synchronized,omitempty"`
	// +kubebuilder:validation:Required
	NetworkInstance string           `json:"networkInstance"`
	Server          []NtpServerState `json:"server,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Ntp is the Schema for the ntps API
type Ntp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NtpSpec   `json:"spec,omitempty"`
	Status NtpStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NtpList contains a list of Ntp
type NtpList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Ntp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Ntp{}, &NtpList{})
}
