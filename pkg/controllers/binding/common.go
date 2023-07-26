package binding

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	configv1alpha1 "github.com/karmada-io/karmada/pkg/apis/config/v1alpha1"
	workv1alpha1 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha1"
	workv1alpha2 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha2"
	"github.com/karmada-io/karmada/pkg/resourceinterpreter"
	"github.com/karmada-io/karmada/pkg/util"
	"github.com/karmada-io/karmada/pkg/util/helper"
	"github.com/karmada-io/karmada/pkg/util/names"
	"github.com/karmada-io/karmada/pkg/util/overridemanager"
)

var bindingPredicateFn = builder.WithPredicates(predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool {
		return true
	},
	UpdateFunc: func(e event.UpdateEvent) bool {
		oldBinding := e.ObjectOld.(*workv1alpha2.ResourceBinding)
		newBinding := e.ObjectNew.(*workv1alpha2.ResourceBinding)

		needUpdate := predicate.GenerationChangedPredicate{}.Update(e)
		if needUpdate {
			klog.Infof("Enqueue binding(%s/%s) for binding update event. ResourceVersion: OLD: %s, NEW: %s. Diff: %s.",
				newBinding.Namespace, newBinding.Name, oldBinding.ResourceVersion, newBinding.ResourceVersion, util.TellDiffForObjects(oldBinding, newBinding))
		}
		return needUpdate
	},
	DeleteFunc: func(event.DeleteEvent) bool {
		return true
	},
	GenericFunc: func(event.GenericEvent) bool {
		return false
	},
})

// ensureWork ensure Work to be created or updated.
func ensureWork(
	c client.Client, resourceInterpreter resourceinterpreter.ResourceInterpreter, workload *unstructured.Unstructured,
	overrideManager overridemanager.OverrideManager, binding metav1.Object, scope apiextensionsv1.ResourceScope, group string,
) error {
	var targetClusters []workv1alpha2.TargetCluster
	var requiredByBindingSnapshot []workv1alpha2.BindingSnapshot
	switch scope {
	case apiextensionsv1.NamespaceScoped:
		bindingObj := binding.(*workv1alpha2.ResourceBinding)
		targetClusters = bindingObj.Spec.Clusters
		requiredByBindingSnapshot = bindingObj.Spec.RequiredBy
	case apiextensionsv1.ClusterScoped:
		bindingObj := binding.(*workv1alpha2.ClusterResourceBinding)
		targetClusters = bindingObj.Spec.Clusters
		requiredByBindingSnapshot = bindingObj.Spec.RequiredBy
	}

	targetClusters = mergeTargetClusters(targetClusters, requiredByBindingSnapshot)

	var jobCompletions []workv1alpha2.TargetCluster
	var err error
	if workload.GetKind() == util.JobKind {
		jobCompletions, err = divideReplicasByJobCompletions(workload, targetClusters)
		if err != nil {
			return err
		}
	}
	hasScheduledReplica, desireReplicaInfos := getReplicaInfos(targetClusters)

	for i := range targetClusters {
		targetCluster := targetClusters[i]
		clonedWorkload := workload.DeepCopy()

		workNamespace := names.GenerateExecutionSpaceName(targetCluster.Name)

		if hasScheduledReplica {
			if resourceInterpreter.HookEnabled(clonedWorkload.GroupVersionKind(), configv1alpha1.InterpreterOperationReviseReplica) {
				clonedWorkload, err = resourceInterpreter.ReviseReplica(clonedWorkload, desireReplicaInfos[targetCluster.Name], targetCluster.Name)
				if err != nil {
					klog.Errorf("Failed to revise replica for %s/%s/%s in cluster %s, err is: %v",
						workload.GetKind(), workload.GetNamespace(), workload.GetName(), targetCluster.Name, err)
					return err
				}
			}

			// Set allocated completions for Job only when the '.spec.completions' field not omitted from resource template.
			// For jobs running with a 'work queue' usually leaves '.spec.completions' unset, in that case we skip
			// setting this field as well.
			// Refer to: https://kubernetes.io/docs/concepts/workloads/controllers/job/#parallel-jobs.
			if len(jobCompletions) > 0 {
				if err = helper.ApplyReplica(clonedWorkload, int64(jobCompletions[i].Replicas), util.CompletionsField); err != nil {
					klog.Errorf("Failed to apply Completions for %s/%s/%s in cluster %s, err is: %v",
						clonedWorkload.GetKind(), clonedWorkload.GetNamespace(), clonedWorkload.GetName(), targetCluster.Name, err)
					return err
				}
			}
		}

		// We should call ApplyOverridePolicies last, as override rules have the highest priority
		cops, ops, err := overrideManager.ApplyOverridePolicies(clonedWorkload, targetCluster.Name)
		if err != nil {
			klog.Errorf("Failed to apply overrides for %s/%s/%s, err is: %v", clonedWorkload.GetKind(), clonedWorkload.GetNamespace(), clonedWorkload.GetName(), err)
			return err
		}
		workLabel := mergeLabel(clonedWorkload, workNamespace, binding, scope)

		annotations := mergeAnnotations(clonedWorkload, binding, scope)
		annotations, err = RecordAppliedOverrides(cops, ops, annotations)
		if err != nil {
			klog.Errorf("Failed to record appliedOverrides, Error: %v", err)
			return err
		}

		workMeta := metav1.ObjectMeta{
			Name:        names.GenerateWorkName(clonedWorkload.GetKind(), clonedWorkload.GetName(), clonedWorkload.GetNamespace()),
			Namespace:   workNamespace,
			Finalizers:  []string{util.ExecutionControllerFinalizer},
			Labels:      workLabel,
			Annotations: annotations,
		}

		if err = helper.CreateOrUpdateWork(c, workMeta, clonedWorkload, group); err != nil {
			return err
		}
	}
	return nil
}

