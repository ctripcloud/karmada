package resourceinterpreter

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	configv1alpha1 "github.com/karmada-io/karmada/pkg/apis/config/v1alpha1"
	workv1alpha2 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha2"
)

var _ ResourceInterpreter = &FakeInterpreter{}

type (
	getReplicasFunc     = func(*unstructured.Unstructured) (int32, *workv1alpha2.ReplicaRequirements, *[]workv1alpha2.TargetCluster, error)
	reviseReplicaFunc   = func(*unstructured.Unstructured, int64, string) (*unstructured.Unstructured, error)
	retainFunc          = func(*unstructured.Unstructured, *unstructured.Unstructured) (*unstructured.Unstructured, error)
	aggregateStatusFunc = func(*unstructured.Unstructured, []workv1alpha2.AggregatedStatusItem) (*unstructured.Unstructured, error)
	getDependenciesFunc = func(*unstructured.Unstructured) ([]configv1alpha1.DependentObjectReference, error)
	reflectStatusFunc   = func(*unstructured.Unstructured) (*runtime.RawExtension, error)
	interpretHealthFunc = func(*unstructured.Unstructured) (bool, error)
)

// FakeInterpreter implements a ResourceInterpreter.
type FakeInterpreter struct {
	getReplicasFunc     map[schema.GroupVersionKind]getReplicasFunc
	reviseReplicaFunc   map[schema.GroupVersionKind]reviseReplicaFunc
	retainFunc          map[schema.GroupVersionKind]retainFunc
	aggregateStatusFunc map[schema.GroupVersionKind]aggregateStatusFunc
	getDependenciesFunc map[schema.GroupVersionKind]getDependenciesFunc
	reflectStatusFunc   map[schema.GroupVersionKind]reflectStatusFunc
	interpretHealthFunc map[schema.GroupVersionKind]interpretHealthFunc
}

// Start always returns nil, just for implementing ResourceInterpreter.
func (f *FakeInterpreter) Start(_ context.Context) (err error) {
	return nil
}

// HookEnabled returns if an interpreter function exists for input GVK and InterpreterOperation.
func (f *FakeInterpreter) HookEnabled(objGVK schema.GroupVersionKind, operationType configv1alpha1.InterpreterOperation) bool {
	var exist bool
	switch operationType {
	case configv1alpha1.InterpreterOperationInterpretReplica:
		_, exist = f.getReplicasFunc[objGVK]
	case configv1alpha1.InterpreterOperationReviseReplica:
		_, exist = f.reviseReplicaFunc[objGVK]
	case configv1alpha1.InterpreterOperationRetain:
		_, exist = f.retainFunc[objGVK]
	case configv1alpha1.InterpreterOperationAggregateStatus:
		_, exist = f.aggregateStatusFunc[objGVK]
	case configv1alpha1.InterpreterOperationInterpretDependency:
		_, exist = f.getDependenciesFunc[objGVK]
	case configv1alpha1.InterpreterOperationInterpretStatus:
		_, exist = f.reflectStatusFunc[objGVK]
	case configv1alpha1.InterpreterOperationInterpretHealth:
		_, exist = f.interpretHealthFunc[objGVK]
	default:
		exist = false
	}

	return exist
}

// GetReplicas calls recorded getReplicasFunc for input object.
func (f *FakeInterpreter) GetReplicas(object *unstructured.Unstructured) (replica int32, requires *workv1alpha2.ReplicaRequirements, clusters *[]workv1alpha2.TargetCluster, err error) {
	return f.getReplicasFunc[object.GetObjectKind().GroupVersionKind()](object)
}

// ReviseReplica calls recorded reviseReplicaFunc for input object.
func (f *FakeInterpreter) ReviseReplica(object *unstructured.Unstructured, replica int64, cluster string) (*unstructured.Unstructured, error) {
	return f.reviseReplicaFunc[object.GetObjectKind().GroupVersionKind()](object, replica, cluster)
}

// Retain calls recorded retainFunc for input object.
func (f *FakeInterpreter) Retain(desired *unstructured.Unstructured, observed *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return f.retainFunc[observed.GetObjectKind().GroupVersionKind()](desired, observed)
}

// AggregateStatus calls recorded aggregateStatusFunc for input object.
func (f *FakeInterpreter) AggregateStatus(object *unstructured.Unstructured, aggregatedStatusItems []workv1alpha2.AggregatedStatusItem) (*unstructured.Unstructured, error) {
	return f.aggregateStatusFunc[object.GetObjectKind().GroupVersionKind()](object, aggregatedStatusItems)
}

