package network

import (
	"strings"

	infrav1alpha1 "github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/api/v1alpha1"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/model"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

func (s *Service) reconcileSubnets() error {
	if s.scope.VPC().Id == "" {
		return errors.New("VPC ID is empty")
	}

	// Check if subnet exists, if not create it
	request := &model.ListSubnetsRequest{
		VpcId: &s.scope.VPC().Id,
	}
	response, err := s.vpcClient.ListSubnets(request)
	if err != nil {
		return errors.Wrap(err, "failed to list subnets")
	}

	var subnet *model.Subnet
	if len(*response.Subnets) == 0 {
		createRequest := &model.CreateSubnetRequest{}
		subnetbody := &model.CreateSubnetOption{
			Name:      "subnet-caph",
			Cidr:      "192.168.1.0/24",
			VpcId:     s.scope.VPC().Id,
			GatewayIp: "192.168.1.1",
		}
		createRequest.Body = &model.CreateSubnetRequestBody{
			Subnet: subnetbody,
		}
		response, err := s.vpcClient.CreateSubnet(createRequest)
		if err != nil {
			return errors.Wrap(err, "failed to create subnet")
		}

		subnet = response.Subnet
		klog.Infof("Subnet created, response: %v", response)
	} else {
		subnet = &(*response.Subnets)[0]
		klog.Infof("Subnet already exists")
	}

	s.scope.SetSubnets([]infrav1alpha1.SubnetSpec{
		{
			Id:               subnet.Id,
			Name:             subnet.Name,
			Cidr:             subnet.Cidr,
			GatewayIp:        subnet.GatewayIp,
			VpcId:            subnet.VpcId,
			NeutronNetworkId: subnet.NeutronNetworkId,
			NeutronSubnetId:  subnet.NeutronSubnetId,
		},
	})

	// Persist the new default subnets to HCCluster
	if err := s.scope.PatchObject(); err != nil {
		klog.Errorf("Failed to patch HCCluster: %v", err)
		return err
	}

	return nil
}

func (s *Service) deleteSubnets() error {
	if s.scope.VPC().Id == "" {
		klog.Infof("VPC ID is empty")
		return nil
	}
	request := &model.ListSubnetsRequest{
		VpcId: &s.scope.VPC().Id,
	}
	response, err := s.vpcClient.ListSubnets(request)
	if err != nil {
		if strings.Contains(err.Error(), "VPC.0202") {
			klog.Infof("VPC not found")
			return nil
		}
		return errors.Wrap(err, "failed to list subnets")
	}

	for _, subnet := range *response.Subnets {
		deleteRequest := &model.DeleteSubnetRequest{
			VpcId:    subnet.VpcId,
			SubnetId: subnet.Id,
		}
		response, err := s.vpcClient.DeleteSubnet(deleteRequest)
		if err != nil {
			return errors.Wrapf(err, "failed to delete subnet %s", subnet.Id)
		}
		klog.Infof("subnet delete response: %v", response)
		klog.Infof("Deleted subnet %s", subnet.Id)
	}

	return nil
}

func (s *Service) FindSubnet(subnetId string) (*model.Subnet, error) {
	request := &model.ShowSubnetRequest{
		SubnetId: subnetId,
	}
	response, err := s.vpcClient.ShowSubnet(request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to show subnet %s", subnetId)
	}
	return response.Subnet, nil
}