func mergeTargetClusters(targetClusters []workv1alpha2.TargetCluster, requiredByBindingSnapshot []workv1alpha2.BindingSnapshot) []workv1alpha2.TargetCluster {
	if len(requiredByBindingSnapshot) == 0 {
		return targetClusters
	}

	scheduledClusterNames := util.ConvertToClusterNames(targetClusters)

	for _, requiredByBinding := range requiredByBindingSnapshot {
		for _, targetCluster := range requiredByBinding.Clusters {
			if !scheduledClusterNames.Has(targetCluster.Name) {
				scheduledClusterNames.Insert(targetCluster.Name)
				targetClusters = append(targetClusters, targetCluster)
			}
		}
	}

	return targetClusters
}

// Always returns true because we need to trigger reviseReplica even if the replicas is 0.
func getReplicaInfos(targetClusters []workv1alpha2.TargetCluster) (bool, map[string]int64) {
	// if helper.HasScheduledReplica(targetClusters) {
	return true, transScheduleResultToMap(targetClusters)
	// }
	// return false, nil
}

func transScheduleResultToMap(scheduleResult []workv1alpha2.TargetCluster) map[string]int64 {
	var desireReplicaInfos = make(map[string]int64, len(scheduleResult))
	for _, clusterInfo := range scheduleResult {
		desireReplicaInfos[clusterInfo.Name] = int64(clusterInfo.Replicas)
	}
	return desireReplicaInfos
}

func mergeLabel(workload *unstructured.Unstructured, workNamespace string, binding metav1.Object, scope apiextensionsv1.ResourceScope) map[string]string {
	var workLabel = make(map[string]string)
	util.MergeLabel(workload, workv1alpha1.WorkNamespaceLabel, workNamespace)
	util.MergeLabel(workload, workv1alpha1.WorkNameLabel, names.GenerateWorkName(workload.GetKind(), workload.GetName(), workload.GetNamespace()))
	if scope == apiextensionsv1.NamespaceScoped {
		util.MergeLabel(workload, workv1alpha2.ResourceBindingReferenceKey, names.GenerateBindingReferenceKey(binding.GetNamespace(), binding.GetName()))
		workLabel[workv1alpha2.ResourceBindingReferenceKey] = names.GenerateBindingReferenceKey(binding.GetNamespace(), binding.GetName())
	} else {
		util.MergeLabel(workload, workv1alpha2.ClusterResourceBindingReferenceKey, names.GenerateBindingReferenceKey("", binding.GetName()))
		workLabel[workv1alpha2.ClusterResourceBindingReferenceKey] = names.GenerateBindingReferenceKey("", binding.GetName())
	}
	return workLabel
}

func mergeAnnotations(workload *unstructured.Unstructured, binding metav1.Object, scope apiextensionsv1.ResourceScope) map[string]string {
	annotations := make(map[string]string)
	if scope == apiextensionsv1.NamespaceScoped {
		util.MergeAnnotation(workload, workv1alpha2.ResourceBindingNamespaceAnnotationKey, binding.GetNamespace())
		util.MergeAnnotation(workload, workv1alpha2.ResourceBindingNameAnnotationKey, binding.GetName())
		annotations[workv1alpha2.ResourceBindingNamespaceAnnotationKey] = binding.GetNamespace()
		annotations[workv1alpha2.ResourceBindingNameAnnotationKey] = binding.GetName()
	} else {
		util.MergeAnnotation(workload, workv1alpha2.ClusterResourceBindingAnnotationKey, binding.GetName())
		annotations[workv1alpha2.ClusterResourceBindingAnnotationKey] = binding.GetName()
	}

	return annotations
}

// RecordAppliedOverrides record applied (cluster) overrides to annotations
func RecordAppliedOverrides(cops *overridemanager.AppliedOverrides, ops *overridemanager.AppliedOverrides,
	annotations map[string]string) (map[string]string, error) {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if cops != nil {
		appliedBytes, err := cops.MarshalJSON()
		if err != nil {
			return nil, err
		}
		if appliedBytes != nil {
			annotations[util.AppliedClusterOverrides] = string(appliedBytes)
		}
	}

	if ops != nil {
		appliedBytes, err := ops.MarshalJSON()
		if err != nil {
			return nil, err
		}
		if appliedBytes != nil {
			annotations[util.AppliedOverrides] = string(appliedBytes)
		}
	}

	return annotations, nil
}

func divideReplicasByJobCompletions(workload *unstructured.Unstructured, clusters []workv1alpha2.TargetCluster) ([]workv1alpha2.TargetCluster, error) {
	var targetClusters []workv1alpha2.TargetCluster
	completions, found, err := unstructured.NestedInt64(workload.Object, util.SpecField, util.CompletionsField)
	if err != nil {
		return nil, err
	}

	if found {
		targetClusters = helper.SpreadReplicasByTargetClusters(int32(completions), clusters, nil)
	}

	return targetClusters, nil
}
