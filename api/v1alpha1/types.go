/*
Copyright 2024.
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
	"k8s.io/apimachinery/pkg/util/sets"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// InstanceState describes the state of an ECS instance.
type InstanceState string

var (
	// InstanceStatePending is the string representing an instance in a pending state.
	InstanceStatePending = InstanceState("pending")

	// InstanceStateRunning is the string representing an instance in a running state.
	InstanceStateRunning = InstanceState("running")

	// InstanceStateShuttingDown is the string representing an instance shutting down.
	InstanceStateShuttingDown = InstanceState("shutting-down")

	// InstanceStateTerminated is the string representing an instance that has been terminated.
	InstanceStateTerminated = InstanceState("terminated")

	// InstanceStateStopping is the string representing an instance
	// that is in the process of being stopped and can be restarted.
	InstanceStateStopping = InstanceState("stopping")

	// InstanceStateStopped is the string representing an instance
	// that has been stopped and can be restarted.
	InstanceStateStopped = InstanceState("stopped")

	// InstanceRunningStates defines the set of states in which an ECS instance is
	// running or going to be running soon.
	InstanceRunningStates = sets.NewString(
		string(InstanceStatePending),
		string(InstanceStateRunning),
	)

	// InstanceOperationalStates defines the set of states in which an ECS instance is
	// or can return to running, and supports all ECS operations.
	InstanceOperationalStates = InstanceRunningStates.Union(
		sets.NewString(
			string(InstanceStateStopping),
			string(InstanceStateStopped),
		),
	)

	// InstanceKnownStates represents all known ECS instance states.
	InstanceKnownStates = InstanceOperationalStates.Union(
		sets.NewString(
			string(InstanceStateShuttingDown),
			string(InstanceStateTerminated),
		),
	)
)

// HuaweiCloudResourceReference is a reference to a specific HuaweiCloud resource by ID.
type HuaweiCloudResourceReference struct {
	// ID of resource
	// +required
	ID *string `json:"id,omitempty"`
}

// Volume encapsulates the configuration options for the storage device.
type Volume struct {
	// Device name
	// +optional
	DeviceName string `json:"deviceName,omitempty"`

	// Size specifies size (in Gi) of the storage device.
	// Must be greater than the image snapshot size or 8 (whichever is greater).
	// +kubebuilder:validation:Minimum=8
	Size int64 `json:"size"`

	// Type is the type of the volume (e.g. gp2, io1, etc...).
	// +optional
	Type VolumeType `json:"type,omitempty"`

	// IOPS is the number of IOPS requested for the disk. Not applicable to all types.
	// +optional
	IOPS int64 `json:"iops,omitempty"`

	// Throughput to provision in MiB/s supported for the volume type. Not applicable to all types.
	// +optional
	Throughput *int64 `json:"throughput,omitempty"`
}

type VolumeType string

// VolumeTypeGPSSD is the string representing a general purpose ssd volume.
var VolumeTypeGPSSD = VolumeType("gpssd")

// Instance describes an HuaweiCloud ECS instance.
type Instance struct {
	ID string `json:"id"`

	// The current state of the instance.
	State InstanceState `json:"instanceState,omitempty"`

	// The instance type.
	Type string `json:"type,omitempty"`

	// The ID of the subnet of the instance.
	SubnetID string `json:"subnetId,omitempty"`

	// The ID of the AMI used to launch the instance.
	ImageID string `json:"imageId,omitempty"`

	// The name of the SSH key pair.
	SSHKeyName *string `json:"sshKeyName,omitempty"`

	// SecurityGroupIDs are one or more security group IDs this instance belongs to.
	SecurityGroupIDs []string `json:"securityGroupIds,omitempty"`

	// UserData is the raw data script passed to the instance which is run upon bootstrap.
	// This field must not be base64 encoded and should only be used when running a new instance.
	UserData *string `json:"userData,omitempty"`

	// The private IPv4 address assigned to the instance.
	PrivateIP *string `json:"privateIp,omitempty"`

	// The public IPv4 address assigned to the instance, if applicable.
	PublicIP *string `json:"publicIp,omitempty"`

	// Configuration options for the root storage volume.
	// +optional
	RootVolume *Volume `json:"rootVolume,omitempty"`

	// Configuration options for the data storage volumes.
	// +optional
	DataVolumes []Volume `json:"dataVolumes,omitempty"`

	// Addresses contains the ECS instance associated addresses.
	Addresses []clusterv1.MachineAddress `json:"addresses,omitempty"`

	// Availability zone of instance
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// PublicIPOnLaunch is the option to associate a public IP on instance launch
	// +optional
	PublicIPOnLaunch *bool `json:"publicIPOnLaunch,omitempty"`
}
