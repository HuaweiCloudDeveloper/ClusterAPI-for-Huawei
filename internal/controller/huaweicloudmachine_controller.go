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

package controller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"

	infrav1 "github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/api/v1alpha1"
	hwbasic "github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/pkg/basic"
	"github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/pkg/scope"
	"github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/pkg/services"
	"github.com/HuaweiCloudDeveloper/cluster-api-provider-huawei/pkg/services/ecs"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
)

const (
	// DefaultReconcilerRequeue is the default value for the reconcile retry.
	DefaultReconcilerRequeue = 30 * time.Second
)

// HuaweiCloudMachineReconciler reconciles a HuaweiCloudMachine object
type HuaweiCloudMachineReconciler struct {
	client.Client
	Recorder    record.EventRecorder
	Scheme      *runtime.Scheme
	Credentials *basic.Credentials
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=huaweicloudmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=huaweicloudmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=huaweicloudmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HuaweiCloudMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *HuaweiCloudMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := log.FromContext(ctx)

	hcMachine := &infrav1.HuaweiCloudMachine{}
	err := r.Get(ctx, req.NamespacedName, hcMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	machine, err := util.GetOwnerMachine(ctx, r.Client, hcMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("machine", klog.KObj(machine))

	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("cluster", klog.KObj(cluster))

	infraCluster, err := r.getInfraCluster(ctx, &log, cluster, hcMachine)
	if err != nil {
		return ctrl.Result{}, errors.Errorf("error getting infra provider cluster or control plane object: %v", err)
	}
	if infraCluster == nil {
		log.Info("HuaweiCloudCluster is not ready yet")
		return ctrl.Result{}, nil
	}

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:       r.Client,
		Cluster:      cluster,
		Machine:      machine,
		InfraCluster: infraCluster,
		HCMachine:    hcMachine,
	})
	if err != nil {
		log.Error(err, "failed to create scope")
		return ctrl.Result{}, err
	}

	// Always close the scope when exiting this function so we can persist any HuaweiCloudMachine changes.
	defer func() {
		if err := machineScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	switch infraScope := infraCluster.(type) {
	case *scope.ClusterScope:
		if !machine.ObjectMeta.DeletionTimestamp.IsZero() {
			return r.reconcileDelete(machineScope, infraScope, infraScope)
		}

		return r.reconcileNormal(ctx, machineScope, infraScope, infraScope)
	default:
		return ctrl.Result{}, errors.New("infraCluster has unknown type")
	}
}

func (r *HuaweiCloudMachineReconciler) getInfraCluster(ctx context.Context, log *logr.Logger, cluster *clusterv1.Cluster, hcMachine *infrav1.HuaweiCloudMachine) (scope.ECSScope, error) {
	var clusterScope *scope.ClusterScope

	hcCluster := &infrav1.HuaweiCloudCluster{}

	infraClusterName := client.ObjectKey{
		Namespace: hcMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err := r.Client.Get(ctx, infraClusterName, hcCluster); err != nil {
		// HuaweiCloudCluster is not ready
		return nil, nil //nolint:nilerr
	}

	// Create the cluster scope
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:      r.Client,
		Logger:      log,
		Cluster:     cluster,
		HCCluster:   hcCluster,
		Credentials: r.Credentials,
	})
	if err != nil {
		return nil, err
	}

	return clusterScope, nil
}

