package util

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/karmada-io/karmada/pkg/util/fedinformer/keys"
	"github.com/karmada-io/karmada/pkg/util/gclient"
)

var mu = &sync.Mutex{}
var scheme = gclient.NewSchema()

var stepInfo = map[string]uint64{}

// AddStepForObject add step for object and return the key.
func AddStepForObject(controller string, object runtime.Object) string {
	return addStepByKey(generateKeyForObject(controller, object))
}

// AddStepForFedKey add step for keys.FederatedKey and return the key.
func AddStepForFedKey(controller string, fedKey keys.FederatedKey) string {
	return addStepByKey(generateKeyForFedKey(controller, fedKey))
}

// AddStepForRequestObject add step for object and controllerruntime.Request and return the key.
func AddStepForRequestObject(controller string, req controllerruntime.Request, obj runtime.Object) string {
	return addStepByKey(generateKeyForRequest(controller, req, obj))
}

func addStepByKey(key string) string {
	mu.Lock()
	defer mu.Unlock()

	i, ok := stepInfo[key]
	if !ok {
		i = uint64(0)
		stepInfo[key] = i
		return fmt.Sprintf("<%d>: %s", i, key)
	}

	stepInfo[key] = i + 1
	return fmt.Sprintf("<%d>: %s", i+1, key)
}

func generateKeyForFedKey(controller string, fedKey keys.FederatedKey) string {
	key := fedKey.Cluster
	if key != "" {
		key = fmt.Sprintf("%s/", key)
	}

	return fmt.Sprintf("%s/%s/%s/%s/%s", controller, key, fedKey.GroupVersion().String(), fedKey.Kind, fedKey.NamespaceKey())
}

func generateKeyForObject(controller string, object runtime.Object) string {
	return generateKeyByControllerGVKAndNamespacedName(controller, generateGVKKey(object.GetObjectKind().GroupVersionKind()), generateNameKey(object))
}

func generateKeyByControllerGVKAndNamespacedName(controller, gvk, namespacedName string) string {
	return fmt.Sprintf("%s/%s/%s", controller, gvk, namespacedName)
}

func generateGVKKey(gvk schema.GroupVersionKind) string {
	return fmt.Sprintf("%s/%s", gvk.GroupVersion().String(), gvk.Kind)
}

func generateNameKey(object runtime.Object) string {
	m, _ := meta.Accessor(object)

	ns := m.GetNamespace()
	name := m.GetName()

	return generateNamespacedNameKey(ns, name)
}

func generateNamespacedNameKey(namespace, name string) string {
	if name == "" {
		return "nil name"
	}

	if namespace != "" {
		return fmt.Sprintf("%s/%s", namespace, name)
	}

	return name
}

func generateKeyForRequest(controller string, req controllerruntime.Request, obj runtime.Object) string {
	gvk, _ := apiutil.GVKForObject(obj, scheme)

	return generateKeyByControllerGVKAndNamespacedName(controller, generateGVKKey(gvk), generateNamespacedNameKey(req.Namespace, req.Name))
}
