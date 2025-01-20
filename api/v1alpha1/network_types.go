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

import "fmt"

// NetworkSpec encapsulates the configuration options for HuaweiCloud network.
type NetworkSpec struct {
	// VPC configuration.
	// +optional
	VPC VPCSpec `json:"vpc,omitempty"`

	// Subnets configuration.
	// +optional
	Subnets Subnets `json:"subnets,omitempty"`
}

type VPCSpec struct {
	// Id is the unique identifier of the VPC. It is a UUID.
	Id string `json:"id"`

	// Name is the name of the VPC. It must be 0-64 characters long and support numbers, letters, Chinese characters, _(underscore), -(hyphen), and .(dot).
	Name string `json:"name"`

	// Cidr is the CIDR of the VPC.
	Cidr string `json:"cidr"`
}

type Subnets []SubnetSpec

// Subnet
type SubnetSpec struct {
	// ID defines a unique identifier to reference this resource.
	Id string `json:"id"`

	// Name is the name of the subnet. It must be 1-64 characters long and support numbers, letters, Chinese characters, _(underscore), -(hyphen), and .(dot).
	Name string `json:"name"`

	// CIDR is the CIDR of the subnet. It must be in CIDR format. The mask length cannot be greater than 28.
	Cidr string `json:"cidr"`

	// GatewayIp is the gateway of the subnet. It must be an IP address in the subnet segment.
	GatewayIp string `json:"gateway_ip"`

	// VPCId is the identifier of the VPC where the subnet is located.
	VpcId string `json:"vpc_id"`

	// NeutronNetworkId is the identifier of the network (OpenStack Neutron interface).
	NeutronNetworkId string `json:"neutron_network_id"`

	// NeutronSubnetId is the identifier of the subnet (OpenStack Neutron interface).
	NeutronSubnetId string `json:"neutron_subnet_id"`
}

// SecurityGroupRole defines the unique role of a security group.
// +kubebuilder:validation:Enum=bastion;node;controlplane;apiserver-lb;lb;node-eks-additional
type SecurityGroupRole string

var (
	// SecurityGroupNode defines a Kubernetes workload node role.
	SecurityGroupNode = SecurityGroupRole("node")

	// SecurityGroupControlPlane defines a Kubernetes control plane node role.
	SecurityGroupControlPlane = SecurityGroupRole("controlplane")

	// SecurityGroupAPIServerLB defines a Kubernetes API Server Load Balancer role.
	SecurityGroupAPIServerLB = SecurityGroupRole("apiserver-lb")

	// SecurityGroupLB defines a container for the cloud provider to inject its load balancer ingress rules.
	SecurityGroupLB = SecurityGroupRole("lb")
)

// SecurityGroupRule
type SecurityGroupRule struct {
	// ID is the unique identifier of the security group rule.
	Id string `json:"id"`

	// Description is the description of the security group rule.
	Description string `json:"description"`

	// SecurityGroupId is the security group id.
	SecurityGroupId string `json:"security_group_id"`

	// Direction is the direction of the security group rule. Accepted values are "ingress" and "egress".
	Direction string `json:"direction"`

	// Ethertype is the IP protocol type. The value can be IPv4 or IPv6.
	Ethertype string `json:"ethertype"`

	// Protocol is the protocol for the security group rule.
	Protocol string `json:"protocol"`

	// PortRangeMin is the start of port range.
	PortRangeMin int32 `json:"port_range_min"`

	// PortRangeMax is the end of port range.
	PortRangeMax int32 `json:"port_range_max"`

	// RemoteIpPrefix is the CIDR block to allow access from.
	RemoteIpPrefix string `json:"remote_ip_prefix"`

	// RemoteGroupId is the remote security group id.
	RemoteGroupId string `json:"remote_group_id"`

	// RemoteAddressGroupId is the remote address group id.
	RemoteAddressGroupId string `json:"remote_address_group_id"`
}

type PoolRef struct {
	// Id is the unique identifier of the pool.
	Id string `json:"id"`
}

type ListenerRef struct {
	// Id is the unique identifier of the listener.
	Id string `json:"id"`
}

// LoadBalancer defines an AWS load balancer.
type LoadBalancer struct {
	// Id is the unique identifier of the loadbalancer.
	Id string `json:"id"`

	// Name is the name of the load balancer.
	Name string `json:"name"`

	// Pools is a list of pool references associated with the load balancer.
	Pools []PoolRef `json:"pools"`

	// Listeners is a list of listener references associated with the load balancer.
	Listeners []ListenerRef `json:"listeners"`
}

// SecurityGroup defines an HuaweiCloud security group.
type SecurityGroup struct {
	// ID is a unique identifier.
	ID string `json:"id"`

	// Name is the security group name.
	Name string `json:"name"`

	// IngressRules is the inbound rules associated with the security group.
	// +optional
	SecurityGroupRules []SecurityGroupRule `json:"ingressRule,omitempty"`
}

// String returns a string representation of the security group.
func (s *SecurityGroup) String() string {
	return fmt.Sprintf("id=%s/name=%s", s.ID, s.Name)
}

// NetworkStatus encapsulates HuaweiCloud networking resources.
type NetworkStatus struct {
	// SecurityGroups is a map from the role/kind of the security group to its unique name, if any.
	SecurityGroups map[SecurityGroupRole]SecurityGroup `json:"securityGroups,omitempty"`

	// ELB is the Elastic Load Balancer associated with the cluster.
	ELB LoadBalancer `json:"elb,omitempty"`

	// NatGatewaysIPs contains the public IPs of the NAT Gateways
	NatGatewaysIPs []string `json:"natGatewaysIPs,omitempty"`
}
