package scope

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/api/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"

	"github.com/go-logr/logr"
)

// MachineScopeParams defines the input parameters used to create a new MachineScope.
type MachineScopeParams struct {
	Client       client.Client
	Logger       *logr.Logger
	Cluster      *clusterv1.Cluster
	Machine      *clusterv1.Machine
	InfraCluster ECSScope
	HCMachine    *infrav1.HuaweiCloudMachine
}

// NewMachineScope creates a new MachineScope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("client is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("machine is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("cluster is required when creating a MachineScope")
	}
	if params.HCMachine == nil {
		return nil, errors.New("huaweicloud machine is required when creating a MachineScope")
	}
	if params.InfraCluster == nil {
		return nil, errors.New("huaweicloud cluster is required when creating a MachineScope")
	}

	if params.Logger == nil {
		logger := logr.FromContextOrDiscard(context.Background())
		params.Logger = &logger
	}

	helper, err := patch.NewHelper(params.HCMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &MachineScope{
		Logger:       *params.Logger,
		client:       params.Client,
		patchHelper:  helper,
		Cluster:      params.Cluster,
		Machine:      params.Machine,
		InfraCluster: params.InfraCluster,
		HCMachine:    params.HCMachine,
	}, nil
}

// MachineScope defines a scope defined around a machine and its cluster.
type MachineScope struct {
	Logger      logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Cluster      *clusterv1.Cluster
	Machine      *clusterv1.Machine
	InfraCluster ECSScope
	HCMachine    *infrav1.HuaweiCloudMachine
}

// Name returns the HuaweiCloudMachine name.
func (m *MachineScope) Name() string {
	return m.HCMachine.Name
}

// Namespace returns the namespace name.
func (m *MachineScope) Namespace() string {
	return m.HCMachine.Namespace
}

// IsControlPlane returns true if the machine is a control plane.
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// Role returns the machine role from the labels.
func (m *MachineScope) Role() string {
	if util.IsControlPlaneMachine(m.Machine) {
		return "control-plane"
	}
	return "node"
}

// GetInstanceID returns the HuaweiCloudMachine instance id by parsing Spec.ProviderID.
func (m *MachineScope) GetInstanceID() *string {
	parsed, err := NewProviderID(m.GetProviderID())
	if err != nil {
		return nil
	}
	return ptr.To[string](parsed.ID())
}

// GetProviderID returns the HuaweiCloudMachine providerID from the spec.
func (m *MachineScope) GetProviderID() string {
	if m.HCMachine.Spec.ProviderID != nil {
		return *m.HCMachine.Spec.ProviderID
	}
	return ""
}

// SetProviderID sets the HuaweiCloudMachine providerID in spec.
func (m *MachineScope) SetProviderID(instanceID, availabilityZone string) {
	providerID := GenerateProviderID(availabilityZone, instanceID)
	m.HCMachine.Spec.ProviderID = ptr.To[string](providerID)
}

// SetInstanceID sets the HuaweiCloudMachine instanceID in spec.
func (m *MachineScope) SetInstanceID(instanceID string) {
	m.HCMachine.Spec.InstanceID = ptr.To[string](instanceID)
}

// GetInstanceState returns the HuaweiCloudMachine instance state from the status.
func (m *MachineScope) GetInstanceState() *infrav1.InstanceState {
	return m.HCMachine.Status.InstanceState
}

// SetInstanceState sets the HuaweiCloudMachine status instance state.
func (m *MachineScope) SetInstanceState(v infrav1.InstanceState) {
	m.HCMachine.Status.InstanceState = &v
}

// SetReady sets the HuaweiCloudMachine Ready Status.
func (m *MachineScope) SetReady() {
	m.HCMachine.Status.Ready = true
}

// SetNotReady sets the HuaweiCloudMachine Ready Status to false.
func (m *MachineScope) SetNotReady() {
	m.HCMachine.Status.Ready = false
}

// SetFailureMessage sets the HuaweiCloudMachine status failure message.
func (m *MachineScope) SetFailureMessage(v error) {
	m.HCMachine.Status.FailureMessage = ptr.To[string](v.Error())
}

// SetFailureReason sets the HuaweiCloudMachine status failure reason.
func (m *MachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.HCMachine.Status.FailureReason = &v
}

// PatchObject persists the machine spec and status.
func (m *MachineScope) PatchObject() error {
	// Always update the readyCondition by summarizing the state of other conditions.
	// A step counter is added to represent progress during the provisioning process
	// (instead we are hiding during the deletion process).
	applicableConditions := []clusterv1.ConditionType{
		infrav1.InstanceReadyCondition,
		infrav1.SecurityGroupsReadyCondition,
	}

	// if m.IsControlPlane() {
	// 	applicableConditions = append(applicableConditions, infrav1.ELBAttachedCondition)
	// }

	conditions.SetSummary(m.HCMachine,
		conditions.WithConditions(applicableConditions...),
		conditions.WithStepCounterIf(m.HCMachine.ObjectMeta.DeletionTimestamp.IsZero()),
		conditions.WithStepCounter(),
	)

	return m.patchHelper.Patch(
		context.TODO(),
		m.HCMachine,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.InstanceReadyCondition,
			infrav1.SecurityGroupsReadyCondition,
			// infrav1.ELBAttachedCondition,
		}})
}

// Close the MachineScope by updating the machine spec, machine status.
func (m *MachineScope) Close() error {
	return m.PatchObject()
}

// HasFailed returns the failure state of the machine scope.
func (m *MachineScope) HasFailed() bool {
	return m.HCMachine.Status.FailureReason != nil || m.HCMachine.Status.FailureMessage != nil
}

// InstanceIsRunning returns the instance state of the machine scope.
func (m *MachineScope) InstanceIsRunning() bool {
	state := m.GetInstanceState()
	return state != nil && infrav1.InstanceRunningStates.Has(string(*state))
}

// InstanceIsOperational returns the operational state of the machine scope.
func (m *MachineScope) InstanceIsOperational() bool {
	state := m.GetInstanceState()
	return state != nil && infrav1.InstanceOperationalStates.Has(string(*state))
}

// InstanceIsInKnownState checks if the machine scope's instance state is known.
func (m *MachineScope) InstanceIsInKnownState() bool {
	state := m.GetInstanceState()
	return state != nil && infrav1.InstanceKnownStates.Has(string(*state))
}