// findInstance queries the ECS apis and retrieves the instance if it exists.
// If providerID is empty, finds instance by tags and if it cannot be found, returns empty instance with nil error.
// If providerID is set, either finds the instance by ID or returns error.
func (r *HuaweiCloudMachineReconciler) findInstance(machineScope *scope.MachineScope, ecsSvc services.ECSInterface) (*infrav1.Instance, error) {
	var instance *infrav1.Instance

	// Parse the ProviderID.
	pid, err := scope.NewProviderID(machineScope.GetProviderID())
	if err != nil {
		//nolint:staticcheck
		if !errors.Is(err, scope.ErrEmptyProviderID) {
			return nil, errors.Wrapf(err, "failed to parse Spec.ProviderID")
		}
		// // If the ProviderID is empty, try to query the instance using tags.
		// // If an instance cannot be found, GetRunningInstanceByTags returns empty instance with nil error.
		// instance, err = ecsSvc.GetRunningInstanceByTags(machineScope)
		// if err != nil {
		// 	return nil, errors.Wrapf(err, "failed to query HuaweiCloudMachine instance by tags")
		// }
	} else {
		// If the ProviderID is populated, describe the instance using the ID.
		// InstanceIfExists() returns error (ErrInstanceNotFoundByID or ErrDescribeInstance) if the instance could not be found.
		//nolint:staticcheck
		instance, err = ecsSvc.InstanceIfExists(ptr.To[string](pid.ID()))
		if err != nil {
			return nil, err
		}
	}

	// The only case where the instance is nil here is when the providerId is empty and instance could not be found by tags.
	return instance, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HuaweiCloudMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.HuaweiCloudMachine{}).
		Named("huaweicloudmachine").
		Complete(r)
}

func (r *HuaweiCloudMachineReconciler) reconcileDelete(machineScope *scope.MachineScope, _ hwbasic.ClusterScoper, ecsScope scope.ECSScope) (ctrl.Result, error) {
	machineScope.Logger.Info("Handling deleted HuaweiCloudMachine")

	ecsSvc, err := ecs.NewService(ecsScope)
	if err != nil {
		machineScope.Logger.Error(err, "failed to get ECS service")
		return ctrl.Result{}, err
	}

	instance, err := r.findInstance(machineScope, ecsSvc)
	if err != nil && err != ecs.ErrInstanceNotFoundByID {
		machineScope.Logger.Error(err, "query to find instance failed")
		return ctrl.Result{}, err
	}

	if instance == nil {
		// The machine was never created or was deleted by some other entity
		// One way to reach this state:
		// 1. Scale deployment to 0
		// 2. Rename ECS machine, and delete ProviderID from spec of both Machine
		// and HuaweiCloudMachine
		// 3. Issue a delete
		// 4. Scale controller deployment to 1
		machineScope.Logger.Info("Unable to locate ECS instance by ID or tags")
		controllerutil.RemoveFinalizer(machineScope.HCMachine, infrav1.MachineFinalizer)
		return ctrl.Result{}, nil
	}

	machineScope.Logger.Info("ECS instance found matching deleted HuaweiCloudMachine", "instance-id", instance.ID)

	// Check the instance state. If it's already shutting down or terminated,
	// do nothing. Otherwise attempt to delete it.
	switch instance.State {
	case infrav1.InstanceStateShuttingDown:
		machineScope.Logger.Info("ECS instance is shutting down or already terminated", "instance-id", instance.ID)
		// requeue reconciliation until we observe termination (or the instance can no longer be looked up)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	case infrav1.InstanceStateTerminated:
		machineScope.Logger.Info("ECS instance terminated successfully", "instance-id", instance.ID)
		controllerutil.RemoveFinalizer(machineScope.HCMachine, infrav1.MachineFinalizer)
		return ctrl.Result{}, nil
	default:
		machineScope.Logger.Info("Terminating ECS instance", "instance-id", instance.ID)

		// Set the InstanceReadyCondition and patch the object before the blocking operation
		conditions.MarkFalse(machineScope.HCMachine, infrav1.InstanceReadyCondition, clusterv1.DeletingReason, clusterv1.ConditionSeverityInfo, "")
		if err := machineScope.PatchObject(); err != nil {
			machineScope.Logger.Error(err, "failed to patch object")
			return ctrl.Result{}, err
		}

		if err := ecsSvc.TerminateInstance(instance.ID); err != nil {
			machineScope.Logger.Error(err, "failed to terminate instance")
			conditions.MarkFalse(machineScope.HCMachine, infrav1.InstanceReadyCondition, "DeletingFailed", clusterv1.ConditionSeverityWarning, "failed to terminate instance: %v", err)
			return ctrl.Result{}, err
		}
		conditions.MarkFalse(machineScope.HCMachine, infrav1.InstanceReadyCondition, clusterv1.DeletedReason, clusterv1.ConditionSeverityInfo, "")

		machineScope.Logger.Info("ECS instance successfully terminated", "instance-id", instance.ID)

		return ctrl.Result{}, nil
	}
}

