/*
Copyright 2016 The Kubernetes Authors.

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

// This code is lifted from the kubefed codebase. It's a list of functions to determine whether the provided cluster
// object needs to be updated according to the desired object and the recorded version.
// For reference:
// https://github.com/kubernetes-sigs/kubefed/blob/master/pkg/controller/util/propagatedversion.go#L30-L59
// https://github.com/kubernetes-sigs/kubefed/blob/master/pkg/controller/util/meta.go#L63-L80

package lifted

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	generationPrefix      = "gen:"
	resourceVersionPrefix = "rv:"
)

// +lifted:source=https://github.com/kubernetes-sigs/kubefed/blob/master/pkg/controller/util/propagatedversion.go#L35-L43

// ObjectVersion retrieves the field type-prefixed value used for
// determining currency of the given cluster object.
func ObjectVersion(obj *unstructured.Unstructured) string {
	generation := obj.GetGeneration()
	if generation != 0 {
		return fmt.Sprintf("%s%d", generationPrefix, generation)
	}
	return fmt.Sprintf("%s%s", resourceVersionPrefix, obj.GetResourceVersion())
}

// +lifted:source=https://github.com/kubernetes-sigs/kubefed/blob/master/pkg/controller/util/propagatedversion.go#L45-L59

// ObjectNeedsUpdate determines whether the 2 objects provided cluster
// object needs to be updated according to the old object and the
// recorded version.
func ObjectNeedsUpdate(oldObj, currentObj *unstructured.Unstructured, recordedVersion string) bool {
	targetVersion := ObjectVersion(currentObj)

	if recordedVersion != targetVersion {
		return true
	}

	// If versions match and the version is sourced from the
	// generation field, a further check of metadata equivalency is
	// required.
	return strings.HasPrefix(targetVersion, generationPrefix) && !objectMetaObjEquivalent(oldObj, currentObj)
}

// ObjectMetaNeedsUpdate determines whether the 2 objects provided cluster
// object's metadata needs to be updated according to the desired object
// and the clusterObj.
func ObjectMetaNeedsUpdate(desiredObj, clusterObj *unstructured.Unstructured) bool {
	targetVersion := ObjectVersion(clusterObj)

	// If versions match and the version is sourced from the
	// generation field, a further check of metadata equivalency is
	// required.
	return strings.HasPrefix(targetVersion, generationPrefix) && !objectMetaObjEquivalent(desiredObj, clusterObj)
}

// CompareObjectVersion compares two non nil objects' generation
// or resourceVersion, return error if objects got invalid version.
func CompareObjectVersion(a, b string) (genRes *int, rvSame bool, err error) {
	ag, arv, aPrefix, err := parseObjectVersion(a)
	if err != nil {
		return nil, false, err
	}
	bg, brv, bPrefix, err := parseObjectVersion(b)
	if err != nil {
		return nil, false, err
	}
	if aPrefix == generationPrefix && bPrefix == generationPrefix {
		res := 0
		if ag-bg < 0 {
			res = -1
		} else if ag-bg > 0 {
			res = 1
		}
		return &res, false, nil
	}
	return nil, arv == brv, nil
}

func parseObjectVersion(s string) (int64, string, string, error) {
	if strings.HasPrefix(s, generationPrefix) {
		genStr := strings.TrimPrefix(s, generationPrefix)
		gen, err := strconv.ParseInt(genStr, 10, 64)
		if err != nil {
			return 0, "", "", err
		}
		if gen == 0 {
			return 0, "", "", fmt.Errorf("generation should not be 0: %s", s)
		}
		return gen, "", generationPrefix, nil
	}

	if strings.HasPrefix(s, resourceVersionPrefix) {
		rvStr := strings.TrimPrefix(s, resourceVersionPrefix)
		_, err := strconv.ParseUint(rvStr, 10, 64)
		if err != nil {
			return 0, "", "", err
		}
		return 0, rvStr, resourceVersionPrefix, nil
	}

	return 0, "", "", fmt.Errorf("unknown object version: %s", s)
}

// +lifted:source=https://github.com/kubernetes-sigs/kubefed/blob/master/pkg/controller/util/meta.go#L63-L80
// +lifted:changed

// objectMetaObjEquivalent checks if cluster-independent, user provided data in two given ObjectMeta are equal. If in
// the future the ObjectMeta structure is expanded then any field that is not populated
// by the api server should be included here.
func objectMetaObjEquivalent(a, b metav1.Object) bool {
	if a.GetName() != b.GetName() {
		return false
	}
	if a.GetNamespace() != b.GetNamespace() {
		return false
	}
	aLabels := a.GetLabels()
	bLabels := b.GetLabels()
	if !reflect.DeepEqual(aLabels, bLabels) && (len(aLabels) != 0 || len(bLabels) != 0) {
		return false
	}

	aAnnotations := a.GetAnnotations()
	bAnnotations := b.GetAnnotations()
	if !reflect.DeepEqual(aAnnotations, bAnnotations) && (len(aAnnotations) != 0 || len(bAnnotations) != 0) {
		return false
	}
	return true
}
