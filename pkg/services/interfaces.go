package services

import (
	infrav1 "github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/api/v1alpha1"
	"github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/pkg/scope"
)

// ECSInterface encapsulates the methods exposed to the machine
// actuator.
type ECSInterface interface {
	InstanceIfExists(id *string) (*infrav1.Instance, error)
	CreateInstance(scope *scope.MachineScope, userData []byte, userDataFormat string) (*infrav1.Instance, error)
	TerminateInstance(id string) error
}