func (r *HuaweiCloudMachineReconciler) reconcileNormal(_ context.Context, machineScope *scope.MachineScope, _ hwbasic.ClusterScoper, ecsScope scope.ECSScope) (ctrl.Result, error) {
	machineScope.Logger.Info("Reconciling HuaweiCloudMachine")

	ecsSvc, err := ecs.NewService(ecsScope)
	if err != nil {
		machineScope.Logger.Error(err, "failed to get ECS service")
		return ctrl.Result{}, err
	}

	// Find existing instance
	instance, err := r.findInstance(machineScope, ecsSvc)
	if err != nil {
		machineScope.Logger.Error(err, "unable to find instance")
		conditions.MarkUnknown(machineScope.HCMachine, infrav1.InstanceReadyCondition, infrav1.InstanceNotFoundReason, "failed to find instance: %v", err)
		return ctrl.Result{}, err
	}

	// If the HuaweiCloudMachine doesn't have our finalizer, add it.
	if controllerutil.AddFinalizer(machineScope.HCMachine, infrav1.MachineFinalizer) {
		// Register the finalizer after first read operation from HuaweiCloud to avoid orphaning HuaweiCloud resources on delete
		if err := machineScope.PatchObject(); err != nil {
			machineScope.Logger.Error(err, "unable to patch object")
			return ctrl.Result{}, err
		}
	}

	// Instance is not found, create a new one
	if instance == nil {
		// Avoid a flickering condition between InstanceProvisionStarted and InstanceProvisionFailed if there's a persistent failure with createInstance
		if conditions.GetReason(machineScope.HCMachine, infrav1.InstanceReadyCondition) != infrav1.InstanceProvisionFailedReason {
			conditions.MarkFalse(machineScope.HCMachine, infrav1.InstanceReadyCondition, infrav1.InstanceProvisionStartedReason, clusterv1.ConditionSeverityInfo, "")
			if patchErr := machineScope.PatchObject(); patchErr != nil {
				machineScope.Logger.Error(patchErr, "failed to patch conditions")
				return ctrl.Result{}, patchErr
			}
		}

		machineScope.Logger.Info("Creating ECS instance")

		userData, userDataFormat, userDataErr := r.resolveUserData(machineScope)
		if userDataErr != nil {
			return ctrl.Result{}, errors.Wrapf(userDataErr, "failed to resolve userdata")
		}

		instance, err = ecsSvc.CreateInstance(machineScope, userData, userDataFormat)
		if err != nil {
			machineScope.Logger.Error(err, "unable to create instance")
			conditions.MarkFalse(machineScope.HCMachine, infrav1.InstanceReadyCondition, infrav1.InstanceProvisionFailedReason, clusterv1.ConditionSeverityError, "failed to create instance: %v", err)
			return ctrl.Result{}, err
		}
		conditions.MarkTrue(machineScope.HCMachine, infrav1.InstanceReadyCondition)
	}

	// Make sure Spec.ProviderID and Spec.InstanceID are always set.
	machineScope.SetProviderID(instance.ID, instance.AvailabilityZone)
	machineScope.SetInstanceID(instance.ID)

	existingInstanceState := machineScope.GetInstanceState()
	machineScope.SetInstanceState(instance.State)

	// Proceed to reconcile the HuaweiCloudMachine state.
	if existingInstanceState == nil || *existingInstanceState != instance.State {
		machineScope.Logger.Info("ECS instance state changed", "state", instance.State, "instance-id", *machineScope.GetInstanceID())
	}

	shouldRequeue := false
	switch instance.State {
	case infrav1.InstanceStatePending:
		machineScope.SetNotReady()
		shouldRequeue = true
		conditions.MarkFalse(machineScope.HCMachine, infrav1.InstanceReadyCondition, infrav1.InstanceNotReadyReason, clusterv1.ConditionSeverityWarning, "")
	case infrav1.InstanceStateStopping, infrav1.InstanceStateStopped:
		machineScope.SetNotReady()
		conditions.MarkFalse(machineScope.HCMachine, infrav1.InstanceReadyCondition, infrav1.InstanceStoppedReason, clusterv1.ConditionSeverityError, "")
	case infrav1.InstanceStateRunning:
		machineScope.SetReady()
		conditions.MarkTrue(machineScope.HCMachine, infrav1.InstanceReadyCondition)
	case infrav1.InstanceStateShuttingDown, infrav1.InstanceStateTerminated:
		machineScope.SetNotReady()
		machineScope.Logger.Info("Unexpected ECS instance termination", "state", instance.State, "instance-id", *machineScope.GetInstanceID())
		conditions.MarkFalse(machineScope.HCMachine, infrav1.InstanceReadyCondition, infrav1.InstanceTerminatedReason, clusterv1.ConditionSeverityError, "")
	default:
		machineScope.SetNotReady()
		machineScope.Logger.Info("ECS instance state is undefined", "state", instance.State, "instance-id", *machineScope.GetInstanceID())
		machineScope.SetFailureReason(capierrors.UpdateMachineError)
		machineScope.SetFailureMessage(errors.Errorf("ECS instance state %q is undefined", instance.State))
		conditions.MarkUnknown(machineScope.HCMachine, infrav1.InstanceReadyCondition, "", "")
	}

	if instance.State == infrav1.InstanceStateTerminated {
		machineScope.SetFailureReason(capierrors.UpdateMachineError)
		machineScope.SetFailureMessage(errors.Errorf("ECS instance state %q is unexpected", instance.State))
	}

	// tasks that can take place during all known instance states
	if machineScope.InstanceIsInKnownState() {
		if err := ecsSvc.AttachInstanceToElb(instance); err != nil {
			machineScope.Logger.Error(err, "failed to attach instance to ELB")
			return ctrl.Result{}, err
		}
	}

	// tasks that can only take place during operational instance states
	if machineScope.InstanceIsOperational() {
		err := r.reconcileOperationalState(ecsSvc, machineScope, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	machineScope.Logger.Info("done reconciling instance", "instance", instance)
	if shouldRequeue {
		machineScope.Logger.Info("but find the instance is pending, requeue", "instance", instance.ID)
		return ctrl.Result{RequeueAfter: DefaultReconcilerRequeue}, nil
	}
	return ctrl.Result{}, nil
}

func (r *HuaweiCloudMachineReconciler) reconcileOperationalState(ecsSvc services.ECSInterface, machineScope *scope.MachineScope, instance *infrav1.Instance) error {
	machineScope.SetAddresses(instance.Addresses)

	// Ensure that the security groups are correct.
	_, err := r.ensureSecurityGroups(ecsSvc, machineScope)
	if err != nil {
		conditions.MarkFalse(machineScope.HCMachine, infrav1.SecurityGroupsReadyCondition, infrav1.SecurityGroupsFailedReason, clusterv1.ConditionSeverityError, "%s", err.Error())
		machineScope.Logger.Error(err, "unable to ensure security groups")
		return err
	}
	conditions.MarkTrue(machineScope.HCMachine, infrav1.SecurityGroupsReadyCondition)

	return nil
}

// Ensures that the security groups of the machine are correct
// Returns bool, error
// Bool indicates if changes were made or not, allowing the caller to decide
// if the machine should be updated.
func (r *HuaweiCloudMachineReconciler) ensureSecurityGroups(ecsSvc services.ECSInterface, scope *scope.MachineScope) (bool, error) {
	_, err := ecsSvc.GetCoreSecurityGroups(scope)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *HuaweiCloudMachineReconciler) resolveUserData(machineScope *scope.MachineScope) ([]byte, string, error) {
	userData, userDataFormat, err := machineScope.GetRawBootstrapDataWithFormat()
	if err != nil {
		r.Recorder.Eventf(machineScope.HCMachine, corev1.EventTypeWarning, "FailedGetBootstrapData", err.Error())
		return nil, "", err
	}
	return userData, userDataFormat, err
}
