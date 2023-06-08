package util

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// MergeOwnerReferences merges exist and new owner-references.
// If the new ones set a controller, the existed controllers would be set to false.
// And if new ones had been set several controllers, the last one would be the active one.
func MergeOwnerReferences(existOwnerRef, newOwnerRef []metav1.OwnerReference) []metav1.OwnerReference {
	if len(existOwnerRef) == 0 {
		return newOwnerRef
	}

	for _, ref := range newOwnerRef {
		existOwnerRef = replaceOrAppendOwnerRef(existOwnerRef, ref)
	}

	return existOwnerRef
}

func replaceOrAppendOwnerRef(ownerReferences []metav1.OwnerReference, target metav1.OwnerReference) []metav1.OwnerReference {
	fi := -1
	for index, r := range ownerReferences {
		if referSameObject(r, target) {
			fi = index
			continue
		}

		// If target owner is controller and other existed owner has set controller, replace the existed ones.
		if pointer.BoolDeref(target.Controller, false) && pointer.BoolDeref(r.Controller, false) {
			ownerReferences[index].Controller = nil
		}
	}

	if fi < 0 {
		ownerReferences = append(ownerReferences, target)
	} else {
		ownerReferences[fi] = target
	}

	return ownerReferences
}

func referSameObject(a, b metav1.OwnerReference) bool {
	return a.APIVersion == b.APIVersion && a.Kind == b.Kind && a.Name == b.Name
}
