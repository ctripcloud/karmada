package fedinformer

import (
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/karmada-io/karmada/pkg/util"
)

// NewHandlerOnAllEvents builds a ResourceEventHandler that the function 'fn' will be called on all events(add/update/delete).
func NewHandlerOnAllEvents(fn func(runtime.Object)) cache.ResourceEventHandler {
	return &cache.ResourceEventHandlerFuncs{
		AddFunc: func(cur interface{}) {
			curObj := cur.(runtime.Object)
			m, _ := meta.Accessor(curObj)
			klog.Infof("Enqueue obj(%s, %s/%s) for add event.", curObj.GetObjectKind().GroupVersionKind().String(), m.GetNamespace(), m.GetName())
			fn(curObj)
		},
		UpdateFunc: func(old, cur interface{}) {
			curObj := cur.(runtime.Object)
			oldObj := old.(runtime.Object)
			if !reflect.DeepEqual(old, cur) {
				newM, _ := meta.Accessor(curObj)
				oldM, _ := meta.Accessor(oldObj)
				klog.Infof("Enqueue obj(%s, %s/%s) for update event. ResourceVersion: OLD: %s, NEW: %s. Diff: %s.",
					curObj.GetObjectKind().GroupVersionKind().String(), newM.GetNamespace(), newM.GetName(), oldM.GetResourceVersion(), newM.GetResourceVersion(), util.TellDiffForObjects(old, cur))
				fn(curObj)
			}
		},
		DeleteFunc: func(old interface{}) {
			if deleted, ok := old.(cache.DeletedFinalStateUnknown); ok {
				// This object might be stale but ok for our current usage.
				old = deleted.Obj
				if old == nil {
					return
				}
			}
			oldObj := old.(runtime.Object)
			fn(oldObj)
		},
	}
}

// NewHandlerOnEvents builds a ResourceEventHandler.
func NewHandlerOnEvents(addFunc func(obj interface{}), updateFunc func(oldObj, newObj interface{}), deleteFunc func(obj interface{})) cache.ResourceEventHandler {
	return &cache.ResourceEventHandlerFuncs{
		AddFunc:    addFunc,
		UpdateFunc: updateFunc,
		DeleteFunc: deleteFunc,
	}
}

// NewFilteringHandlerOnAllEvents builds a FilteringResourceEventHandler applies the provided filter to all events
// coming in, ensuring the appropriate nested handler method is invoked.
//
// Note: An object that starts passing the filter after an update is considered an add, and
// an object that stops passing the filter after an update is considered a delete.
// Like the handlers, the filter MUST NOT modify the objects it is given.
func NewFilteringHandlerOnAllEvents(filterFunc func(obj interface{}) bool, addFunc func(obj interface{}),
	updateFunc func(oldObj, newObj interface{}), deleteFunc func(obj interface{})) cache.ResourceEventHandler {
	return &cache.FilteringResourceEventHandler{
		FilterFunc: filterFunc,
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    addFunc,
			UpdateFunc: updateFunc,
			DeleteFunc: deleteFunc,
		},
	}
}
