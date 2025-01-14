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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
)

const (
	// MachineFinalizer allows HuaweiCloudMachineReconciler to clean up HuaweiCloud resources associated with HuaweiCloudMachine before
	// removing it from the apiserver.
	MachineFinalizer = "huaweicloudmachine.infrastructure.cluster.x-k8s.io"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HuaweiCloudMachineSpec defines the desired state of HuaweiCloudMachine.
type HuaweiCloudMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ProviderID is the unique identifier as specified by the cloud provider.
	ProviderID *string `json:"providerID,omitempty"`

	// InstanceID is the ECS instance ID for this machine.
	InstanceID *string `json:"instanceID,omitempty"`

	// 镜像ID或者镜像资源的URL
	// ImageRef is the reference from which to create the machine instance.
	ImageRef *string `json:"imageRef,omitempty"`

	// FlavorRef is similar to instanceType.
	// FlavorRef is the type of instance to create. Example: s2.small.1
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength:=2
	FlavorRef string `json:"flavorRef"`

	// SSHKeyName is the name of the ssh key to attach to the instance. Valid values are empty string (do not use SSH keys), a valid SSH key name, or omitted (use the default SSH key name)
	// +optional
	SSHKeyName *string `json:"sshKeyName,omitempty"`

	// RootVolume encapsulates the configuration options for the root volume
	// +optional
	RootVolume *Volume `json:"rootVolume,omitempty"`

	// TODO
	// Configuration options for the data storage volumes.
	// +optional
	// DataVolumes *[]model.DataVolumes `json:"dataVolumes,omitempty"`

	// PublicIP specifies whether the instance should get a public IP.
	// Precedence for this setting is as follows:
	// 1. This field if set
	// 2. Cluster/flavor setting
	// 3. Subnet default
	// +optional
	PublicIP *bool `json:"publicIP,omitempty"`

	// ElasticIPPool is the configuration to allocate Public IPv4 address (Elastic IP/EIP) from user-defined pool.
	//
	// +optional
	ElasticIPPool *ElasticIPPool `json:"elasticIpPool,omitempty"`

	// TODO, more fields need to be defined in the future
	// AdminPass *string `json:"admin_pass,omitempty"`
	// NetConfig *NetConfig `json:"net_config"`
	// Bandwidth *BandwidthConfig `json:"bandwidth,omitempty"`

	// Subnet is a reference to the subnet to use for this instance. If not specified,
	// the cluster subnet will be used.
	// +optional
	Subnet *HuaweiCloudResourceReference `json:"subnet,omitempty"`
}

// HuaweiCloudMachineStatus defines the observed state of HuaweiCloudMachine.
type HuaweiCloudMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Ready is true when the provider resource is ready.
	// +optional
	Ready bool `json:"ready"`

	// Addresses contains the ECS instance associated addresses.
	Addresses []clusterv1.MachineAddress `json:"addresses,omitempty"`

	// InstanceState is the state of the ECS instance for this machine.
	// +optional
	InstanceState *InstanceState `json:"instanceState,omitempty"`

	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// +optional
	FailureReason *capierrors.MachineStatusError `json:"failureReason,omitempty"`

	// Conditions defines current service state of the HuaweiCloudMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// HuaweiCloudMachine is the Schema for the huaweicloudmachines API.
type HuaweiCloudMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HuaweiCloudMachineSpec   `json:"spec,omitempty"`
	Status HuaweiCloudMachineStatus `json:"status,omitempty"`
}

// GetConditions returns the observations of the operational state of the HuaweiCloudMachine resource.
func (r *HuaweiCloudMachine) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

// SetConditions sets the underlying service state of the HuaweiMachine to the predescribed clusterv1.Conditions.
func (r *HuaweiCloudMachine) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

// +kubebuilder:object:root=true

// HuaweiCloudMachineList contains a list of HuaweiCloudMachine.
type HuaweiCloudMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HuaweiCloudMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HuaweiCloudMachine{}, &HuaweiCloudMachineList{})
}
