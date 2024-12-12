package mutating

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	workv1alpha1 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha1"
	workv1alpha2 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha2"
	"github.com/karmada-io/karmada/pkg/resourceinterpreter/default/native/prune"
	"github.com/karmada-io/karmada/pkg/util"
)

// MutateWork mutates the Work object.
func MutateWork(work *workv1alpha1.Work) error {
	var manifests []workv1alpha1.Manifest
	for _, manifest := range work.Spec.Workload.Manifests {
		workloadObj := &unstructured.Unstructured{}
		err := json.Unmarshal(manifest.Raw, workloadObj)
		if err != nil {
			klog.Errorf("Failed to unmarshal the work(%s/%s) manifest to Unstructured, err: %v", work.Namespace, work.Name, err)
			return err
		}

		err = prune.RemoveIrrelevantFields(workloadObj, prune.RemoveJobTTLSeconds)
		if err != nil {
			klog.Errorf("Failed to remove irrelevant fields for the work(%s/%s), err: %v", work.Namespace, work.Name, err)
			return err
		}

		// Skip label/annotate the workload of Work that is not intended to be propagated.
		if work.Labels[util.PropagationInstruction] != util.PropagationInstructionSuppressed {
			setLabelsAndAnnotationsForWorkload(workloadObj, work)
		}

		workloadJSON, err := workloadObj.MarshalJSON()
		if err != nil {
			klog.Errorf("Failed to marshal workload of the work(%s/%s), err: %s", work.Namespace, work.Name, err)
			return err
		}
		manifests = append(manifests, workv1alpha1.Manifest{RawExtension: runtime.RawExtension{Raw: workloadJSON}})
	}

	work.Spec.Workload.Manifests = manifests
	return nil
}

// setLabelsAndAnnotationsForWorkload sets the associated work object labels and annotations for workload.
func setLabelsAndAnnotationsForWorkload(workload *unstructured.Unstructured, work *workv1alpha1.Work) {
	util.RecordManagedAnnotations(workload)
	workload.SetLabels(util.DedupeAndMergeLabels(workload.GetLabels(), map[string]string{
		workv1alpha2.WorkPermanentIDLabel: work.Labels[workv1alpha2.WorkPermanentIDLabel],
	}))
	util.RecordManagedLabels(workload)
}
