package detector

import (
	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	"github.com/karmada-io/karmada/pkg/util"
	"github.com/karmada-io/karmada/pkg/util/fedinformer/keys"
)

// ClusterWideKeyFunc generates a ClusterWideKey for object.
func ClusterWideKeyFunc(obj interface{}) (util.QueueKey, error) {
	return keys.ClusterWideKeyFunc(obj)
}

func cleanUpPolicyLabels(labels map[string]string) map[string]string {
	delete(labels, policyv1alpha1.PropagationPolicyNameLabel)
	delete(labels, policyv1alpha1.PropagationPolicyNamespaceLabel)
	delete(labels, policyv1alpha1.ClusterPropagationPolicyLabel)

	return labels
}
