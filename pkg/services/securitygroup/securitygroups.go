package securitygroup

import (
	"fmt"

	infrav1alpha1 "github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/api/v1alpha1"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/model"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
)

func (s *Service) ReconcileSecurityGroups() error {
	klog.Info("Reconciling security groups")

	securityGroupName := "sg-caph"

	// Check if the security group already exists
	listSecurityGroupsRequest := &model.ListSecurityGroupsRequest{
		VpcId: &s.scope.VPC().Id,
	}
	listSecurityGroupsResponse, err := s.vpcClient.ListSecurityGroups(listSecurityGroupsRequest)
	if err != nil {
		return fmt.Errorf("failed to list security groups: %v", err)
	}

	var securityGroupID string
	for _, sg := range *listSecurityGroupsResponse.SecurityGroups {
		if sg.Name == securityGroupName {
			securityGroupID = sg.Id
			klog.Infof("Security group already exists: %s", securityGroupID)
			break
		}
	}
	// If the security group does not exist, create it
	if securityGroupID == "" {
		createSecurityGroupRequest := &model.CreateSecurityGroupRequest{
			Body: &model.CreateSecurityGroupRequestBody{
				SecurityGroup: &model.CreateSecurityGroupOption{
					VpcId: &s.scope.VPC().Id,
					Name:  securityGroupName,
				},
			},
		}
		createSecurityGroupResponse, err := s.vpcClient.CreateSecurityGroup(createSecurityGroupRequest)
		if err != nil {
			return fmt.Errorf("failed to create security group: %v", err)
		}
		securityGroupID = createSecurityGroupResponse.SecurityGroup.Id
		klog.Infof("Created security group: %s", securityGroupID)

		// Delete insecure security group rules
		if err := s.deleteDefaultSecurityGroupRules(securityGroupID); err != nil {
			return err
		}
	}

	// Define ingress rules
	ingressRules := []model.NeutronCreateSecurityGroupRuleOption{
		{
			SecurityGroupId: securityGroupID,
			Direction:       model.GetNeutronCreateSecurityGroupRuleOptionDirectionEnum().INGRESS,
			PortRangeMax:    int32Ptr(6443),
			PortRangeMin:    int32Ptr(6443),
			Protocol:        strPtr("tcp"),
			RemoteIpPrefix:  strPtr("0.0.0.0/0"),
		},
	}

	securityGroupRules := []infrav1alpha1.SecurityGroupRule{}
	// Check and create ingress rules
	for _, rule := range ingressRules {
		if !s.securityGroupRuleExists(securityGroupID, rule) {
			createSecurityGroupRuleRequest := &model.NeutronCreateSecurityGroupRuleRequest{
				Body: &model.NeutronCreateSecurityGroupRuleRequestBody{
					SecurityGroupRule: &rule,
				},
			}
			ruleRep, err := s.vpcClient.NeutronCreateSecurityGroupRule(createSecurityGroupRuleRequest)
			if err != nil {
				return fmt.Errorf("failed to create security group rule: %v", err)
			}

			securityGroupRules = append(securityGroupRules, infrav1alpha1.SecurityGroupRule{
				Id:             ruleRep.SecurityGroupRule.Id,
				Direction:      ruleRep.SecurityGroupRule.Direction.Value(),
				Ethertype:      ruleRep.SecurityGroupRule.Ethertype,
				PortRangeMin:   ruleRep.SecurityGroupRule.PortRangeMin,
				PortRangeMax:   ruleRep.SecurityGroupRule.PortRangeMax,
				Protocol:       ruleRep.SecurityGroupRule.Protocol,
				RemoteIpPrefix: ruleRep.SecurityGroupRule.RemoteIpPrefix,
			})
			klog.Infof("Created security group rule: %+v", rule)
		} else {
			klog.Infof("Security group rule already exists: %+v", rule)
		}
	}

	securityGroups := s.scope.SecurityGroups()
	if securityGroups == nil {
		securityGroups = make(map[infrav1alpha1.SecurityGroupRole]infrav1alpha1.SecurityGroup)
		s.scope.SetSecurityGroups(securityGroups)
	}

	for _, role := range s.roles {
		securityGroups[role] = infrav1alpha1.SecurityGroup{
			ID:                 securityGroupID,
			Name:               securityGroupName,
			SecurityGroupRules: securityGroupRules,
		}
	}

	s.scope.SetSecurityGroups(securityGroups)

	conditions.MarkTrue(s.scope.InfraCluster(), infrav1alpha1.ClusterSecurityGroupsReadyCondition)
	if err := s.scope.PatchObject(); err != nil {
		return fmt.Errorf("failed to patch HCCluster: %v", err)
	}
	return nil
}

func (s *Service) deleteDefaultSecurityGroupRules(securityGroupId string) error {
	listReq := &model.ListSecurityGroupRulesRequest{SecurityGroupId: &securityGroupId}
	listResponse, err := s.vpcClient.ListSecurityGroupRules(listReq)
	if err != nil {
		return fmt.Errorf("failed to list security group rule: %v", err)
	}
	for _, rule := range *listResponse.SecurityGroupRules {
		if rule.Direction == "ingress" && rule.RemoteIpPrefix == "0.0.0.0/0" {
			deleteReq := &model.DeleteSecurityGroupRuleRequest{SecurityGroupRuleId: rule.Id}
			_, err = s.vpcClient.DeleteSecurityGroupRule(deleteReq)
			if err != nil {
				return fmt.Errorf("failed to delete security group rule: %v", err)
			}
		}
	}
	return nil
}

