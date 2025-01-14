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
	infrav1alpha1 "github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/api/v1alpha1"
	"github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/pkg/basic"
)

// ECSScope is the interface for the scope to be used with the ecs service.
type ECSScope interface {
	basic.ClusterScoper

	// VPC returns the cluster VPC.
	VPC() *infrav1alpha1.VPCSpec

	// Subnets returns the cluster subnets.
	Subnets() infrav1alpha1.Subnets

	// Network returns the cluster network object.
	Network() *infrav1alpha1.NetworkStatus

	// SecurityGroups returns the cluster security groups as a map, it creates the map if empty.
	SecurityGroups() map[infrav1alpha1.SecurityGroupRole]infrav1alpha1.SecurityGroup

	// SSHKeyName returns the SSH key name to use for instances.
	SSHKeyName() *string

	// ImageLookupFormat returns the format string to use when looking up AMIs
	ImageLookupFormat() string

	// ImageLookupOrg returns the organization name to use when looking up AMIs
	ImageLookupOrg() string

	// ImageLookupBaseOS returns the base operating system name to use when looking up AMIs
	ImageLookupBaseOS() string
}
