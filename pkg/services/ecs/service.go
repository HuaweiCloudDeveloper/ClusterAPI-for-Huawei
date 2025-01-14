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
	ecsiface "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
	"github.com/pkg/errors"

	"github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/pkg/scope"
	"github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/pkg/services/network"
)

// Service holds a collection of interfaces.
// The interfaces are broken down like this to group functions together.
// One alternative is to have a large list of functions from the ecs client.
type Service struct {
	scope      scope.ECSScope
	ECSClient  *ecsiface.EcsClient
	netService *network.Service
}

// NewService returns a new service given the ECS api client.
func NewService(clusterScope scope.ECSScope) (*Service, error) {
	ecsClient, err := scope.NewECSClient(clusterScope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ECS client")
	}

	netSvc, err := network.NewService(clusterScope.(*scope.ClusterScope))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create network service")
	}

	return &Service{
		scope:      clusterScope,
		ECSClient:  ecsClient,
		netService: netSvc,
	}, nil
}
