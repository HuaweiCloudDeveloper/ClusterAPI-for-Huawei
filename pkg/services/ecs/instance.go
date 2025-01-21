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

package ecs

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"time"

	ecsModel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	vpcModel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/model"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"

	infrav1 "github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/api/v1alpha1"
	"github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/pkg/ecserrors"
	"github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/pkg/scope"
)

func (s *Service) findSubnet(scope *scope.MachineScope) (string, error) {
	// Check Machine.Spec.FailureDomain first
	// as it's used by KubeadmControlPlane to spread machines across failure domains.
	failureDomain := scope.Machine.Spec.FailureDomain

	// We basically have 2 sources for subnets:
	//   1. If subnet.id or subnet.filters are specified, we directly query ECS
	//   2. All other cases use the subnets provided in the cluster network spec without ever calling ECS

	switch {
	case scope.HCMachine.Spec.Subnet != nil && scope.HCMachine.Spec.Subnet.ID != nil:
		var filtered []*vpcModel.Subnet

		subnet, err := s.netService.FindSubnet(*scope.HCMachine.Spec.Subnet.ID)
		if err != nil {
			return "", errors.Wrapf(err, "failed to find subnet %s", *scope.HCMachine.Spec.Subnet.ID)
		}

		var errMessage string
		if failureDomain != nil && subnet.AvailabilityZone != *failureDomain {
			errMessage += fmt.Sprintf(" subnet %q availability zone %q does not match failure domain %q.",
				subnet.Id, subnet.AvailabilityZone, *failureDomain)
		}

		if ptr.Deref(scope.HCMachine.Spec.PublicIP, false) {
			matchingSubnet := s.scope.Subnets().FindByID(subnet.Id)
			if matchingSubnet == nil {
				errMessage += fmt.Sprintf(" unable to find subnet %q among the HuaweiCloudCluster subnets.", subnet.Id)
			}
			if !ptr.Deref(scope.HCMachine.Spec.PublicIP, false) {
				errMessage += fmt.Sprintf(" subnet %q is a private subnet.", subnet.Id)
			}
		}
		filtered = append(filtered, subnet)

		clusterVPC := s.scope.VPC().Id
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].VpcId == clusterVPC
		})

		if len(filtered) == 0 {
			errMessage = fmt.Sprintf("failed to run machine %q, subnet %q not found",
				scope.Name(), *scope.HCMachine.Spec.Subnet.ID) + errMessage
			return "", errors.New(errMessage)
		}
		return filtered[0].Id, nil

	default:
		sns := s.scope.Subnets().FilterPrivate()
		if len(sns) == 0 {
			errMessage := fmt.Sprintf("failed to run machine %q, no subnets available", scope.Name())
			return "", errors.New(errMessage)
		}
		return sns[0].GetResourceID(), nil
	}
}

func (s *Service) GetCoreSecurityGroups(scope *scope.MachineScope) ([]string, error) {
	// These are common across both controlplane and node machines
	sgRoles := []infrav1.SecurityGroupRole{
		infrav1.SecurityGroupNode,
	}
	switch scope.Role() {
	case "control-plane":
		sgRoles = append(sgRoles, infrav1.SecurityGroupControlPlane)
	default:
		return nil, errors.Errorf("Unknown node role %q", scope.Role())
	}
	ids := make([]string, 0, len(sgRoles))
	for _, sg := range sgRoles {
		if _, ok := s.scope.SecurityGroups()[sg]; ok {
			ids = append(ids, s.scope.SecurityGroups()[sg].ID)
			continue
		}
		return nil, errors.New(fmt.Sprintf("%s security group not available", sg))
	}
	return ids, nil
}

func generateInstanceName(prefix string) string {
	const (
		chars     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		suffixLen = 8
	)

	// Generate 8 random characters
	suffix := make([]byte, suffixLen)
	uuid := uuid.NewUUID()
	for i := 0; i < suffixLen; i++ {
		// Use each byte of UUID to index into chars
		suffix[i] = chars[uuid[i]%byte(len(chars))]
	}

	return fmt.Sprintf("%s-%s", prefix, string(suffix))
}

