package basic

import (
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth"
	"sigs.k8s.io/cluster-api/util/conditions"

	infrav1 "github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/api/v1alpha1"
)

type ClusterScoper interface {
	CoreCluster() conditions.Setter

	InfraCluster() conditions.Setter

	VPC() *infrav1.VPCSpec

	Region() string

	PatchObject() error

	Subnets() infrav1.Subnets

	Network() *infrav1.NetworkStatus

	SecurityGroups() map[infrav1.SecurityGroupRole]infrav1.SecurityGroup

	SSHKeyName() *string

	ImageLookupFormat() string

	ImageLookupOrg() string

	ImageLookupBaseOS() string

	Credential() auth.ICredential

	Close() error

	ELB() infrav1.LoadBalancer
}
