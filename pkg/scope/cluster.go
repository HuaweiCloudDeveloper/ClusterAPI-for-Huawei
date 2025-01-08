/*
Copyright 2018 The Kubernetes Authors.

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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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

	return clusterScope, nil
}

func (s *ClusterScope) Close() error {
	// TODO: persist the cluster configuration and status.
	return nil
}

func (s *ClusterScope) PatchObject() error {
	helper, err := patch.NewHelper(s.HCCluster, s.client)
	if err != nil {
		return err
	}
	return helper.Patch(context.TODO(), s.HCCluster)
	// return s.client.Patch(context.TODO(), s.Cluster, client.MergeFrom(s.Cluster.DeepCopy()))
}

// VPC returns the cluster VPC.
func (s *ClusterScope) VPC() *infrav1alpha1.VPCSpec {
	return &s.HCCluster.Spec.NetworkSpec.VPC
}

// Region returns the cluster region.
func (s *ClusterScope) Region() string {
	return s.HCCluster.Spec.Region
}
