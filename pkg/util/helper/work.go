package helper

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	workv1alpha1 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha1"
	workv1alpha2 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha2"
	"github.com/karmada-io/karmada/pkg/util"
	"github.com/karmada-io/karmada/pkg/util/backoff"
	"github.com/karmada-io/karmada/pkg/util/names"
)

// CreateOrUpdateWork creates a Work object if not exist, or updates if it already exist.
func CreateOrUpdateWork(client client.Client, workMeta metav1.ObjectMeta, resource *unstructured.Unstructured, group string) error {
	workload := resource.DeepCopy()
	util.MergeAnnotation(workload, workv1alpha2.ResourceTemplateUIDAnnotation, string(workload.GetUID()))
	util.RecordManagedAnnotations(workload)
	util.RecordManagedLabels(workload)
	workloadJSON, err := workload.MarshalJSON()
	if err != nil {
		klog.Errorf("Failed to marshal workload(%s/%s), Error: %v", workload.GetNamespace(), workload.GetName(), err)
		return err
	}

	work := &workv1alpha1.Work{
		ObjectMeta: workMeta,
		Spec: workv1alpha1.WorkSpec{
			Workload: workv1alpha1.WorkloadTemplate{
				Manifests: []workv1alpha1.Manifest{
					{
						RawExtension: runtime.RawExtension{
							Raw: workloadJSON,
						},
					},
				},
			},
		},
	}

	runtimeObject := work.DeepCopy()
	var operationResult controllerutil.OperationResult
	runtimeObjectCopy := work.DeepCopy()
	attempt := 0
	err = retry.RetryOnConflict(backoff.Retry, func() (err error) {
		attempt++
		operationResult, err = controllerutil.CreateOrUpdate(context.TODO(), client, runtimeObject, func() error {
			klog.Infof("[Group: %s] Attempt to ensure work %s/%s for %d times, ResoureceVersion: OLD: %s, CUR: %s; Diff: %s",
				group, work.Namespace, work.Name, attempt, runtimeObjectCopy.ResourceVersion, runtimeObject.ResourceVersion, util.TellDiffForObjects(runtimeObjectCopy, runtimeObject))
			runtimeObjectCopy = runtimeObject.DeepCopy()
			runtimeObject.Spec = work.Spec
			runtimeObject.Labels = work.Labels
			runtimeObject.Annotations = work.Annotations
			return nil
		})
		if err != nil {
			klog.Errorf("create or update work %s/%s fail, opResult: %v, err: %v", work.GetNamespace(), work.GetName(), operationResult, err)
			return err
		}
		return nil
	})
	if err != nil {
		klog.Errorf("[Group: %s] Failed to create/update work %s/%s. Error: %v", group, work.GetNamespace(), work.GetName(), err)
		return err
	}

	if operationResult == controllerutil.OperationResultCreated {
		klog.V(2).Infof("Create work %s/%s successfully.", work.GetNamespace(), work.GetName())
	} else if operationResult == controllerutil.OperationResultUpdated {
		workNew := &workv1alpha1.Work{}
		err = client.Get(context.TODO(), types.NamespacedName{Namespace: workMeta.Namespace, Name: workMeta.Name}, workNew)
		if err != nil {
			klog.Errorf("[Group %s] Failed to get latest Work(%s/%s): %v", group, workMeta.Namespace, workMeta.Name, err)
		} else {
			klog.Infof("[Group %s] Updated Work(%s/%s): resourceVersion: OLD: %s, NEW: %s; Diff: %s",
				group, workMeta.Namespace, workMeta.Name, runtimeObjectCopy.ResourceVersion, workNew.ResourceVersion, util.TellDiffForObjects(runtimeObjectCopy, workNew))
		}
		klog.V(2).Infof("[Group %s] Update work %s/%s successfully.", group, work.GetNamespace(), work.GetName())
	} else {
		klog.V(2).Infof("[Group %s] Work %s/%s is up to date.", group, work.GetNamespace(), work.GetName())
	}

	return nil
}

// GetWorksByLabelsSet get WorkList by matching labels.Set.
func GetWorksByLabelsSet(c client.Client, ls labels.Set) (*workv1alpha1.WorkList, error) {
	workList := &workv1alpha1.WorkList{}
	listOpt := &client.ListOptions{LabelSelector: labels.SelectorFromSet(ls)}

	return workList, c.List(context.TODO(), workList, listOpt)
}

// GetWorksByBindingNamespaceName get WorkList by matching same Namespace and same Name.
func GetWorksByBindingNamespaceName(c client.Client, bindingNamespace, bindingName string) (*workv1alpha1.WorkList, error) {
	referenceKey := names.GenerateBindingReferenceKey(bindingNamespace, bindingName)
	var ls labels.Set
	if bindingNamespace != "" {
		ls = labels.Set{workv1alpha2.ResourceBindingReferenceKey: referenceKey}
	} else {
		ls = labels.Set{workv1alpha2.ClusterResourceBindingReferenceKey: referenceKey}
	}

	workList, err := GetWorksByLabelsSet(c, ls)
	if err != nil {
		return nil, err
	}
	retWorkList := &workv1alpha1.WorkList{}
	// Due to the hash collision problem, we have to filter the Works by annotation.
	// More details please refer to https://github.com/karmada-io/karmada/issues/2071.
	for i := range workList.Items {
		if len(bindingNamespace) > 0 { // filter Works that derived by 'ResourceBinding'
			if util.GetAnnotationValue(workList.Items[i].GetAnnotations(), workv1alpha2.ResourceBindingNameAnnotationKey) == bindingName &&
				util.GetAnnotationValue(workList.Items[i].GetAnnotations(), workv1alpha2.ResourceBindingNamespaceAnnotationKey) == bindingNamespace {
				retWorkList.Items = append(retWorkList.Items, workList.Items[i])
			}
		} else { // filter Works that derived by 'ClusterResourceBinding'
			if util.GetAnnotationValue(workList.Items[i].GetAnnotations(), workv1alpha2.ClusterResourceBindingAnnotationKey) == bindingName {
				retWorkList.Items = append(retWorkList.Items, workList.Items[i])
			}
		}
	}

	return retWorkList, nil
}

// GenEventRef returns the event reference. sets the UID(.spec.uid) that might be missing for fire events.
// Do nothing if the UID already exist, otherwise set the UID from annotation.
func GenEventRef(resource *unstructured.Unstructured) (*corev1.ObjectReference, error) {
	ref := &corev1.ObjectReference{
		Kind:       resource.GetKind(),
		Namespace:  resource.GetNamespace(),
		Name:       resource.GetName(),
		UID:        resource.GetUID(),
		APIVersion: resource.GetAPIVersion(),
	}

	if len(resource.GetUID()) == 0 {
		uid := util.GetAnnotationValue(resource.GetAnnotations(), workv1alpha2.ResourceTemplateUIDAnnotation)
		ref.UID = types.UID(uid)
	}

	if len(ref.UID) == 0 {
		return nil, fmt.Errorf("missing mandatory uid")
	}

	if len(ref.Name) == 0 {
		return nil, fmt.Errorf("missing mandatory name")
	}

	if len(ref.Kind) == 0 {
		return nil, fmt.Errorf("missing mandatory kind")
	}

	return ref, nil
}
