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
	ecsiface "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
	ecsRegion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/region"
	"k8s.io/klog/v2"
)

func NewECSClient(scope ECSScope) (*ecsiface.EcsClient, error) {
	region, err := ecsRegion.SafeValueOf(scope.Region())
	if err != nil {
		klog.Errorf("Failed to get region: %v", err)
		return nil, err
	}
	ecsHcClient, err := ecsiface.EcsClientBuilder().
		WithRegion(region).
		WithCredential(scope.Credential()).
		SafeBuild()
	if err != nil {
		return nil, err
	}

	ecsClient := ecsiface.NewEcsClient(ecsHcClient)
	return ecsClient, nil
}
