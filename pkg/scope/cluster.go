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

package scope

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1alpha1 "github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/api/v1alpha1"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	Client      client.Client
	Logger      *logr.Logger
	Cluster     *clusterv1.Cluster
	HCCluster   *infrav1alpha1.HuaweiCloudCluster
	Credentials *basic.Credentials
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	client      client.Client
	patchHelper *patch.Helper
	Logger      *logr.Logger
	Cluster     *clusterv1.Cluster
	HCCluster   *infrav1alpha1.HuaweiCloudCluster
	Credentials *basic.Credentials
}

// NewClusterScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	if params.Cluster == nil {
		return nil, errors.New("failed to generate new scope from nil Cluster")
	}
	if params.HCCluster == nil {
		return nil, errors.New("failed to generate new scope from nil HCCluster")
	}

	if params.Logger == nil {
		return nil, errors.New("failed to generate new scope from nil Logger")
	}

	clusterScope := &ClusterScope{
		Logger:      params.Logger,
		client:      params.Client,
		Cluster:     params.Cluster,
		HCCluster:   params.HCCluster,
		Credentials: params.Credentials,
	}

	helper, err := patch.NewHelper(params.HCCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	clusterScope.patchHelper = helper

	return clusterScope, nil
}

func (s *ClusterScope) Close() error {
	// Always attempt to patch the HuaweiCloudCluster object to update conditions, status, etc.
	return s.PatchObject()
}

// Name returns the CAPI Cluster name.
func (s *ClusterScope) ClusterName() string {
	return s.Cluster.Name
}

// CoreCluster returns the core cluster object.
func (s *ClusterScope) CoreCluster() conditions.Setter {
	return s.Cluster
}

// InfraCluster returns the huaweicloud cluster object.
func (s *ClusterScope) InfraCluster() conditions.Setter {
	return s.HCCluster
}

// VPC returns the cluster VPC.
func (s *ClusterScope) VPC() *infrav1alpha1.VPCSpec {
	return &s.HCCluster.Spec.NetworkSpec.VPC
}

// Subnets returns the cluster subnets.
func (s *ClusterScope) Subnets() infrav1alpha1.Subnets {
	return s.HCCluster.Spec.NetworkSpec.Subnets
}

// SetSubnets updates the clusters subnets.
func (s *ClusterScope) SetSubnets(subnets infrav1alpha1.Subnets) {
	s.HCCluster.Spec.NetworkSpec.Subnets = subnets
}

// SetNatGatewaysIPs sets the Nat Gateways Public IPs.
func (s *ClusterScope) SetNatGatewaysIPs(ips []string) {
	s.HCCluster.Status.Network.NatGatewaysIPs = ips
}

// Region returns the cluster region.
func (s *ClusterScope) Region() string {
	return s.HCCluster.Spec.Region
}

// SecurityGroups returns the cluster security groups as a map, it creates the map if empty.
func (s *ClusterScope) SecurityGroups() map[infrav1alpha1.SecurityGroupRole]infrav1alpha1.SecurityGroup {
	return s.HCCluster.Status.Network.SecurityGroups
}

// SetSecurityGroups updates the cluster security groups.
func (s *ClusterScope) SetSecurityGroups(sg map[infrav1alpha1.SecurityGroupRole]infrav1alpha1.SecurityGroup) {
	s.HCCluster.Status.Network.SecurityGroups = sg
}

// ELB returns the cluster ELB.
func (s *ClusterScope) ELB() infrav1alpha1.LoadBalancer {
	return s.HCCluster.Status.Network.ELB
}

// SetELB updates the cluster ELB.
func (s *ClusterScope) SetELB(elb infrav1alpha1.LoadBalancer) {
	s.HCCluster.Status.Network.ELB = elb
}

// PatchObject persists the cluster configuration and status.
func (s *ClusterScope) PatchObject() error {
	applicableConditions := []clusterv1.ConditionType{
		infrav1alpha1.VpcReadyCondition,
		infrav1alpha1.SubnetsReadyCondition,
		infrav1alpha1.ClusterSecurityGroupsReadyCondition,
		infrav1alpha1.NatGatewaysReadyCondition,
	}

	conditions.SetSummary(s.HCCluster,
		conditions.WithConditions(applicableConditions...),
		conditions.WithStepCounterIf(s.HCCluster.ObjectMeta.DeletionTimestamp.IsZero()),
		conditions.WithStepCounter(),
	)

	return s.patchHelper.Patch(
		context.TODO(),
		s.HCCluster,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1alpha1.VpcReadyCondition,
			infrav1alpha1.SubnetsReadyCondition,
			infrav1alpha1.ClusterSecurityGroupsReadyCondition,
			infrav1alpha1.NatGatewaysReadyCondition,
		}})
}

func (c *ClusterScope) Network() *infrav1alpha1.NetworkStatus {
	return &c.HCCluster.Status.Network
}

func (c *ClusterScope) SSHKeyName() *string {
	panic("TODO: Implement")
}

func (c *ClusterScope) ImageLookupFormat() string {
	panic("TODO: Implement")
}

func (c *ClusterScope) ImageLookupOrg() string {
	panic("TODO: Implement")
}

func (c *ClusterScope) ImageLookupBaseOS() string {
	panic("TODO: Implement")
}

func (c *ClusterScope) Credential() auth.ICredential {
	return c.Credentials
}
