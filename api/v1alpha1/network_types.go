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
	"fmt"
	"sort"
)

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

// SubnetSpec configures an HuaweiCloud VPC Subnet.
type SubnetSpec struct {
	// Id defines a unique identifier to reference this resource.
	Id string `json:"id"`

	// Name is the name of the subnet. It must be 1-64 characters long and support numbers, letters, Chinese characters, _(underscore), -(hyphen), and .(dot).
	Name string `json:"name"`

	// ResourceID is the subnet identifier from HuaweiCloud, READ ONLY.
	// This field is populated when the provider manages the subnet.
	// +optional
	ResourceID string `json:"resourceID,omitempty"`

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

	// IPv6CidrBlock is the IPv6 CIDR block to be used when the provider creates a managed VPC.
	// A subnet can have an IPv4 and an IPv6 address.
	// IPv6 is only supported in managed clusters, this field cannot be set on HuaweiCloudCluster object.
	// +optional
	IPv6CidrBlock string `json:"ipv6CidrBlock,omitempty"`

	// AvailabilityZone defines the availability zone to use for this subnet in the cluster's region.
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// IsPublic defines the subnet as a public subnet. A subnet is public when it is associated with a route table that has a route to an internet gateway.
	// +optional
	IsPublic bool `json:"isPublic"`

	// IsIPv6 defines the subnet as an IPv6 subnet. A subnet is IPv6 when it is associated with a VPC that has IPv6 enabled.
	// IPv6 is only supported in managed clusters, this field cannot be set on HuaweiCloudCluster object.
	// +optional
	IsIPv6 bool `json:"isIpv6,omitempty"`
}

// GetResourceID returns the identifier for this subnet,
// if the subnet was not created or reconciled, it returns the subnet ID.
func (s *SubnetSpec) GetResourceID() string {
	if s.ResourceID != "" {
		return s.ResourceID
	}
	return s.Id
}

// Subnets is a slice of Subnet.
// +listType=map
// +listMapKey=id
type Subnets []SubnetSpec

// FindByID returns a single subnet matching the given id or nil.
//
// The returned pointer can be used to write back into the original slice.
func (s Subnets) FindByID(id string) *SubnetSpec {
	for i := range s {
		x := &(s[i]) // pointer to original structure
		if x.GetResourceID() == id {
			return x
		}
	}
	return nil
}

// FilterPrivate returns a slice containing all subnets marked as private.
func (s Subnets) FilterPrivate() (res Subnets) {
	for _, x := range s {
		if !x.IsPublic {
			res = append(res, x)
		}
	}
	return
}

// SecurityGroupProtocol defines the protocol type for a security group rule.
type SecurityGroupProtocol string

var (
	// SecurityGroupProtocolAll is a wildcard for all IP protocols.
	SecurityGroupProtocolAll = SecurityGroupProtocol("-1")

	// SecurityGroupProtocolIPinIP represents the IP in IP protocol in ingress rules.
	SecurityGroupProtocolIPinIP = SecurityGroupProtocol("4")

	// SecurityGroupProtocolTCP represents the TCP protocol in ingress rules.
	SecurityGroupProtocolTCP = SecurityGroupProtocol("tcp")

	// SecurityGroupProtocolUDP represents the UDP protocol in ingress rules.
	SecurityGroupProtocolUDP = SecurityGroupProtocol("udp")

	// SecurityGroupProtocolICMP represents the ICMP protocol in ingress rules.
	SecurityGroupProtocolICMP = SecurityGroupProtocol("icmp")

	// SecurityGroupProtocolICMPv6 represents the ICMPv6 protocol in ingress rules.
	SecurityGroupProtocolICMPv6 = SecurityGroupProtocol("58")

	// SecurityGroupProtocolESP represents the ESP protocol in ingress rules.
	SecurityGroupProtocolESP = SecurityGroupProtocol("50")
)