func (s *Service) CreateInstance(scope *scope.MachineScope, userData []byte,
	userDataFormat string,
) (*infrav1.Instance, error) {
	scope.Logger.Info("Creating ECS instance")

	input := &infrav1.Instance{
		Type:       scope.HCMachine.Spec.FlavorRef,
		RootVolume: scope.HCMachine.Spec.RootVolume.DeepCopy(),
	}

	if input.RootVolume == nil {
		input.RootVolume = &infrav1.Volume{
			Size: 15,
		}
	}

	if scope.HCMachine.Spec.ImageRef != nil {
		input.ImageID = *scope.HCMachine.Spec.ImageRef
	}

	subnetID, err := s.findSubnet(scope)
	if err != nil {
		return nil, err
	}
	input.SubnetID = subnetID

	// Preserve user-defined PublicIp option.
	input.PublicIPOnLaunch = scope.HCMachine.Spec.PublicIP

	// Public address from Public IPv4 Pools need to be associated after launch (main machine
	// reconciliate loop) preventing duplicated public IP. The map on launch is explicitly
	// disabled in instances with PublicIP defined to true.
	if scope.HCMachine.Spec.ElasticIPPool != nil && scope.HCMachine.Spec.ElasticIPPool.PublicIpv4Pool != nil {
		input.PublicIPOnLaunch = ptr.To(false)
	}

	input.UserData = ptr.To[string](base64.StdEncoding.EncodeToString(userData))

	// Set security groups.
	ids, err := s.GetCoreSecurityGroups(scope)
	if err != nil {
		return nil, err
	}
	input.SecurityGroupIDs = append(input.SecurityGroupIDs, ids...)

	out, err := s.runInstance(input)
	if err != nil {
		return nil, err
	}

	// Set the providerID and instanceID as soon as we create an instance so that we keep it in case of errors afterward
	scope.SetProviderID(out.ID, out.AvailabilityZone)
	scope.SetInstanceID(out.ID)

	return out, nil
}

