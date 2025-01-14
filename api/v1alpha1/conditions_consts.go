/*
Copyright 2022 The Kubernetes Authors.

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

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

const (
	// InstanceReadyCondition reports on current status of the ECS instance. Ready indicates the instance is in a Running state.
	InstanceReadyCondition clusterv1.ConditionType = "InstanceReady"

	// InstanceNotFoundReason used when the instance couldn't be retrieved.
	InstanceNotFoundReason = "InstanceNotFound"
	// InstanceTerminatedReason instance is in a terminated state.
	InstanceTerminatedReason = "InstanceTerminated"
	// InstanceStoppedReason instance is in a stopped state.
	InstanceStoppedReason = "InstanceStopped"
	// InstanceNotReadyReason used when the instance is in a pending state.
	InstanceNotReadyReason = "InstanceNotReady"
	// InstanceProvisionStartedReason set when the provisioning of an instance started.
	InstanceProvisionStartedReason = "InstanceProvisionStarted"
	// InstanceProvisionFailedReason used for failures during instance provisioning.
	InstanceProvisionFailedReason = "InstanceProvisionFailed"
	// WaitingForClusterInfrastructureReason used when machine is waiting for cluster infrastructure to be ready before proceeding.
	WaitingForClusterInfrastructureReason = "WaitingForClusterInfrastructure"
	// WaitingForBootstrapDataReason used when machine is waiting for bootstrap data to be ready before proceeding.
	WaitingForBootstrapDataReason = "WaitingForBootstrapData"
)

const (
	// SecurityGroupsReadyCondition indicates the security groups are up to date on the HuaweiCloudMachine.
	SecurityGroupsReadyCondition clusterv1.ConditionType = "SecurityGroupsReady"

	// SecurityGroupsFailedReason used when the security groups could not be synced.
	SecurityGroupsFailedReason = "SecurityGroupsSyncFailed"
)

const (
	// VpcReadyCondition reports on the successful reconciliation of a VPC.
	VpcReadyCondition clusterv1.ConditionType = "VpcReady"
	// VpcCreationStartedReason used when attempting to create a VPC for a managed cluster.
	// Will not be applied to unmanaged clusters.
	VpcCreationStartedReason = "VpcCreationStarted"
	// VpcReconciliationFailedReason used when errors occur during VPC reconciliation.
	VpcReconciliationFailedReason = "VpcReconciliationFailed"
)

const (
	// SubnetsReadyCondition reports on the successful reconciliation of subnets.
	SubnetsReadyCondition clusterv1.ConditionType = "SubnetsReady"
	// SubnetsReconciliationFailedReason used to report failures while reconciling subnets.
	SubnetsReconciliationFailedReason = "SubnetsReconciliationFailed"
)

const (
	// ClusterSecurityGroupsReadyCondition reports successful reconciliation of security groups.
	ClusterSecurityGroupsReadyCondition clusterv1.ConditionType = "ClusterSecurityGroupsReady"
	// ClusterSecurityGroupReconciliationFailedReason used when any errors occur during reconciliation of security groups.
	ClusterSecurityGroupReconciliationFailedReason = "SecurityGroupReconciliationFailed"
)

const (
	// LoadBalancerReadyCondition reports on whether a control plane load balancer was successfully reconciled.
	LoadBalancerReadyCondition clusterv1.ConditionType = "LoadBalancerReady"
	// LoadBalancerFailedReason used when an error occurs during load balancer reconciliation.
	LoadBalancerFailedReason = "LoadBalancerFailed"
)

const (
	// NatGatewaysReadyCondition reports successful reconciliation of NAT gateways.
	// Only applicable to managed clusters.
	NatGatewaysReadyCondition clusterv1.ConditionType = "NatGatewaysReady"
	// NatGatewaysCreationStartedReason set once when creating new NAT gateways.
	NatGatewaysCreationStartedReason = "NatGatewaysCreationStarted"
	// NatGatewaysReconciliationFailedReason used when any errors occur during reconciliation of NAT gateways.
	NatGatewaysReconciliationFailedReason = "NatGatewaysReconciliationFailed"
)
