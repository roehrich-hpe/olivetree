/*
Copyright 2023.

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

const (
	// A label for pods running DAOS.
	DAOSLabel = "dmg.hpe.com/daos"

	// A label value for pods running DAOS server.
	DAOSServerLabel = "server"

	// The namespace for our DAOS controllers.
	DAOSPodNamespace = "olivetree-system"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DmgSpec defines the desired state of Dmg
type DmgSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Dmg. Edit dmg_types.go to remove/update
	Foo string `json:"foo,omitempty"`

	// Cmd is the "dmg" command to execute
	Cmd string `json:"cmd,omitempty"`

	// Set to true if the command operation should be canceled.
	// +kubebuilder:default:=false
	Cancel bool `json:"cancel,omitempty"`
}

// Types describing the various command status conditions.
// +kubebuilder:validation:Enum:=Running;Finished
type DmgConditionType string

const (
	DmgConditionTypeRunning  DmgConditionType = "Running"
	DmgConditionTypeFinished DmgConditionType = "Finished"
	// NOTE: You must ensure any new value is added to the above kubebuilder validation enum.
)

// DmgStatus defines the observed state of Dmg
type DmgStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Output contains any output from the command
	Output string `json:"output"`

	// ExitStatus contains the dmg command's process exit status
	ExitStatus string `json:"exitStatus"`

	// Status is the state of the dmg command
	Status DmgConditionType `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Dmg is the Schema for the dmgs API
type Dmg struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DmgSpec   `json:"spec,omitempty"`
	Status DmgStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DmgList contains a list of Dmg
type DmgList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dmg `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Dmg{}, &DmgList{})
}