// GetDependencies calls recorded getDependenciesFunc for input object.
func (f *FakeInterpreter) GetDependencies(object *unstructured.Unstructured) (dependencies []configv1alpha1.DependentObjectReference, err error) {
	return f.getDependenciesFunc[object.GetObjectKind().GroupVersionKind()](object)
}

// ReflectStatus calls recorded reflectStatusFunc for input object.
func (f *FakeInterpreter) ReflectStatus(object *unstructured.Unstructured) (status *runtime.RawExtension, err error) {
	return f.reflectStatusFunc[object.GetObjectKind().GroupVersionKind()](object)
}

// InterpretHealth calls recorded interpretHealthFunc for input object.
func (f *FakeInterpreter) InterpretHealth(object *unstructured.Unstructured) (healthy bool, err error) {
	return f.interpretHealthFunc[object.GetObjectKind().GroupVersionKind()](object)
}

// NewFakeResourceInterpreter returns a new empty FakeInterpreter.
func NewFakeResourceInterpreter() *FakeInterpreter {
	return &FakeInterpreter{}
}

// WithGetReplicas updates getReplciasFunc for input GVK.
func (f *FakeInterpreter) WithGetReplicas(objGVK schema.GroupVersionKind, iFunc getReplicasFunc) *FakeInterpreter {
	if f.getReplicasFunc == nil {
		f.getReplicasFunc = make(map[schema.GroupVersionKind]getReplicasFunc)
	}
	f.getReplicasFunc[objGVK] = iFunc

	return f
}

// WithReviseReplica updates reviseReplicaFunc for input GVK.
func (f *FakeInterpreter) WithReviseReplica(objGVK schema.GroupVersionKind, iFunc reviseReplicaFunc) *FakeInterpreter {
	if f.reviseReplicaFunc == nil {
		f.reviseReplicaFunc = make(map[schema.GroupVersionKind]reviseReplicaFunc)
	}
	f.reviseReplicaFunc[objGVK] = iFunc

	return f
}

// WithRetain updates retainFunc for input GVK.
func (f *FakeInterpreter) WithRetain(objGVK schema.GroupVersionKind, iFunc retainFunc) *FakeInterpreter {
	if f.retainFunc == nil {
		f.retainFunc = make(map[schema.GroupVersionKind]retainFunc)
	}
	f.retainFunc[objGVK] = iFunc

	return f
}

// WithAggregateStatus updates aggregateStatusFunc for input GVK.
func (f *FakeInterpreter) WithAggregateStatus(objGVK schema.GroupVersionKind, iFunc aggregateStatusFunc) *FakeInterpreter {
	if f.aggregateStatusFunc == nil {
		f.aggregateStatusFunc = make(map[schema.GroupVersionKind]aggregateStatusFunc)
	}
	f.aggregateStatusFunc[objGVK] = iFunc

	return f
}

// WithGetDependencies updates getDependenciesFunc for input GVK.
func (f *FakeInterpreter) WithGetDependencies(objGVK schema.GroupVersionKind, iFunc getDependenciesFunc) *FakeInterpreter {
	if f.getDependenciesFunc == nil {
		f.getDependenciesFunc = make(map[schema.GroupVersionKind]getDependenciesFunc)
	}
	f.getDependenciesFunc[objGVK] = iFunc

	return f
}

// WithReflectStatus updates reflectStatusFunc for input GVK.
func (f *FakeInterpreter) WithReflectStatus(objGVK schema.GroupVersionKind, iFunc reflectStatusFunc) *FakeInterpreter {
	if f.reflectStatusFunc == nil {
		f.reflectStatusFunc = make(map[schema.GroupVersionKind]reflectStatusFunc)
	}
	f.reflectStatusFunc[objGVK] = iFunc

	return f
}

// WithInterpretHealth updates interpretHealthFunc for input GVK.
func (f *FakeInterpreter) WithInterpretHealth(objGVK schema.GroupVersionKind, iFunc interpretHealthFunc) *FakeInterpreter {
	if f.interpretHealthFunc == nil {
		f.interpretHealthFunc = make(map[schema.GroupVersionKind]interpretHealthFunc)
	}
	f.interpretHealthFunc[objGVK] = iFunc

	return f
}