func (s *Service) securityGroupRuleExists(
	securityGroupID string, rule model.NeutronCreateSecurityGroupRuleOption,
) bool {
	listSecurityGroupRulesRequest := &model.ListSecurityGroupRulesRequest{
		SecurityGroupId: &securityGroupID,
	}
	listSecurityGroupRulesResponse, err := s.vpcClient.ListSecurityGroupRules(listSecurityGroupRulesRequest)
	if err != nil {
		klog.Errorf("Failed to list security group rules: %v", err)
		return false
	}

	for _, existingRule := range *listSecurityGroupRulesResponse.SecurityGroupRules {
		if existingRule.Direction == rule.Direction.Value() &&
			existingRule.PortRangeMin == *rule.PortRangeMin &&
			existingRule.PortRangeMax == *rule.PortRangeMax &&
			existingRule.Protocol == *rule.Protocol &&
			existingRule.RemoteIpPrefix == *rule.RemoteIpPrefix {
			return true
		}
	}
	return false
}

func int32Ptr(i int32) *int32 {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func (s *Service) DeleteSecurityGroups() error {
	klog.Info("Deleting security groups")
	if s.scope.VPC().Id == "" {
		klog.Info("Skipping security group deletion, vpc-id is nil", "vpc-id", s.scope.VPC().Id)
		conditions.MarkFalse(
			s.scope.InfraCluster(),
			infrav1alpha1.ClusterSecurityGroupsReadyCondition,
			clusterv1.DeletedReason,
			clusterv1.ConditionSeverityInfo,
			"")
		return nil
	}

	conditions.MarkFalse(
		s.scope.InfraCluster(),
		infrav1alpha1.ClusterSecurityGroupsReadyCondition,
		clusterv1.DeletingReason,
		clusterv1.ConditionSeverityInfo,
		"")
	if err := s.scope.PatchObject(); err != nil {
		return err
	}

	// Retrieve the security group by name
	securityGroupName := "sg-caph"
	listSecurityGroupsRequest := &model.NeutronListSecurityGroupsRequest{
		Name: &securityGroupName,
	}
	listSecurityGroupsResponse, err := s.vpcClient.NeutronListSecurityGroups(listSecurityGroupsRequest)
	if err != nil {
		conditions.MarkFalse(
			s.scope.InfraCluster(),
			infrav1alpha1.ClusterSecurityGroupsReadyCondition,
			"DeletingFailed",
			clusterv1.ConditionSeverityWarning,
			"failed to list security groups")
		return fmt.Errorf("failed to list security groups: %v", err)
	}
	securityGroups := *listSecurityGroupsResponse.SecurityGroups
	if len(securityGroups) == 0 {
		klog.Infof("No security group found with name: %s", securityGroupName)
		conditions.MarkFalse(
			s.scope.InfraCluster(),
			infrav1alpha1.ClusterSecurityGroupsReadyCondition,
			clusterv1.DeletedReason,
			clusterv1.ConditionSeverityInfo,
			"")
		return nil
	}
	securityGroupID := securityGroups[0].Id
	klog.Infof("Found security group: %s", securityGroupID)
	// Delete all security group rules
	listSecurityGroupRulesRequest := &model.NeutronListSecurityGroupRulesRequest{
		SecurityGroupId: &securityGroupID,
	}
	listSecurityGroupRulesResponse, err := s.vpcClient.NeutronListSecurityGroupRules(listSecurityGroupRulesRequest)
	if err != nil {
		conditions.MarkFalse(
			s.scope.InfraCluster(),
			infrav1alpha1.ClusterSecurityGroupsReadyCondition,
			"DeletingFailed",
			clusterv1.ConditionSeverityWarning,
			"failed to list security group rules")
		return fmt.Errorf("failed to list security group rules: %v", err)
	}
	for _, rule := range *listSecurityGroupRulesResponse.SecurityGroupRules {
		deleteSecurityGroupRuleRequest := &model.NeutronDeleteSecurityGroupRuleRequest{
			SecurityGroupRuleId: rule.Id,
		}
		_, err := s.vpcClient.NeutronDeleteSecurityGroupRule(deleteSecurityGroupRuleRequest)
		if err != nil {
			conditions.MarkFalse(
				s.scope.InfraCluster(),
				infrav1alpha1.ClusterSecurityGroupsReadyCondition,
				"DeletingFailed",
				clusterv1.ConditionSeverityWarning,
				"failed to delete security group rule")
			return fmt.Errorf("failed to delete security group rule: %v", err)
		}
		klog.Infof("Deleted security group rule: %s", rule.Id)
	}
	// Delete the security group
	deleteSecurityGroupRequest := &model.NeutronDeleteSecurityGroupRequest{
		SecurityGroupId: securityGroupID,
	}
	_, err = s.vpcClient.NeutronDeleteSecurityGroup(deleteSecurityGroupRequest)
	if err != nil {
		conditions.MarkFalse(
			s.scope.InfraCluster(),
			infrav1alpha1.ClusterSecurityGroupsReadyCondition,
			"DeletingFailed",
			clusterv1.ConditionSeverityWarning,
			"failed to delete security group")
		return fmt.Errorf("failed to delete security group: %v", err)
	}
	conditions.MarkFalse(
		s.scope.InfraCluster(),
		infrav1alpha1.ClusterSecurityGroupsReadyCondition,
		clusterv1.DeletedReason,
		clusterv1.ConditionSeverityInfo,
		"")
	klog.Infof("Deleted security group: %s", securityGroupID)
	return nil
}