// IngressRule defines an HuaweiCloud ECS ingress rule for security groups.
type IngressRule struct {
	// Description provides extended information about the ingress rule.
	Description string `json:"description"`
	// Protocol is the protocol for the ingress rule. Accepted values are "-1" (all), "4" (IP in IP),"tcp", "udp", "icmp", and "58" (ICMPv6), "50" (ESP).
	// +kubebuilder:validation:Enum="-1";"4";tcp;udp;icmp;"58";"50"
	Protocol SecurityGroupProtocol `json:"protocol"`
	// PortRangeMin is the start of port range.
	PortRangeMin int64 `json:"portRangeMin"`
	// PortRangeMax is the end of port range.
	PortRangeMax int64 `json:"portRangeMax"`

	// List of CIDR blocks to allow access from. Cannot be specified with SourceSecurityGroupID.
	// +optional
	CidrBlocks []string `json:"cidrBlocks,omitempty"`

	// List of IPv6 CIDR blocks to allow access from. Cannot be specified with SourceSecurityGroupID.
	// +optional
	IPv6CidrBlocks []string `json:"ipv6CidrBlocks,omitempty"`

	// The security group id to allow access from. Cannot be specified with CidrBlocks.
	// +optional
	SourceSecurityGroupIDs []string `json:"sourceSecurityGroupIds,omitempty"`

	// The security group role to allow access from. Cannot be specified with CidrBlocks.
	// The field will be combined with source security group IDs if specified.
	// +optional
	SourceSecurityGroupRoles []SecurityGroupRole `json:"sourceSecurityGroupRoles,omitempty"`

	// NatGatewaysIPsSource use the NAT gateways IPs as the source for the ingress rule.
	// +optional
	NatGatewaysIPsSource bool `json:"natGatewaysIPsSource,omitempty"`
}

// IngressRules is a slice of HuaweiCloud ECS ingress rules for security groups.
type IngressRules []IngressRule

// Difference returns the difference between this slice and the other slice.
func (i IngressRules) Difference(o IngressRules) (out IngressRules) {
	for index := range i {
		x := i[index]
		found := false
		for oIndex := range o {
			y := o[oIndex]
			if x.Equals(&y) {
				found = true
				break
			}
		}

		if !found {
			out = append(out, x)
		}
	}

	return
}

// Equals returns true if two IngressRule are equal.
func (i *IngressRule) Equals(o *IngressRule) bool {
	// ipv4
	if len(i.CidrBlocks) != len(o.CidrBlocks) {
		return false
	}

	sort.Strings(i.CidrBlocks)
	sort.Strings(o.CidrBlocks)

	for i, v := range i.CidrBlocks {
		if v != o.CidrBlocks[i] {
			return false
		}
	}
	// ipv6
	if len(i.IPv6CidrBlocks) != len(o.IPv6CidrBlocks) {
		return false
	}

	sort.Strings(i.IPv6CidrBlocks)
	sort.Strings(o.IPv6CidrBlocks)

	for i, v := range i.IPv6CidrBlocks {
		if v != o.IPv6CidrBlocks[i] {
			return false
		}
	}

	if len(i.SourceSecurityGroupIDs) != len(o.SourceSecurityGroupIDs) {
		return false
	}

	sort.Strings(i.SourceSecurityGroupIDs)
	sort.Strings(o.SourceSecurityGroupIDs)

	for i, v := range i.SourceSecurityGroupIDs {
		if v != o.SourceSecurityGroupIDs[i] {
			return false
		}
	}

	if i.Description != o.Description || i.Protocol != o.Protocol {
		return false
	}

	// HuaweiCloud ECS seems to ignore the From/To port when set on protocols where it doesn't apply, but
	// we avoid serializing it out for clarity's sake.
	switch i.Protocol {
	case SecurityGroupProtocolTCP,
		SecurityGroupProtocolUDP,
		SecurityGroupProtocolICMP,
		SecurityGroupProtocolICMPv6:
		return i.PortRangeMin == o.PortRangeMin && i.PortRangeMax == o.PortRangeMax
	case SecurityGroupProtocolAll, SecurityGroupProtocolIPinIP, SecurityGroupProtocolESP:
		// FromPort / ToPort are not applicable
	}

	return true
}

// ElasticIPPool allows configuring a Elastic IP pool for resources allocating
// public IPv4 addresses on public subnets.
type ElasticIPPool struct {
	// PublicIpv4Pool is ID of the Public IPv4 Pool. It sets a custom Public IPv4 Pool used to create
	// Elastic IP address for resources created in public IPv4 subnets. Every IPv4 address, Elastic IP,
	// will be allocated from the custom Public IPv4 pool that you brought to ECS, instead of
	// Amazon-provided pool.
	//
	// +kubebuilder:validation:MaxLength=30
	// +optional
	PublicIpv4Pool *string `json:"publicIpv4Pool,omitempty"`
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

// LoadBalancer defines an HuaweiCloud load balancer.
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
