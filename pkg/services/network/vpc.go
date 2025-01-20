package network

import (
	"strings"

	infrav1alpha1 "github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/api/v1alpha1"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/model"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
)

func (s *Service) reconcileVPC() error {
	// check if VPC exists, if not create it
	if s.scope.VPC().Id != "" {
		klog.Infof("VPC %s already exists", s.scope.VPC().Id)
		return nil
	}

	createRequest := &model.CreateVpcRequest{}
	cidrVpc := "192.168.0.0/16"
	nameVpc := "vpc-caph"
	vpcbody := &model.CreateVpcOption{
		Cidr: &cidrVpc,
		Name: &nameVpc,
	}
	createRequest.Body = &model.CreateVpcRequestBody{
		Vpc: vpcbody,
	}

	createRes, err := s.vpcClient.CreateVpc(createRequest)
	if err != nil {
		return errors.Wrap(err, "failed to create VPC")
	}
	vpc := createRes.Vpc
	klog.Infof("VPC create response: %v", createRes)
	klog.Infof("Created VPC %s", vpc.Id)

	s.scope.VPC().Id = vpc.Id
	s.scope.VPC().Name = vpc.Name
	s.scope.VPC().Cidr = vpc.Cidr

	if !conditions.Has(s.scope.InfraCluster(), infrav1alpha1.VpcReadyCondition) {
		conditions.MarkFalse(
			s.scope.InfraCluster(),
			infrav1alpha1.VpcReadyCondition,
			infrav1alpha1.VpcCreationStartedReason,
			clusterv1.ConditionSeverityInfo,
			"")
		if err := s.scope.PatchObject(); err != nil {
			return errors.Wrap(err, "failed to patch conditions")
		}
	}

	return nil
}

func (s *Service) deleteVPC() error {
	if s.scope.VPC().Id == "" {
		klog.Warning("VPC ID is empty")
		return nil
	}

	deleteRequest := &model.DeleteVpcRequest{
		VpcId: s.scope.VPC().Id,
	}
	response, err := s.vpcClient.DeleteVpc(deleteRequest)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			klog.Info("VPC already deleted", "vpcID", s.scope.VPC().Id)
			return nil
		}
		return errors.Wrapf(err, "failed to delete VPC %s", s.scope.VPC().Id)
	}
	klog.Infof("VPC delete response: %v", response)
	klog.Infof("Deleted VPC %s", s.scope.VPC().Id)
	return nil
}