func (s *Service) runInstance(i *infrav1.Instance) (*infrav1.Instance, error) {
	createReq := &ecsModel.CreateServersRequest{
		XClientToken: ptr.To(string(uuid.NewUUID())),
		Body: &ecsModel.CreateServersRequestBody{
			Server: &ecsModel.PrePaidServer{
				AdminPass: ptr.To("dangerous@2025"),
				Name:      generateInstanceName("caphw-ecs"),
				ImageRef:  i.ImageID,
				FlavorRef: i.Type,
				Vpcid:     s.scope.VPC().Id,
				UserData:  i.UserData,
				Nics: []ecsModel.PrePaidServerNic{
					{
						SubnetId: i.SubnetID,
					},
				},
			},
		},
	}

	if i.PublicIPOnLaunch != nil {
		createReq.Body.Server.Publicip = &ecsModel.PrePaidServerPublicip{
			DeleteOnTermination: ptr.To(true),
			Eip: &ecsModel.PrePaidServerEip{
				Iptype: "5_bgp",
				Bandwidth: &ecsModel.PrePaidServerEipBandwidth{
					Size:       ptr.To(int32(5)),
					Sharetype:  ecsModel.GetPrePaidServerEipBandwidthSharetypeEnum().PER,
					Chargemode: ptr.To(""),
				},
			},
		}
	}

	securityGroups := []ecsModel.PrePaidServerSecurityGroup{}
	for _, e := range i.SecurityGroupIDs {
		securityGroups = append(securityGroups, ecsModel.PrePaidServerSecurityGroup{
			Id: &e,
		})
	}
	createReq.Body.Server.SecurityGroups = &securityGroups

	createReq.Body.Server.RootVolume = &ecsModel.PrePaidServerRootVolume{
		Volumetype: ecsModel.GetPrePaidServerRootVolumeVolumetypeEnum().GPSSD,
		Size:       ptr.To(int32(i.RootVolume.Size)),
	}
	switch i.RootVolume.Type {
	case infrav1.VolumeTypeGPSSD:
		createReq.Body.Server.RootVolume.Volumetype = ecsModel.GetPrePaidServerRootVolumeVolumetypeEnum().GPSSD
	}

	response, err := s.ECSClient.CreateServers(createReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run instance")
	}

	if err := s.CheckJob(*response.JobId); err != nil {
		return nil, errors.Wrap(err, "failed to check job")
	}

	sdkInstance, err := s.ShowInstance((*response.ServerIds)[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to show server")
	}

	return s.SDKToInstance(sdkInstance)
}

func (s *Service) TerminateInstance(id string) error {
	_, err := s.ECSClient.DeleteServers(&ecsModel.DeleteServersRequest{
		Body: &ecsModel.DeleteServersRequestBody{
			Servers: []ecsModel.ServerId{
				{
					Id: id,
				},
			},
			DeletePublicip: ptr.To(true),
			DeleteVolume:   ptr.To(true),
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete server")
	}

	return nil
}

func (s *Service) SDKToInstance(v *ecsModel.ShowServerResponse) (*infrav1.Instance, error) {
	if v == nil || v.Server == nil {
		return nil, fmt.Errorf("server response or server details is nil")
	}

	instance := &infrav1.Instance{
		ID:   v.Server.Id,
		Type: v.Server.Flavor.Name,
	}

	// Map status to instance state
	switch v.Server.Status {
	case "ACTIVE":
		instance.State = infrav1.InstanceStateRunning
	case "BUILD", "REBUILD":
		instance.State = infrav1.InstanceStatePending
	case "SHUTOFF":
		instance.State = infrav1.InstanceStateStopped
	case "REBOOT", "HARD_REBOOT":
		instance.State = infrav1.InstanceStateRunning
	case "DELETED", "SOFT_DELETED":
		instance.State = infrav1.InstanceStateTerminated
	default:
		instance.State = infrav1.InstanceState(strings.ToLower(v.Server.Status))
	}

	addrs := []clusterv1.MachineAddress{}
	// Get private IP from the first available network interface
	for _, addresses := range v.Server.Addresses {
		for _, addr := range addresses {
			if *addr.OSEXTIPStype == ecsModel.GetServerAddressOSEXTIPStypeEnum().FIXED {
				instance.PrivateIP = &addr.Addr
				addrs = append(addrs, clusterv1.MachineAddress{
					Type:    clusterv1.MachineInternalIP,
					Address: addr.Addr,
				})
				break
			}
		}
		if instance.PrivateIP != nil {
			break
		}
	}

	// Get public IP from the first available network interface
	for _, addresses := range v.Server.Addresses {
		for _, addr := range addresses {
			if *addr.OSEXTIPStype == ecsModel.GetServerAddressOSEXTIPStypeEnum().FLOATING {
				instance.PublicIP = &addr.Addr
				addrs = append(addrs, clusterv1.MachineAddress{
					Type:    clusterv1.MachineExternalIP,
					Address: addr.Addr,
				})
				break
			}
		}
		if instance.PublicIP != nil {
			break
		}
	}

	instance.Addresses = addrs

	if v.Server.Image != nil {
		instance.ImageID = v.Server.Image.Id
	}

	instance.AvailabilityZone = v.Server.OSEXTAZavailabilityZone

	return instance, nil
}

func (s *Service) ShowInstance(serverId string) (*ecsModel.ShowServerResponse, error) {
	return s.ECSClient.ShowServer(&ecsModel.ShowServerRequest{
		ServerId: serverId,
	})
}

func (s *Service) CheckJob(jobId string) error {
	req := &ecsModel.ShowJobRequest{
		JobId: jobId,
	}

	timeout := time.After(time.Minute)
	for {
		resp, err := s.ECSClient.ShowJob(req)
		if err != nil {
			return fmt.Errorf("failed to show job: %v", err)
		}

		switch *resp.Status {
		case ecsModel.GetShowJobResponseStatusEnum().SUCCESS:
			return nil
		case ecsModel.GetShowJobResponseStatusEnum().FAIL:
			return fmt.Errorf("job failed, type: %s, errorcode: %s, reason: %s",
				*resp.JobType, *resp.ErrorCode, *resp.FailReason)
		case ecsModel.GetShowJobResponseStatusEnum().INIT, ecsModel.GetShowJobResponseStatusEnum().RUNNING:
			select {
			case <-timeout:
				return fmt.Errorf("job timed out after 1 minute")
			case <-time.After(1 * time.Second):
				continue
			}
		default:
			return fmt.Errorf("unknown job status: %s", *resp.Status)
		}
	}
}

func (s *Service) InstanceIfExists(id *string) (*infrav1.Instance, error) {
	if id == nil {
		klog.Info("Instance does not have an instance id")
		return nil, nil
	}

	klog.Info("Looking for instance by id", "instance-id", *id)

	out, err := s.ShowInstance(*id)
	switch {
	case ecserrors.IsNotFound(err):
		return nil, ErrInstanceNotFoundByID
	case err != nil:
		return nil, ErrShowInstance
	}

	return s.SDKToInstance(out)
}

func (s *Service) AttachInstanceToElb(instance *infrav1.Instance) error {
	return s.elbService.CreateMember(s.scope.ELB().Pools, instance)
}

func (s *Service) DetachInstanceFromElb(instance *infrav1.Instance) error {
	return s.elbService.DeleteMember(s.scope.ELB().Pools, instance)
}
