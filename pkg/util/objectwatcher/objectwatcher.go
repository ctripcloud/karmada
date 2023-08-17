package objectwatcher

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/karmada-io/karmada/pkg/apis/config/v1alpha1"
	workv1alpha1 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha1"
	workv1alpha2 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha2"
	"github.com/karmada-io/karmada/pkg/resourceinterpreter"
	"github.com/karmada-io/karmada/pkg/util"
	"github.com/karmada-io/karmada/pkg/util/backoff"
	"github.com/karmada-io/karmada/pkg/util/lifted"
	"github.com/karmada-io/karmada/pkg/util/restmapper"
)

// ObjectWatcher manages operations for object dispatched to member clusters.
type ObjectWatcher interface {
	Create(clusterName string, desireObj *unstructured.Unstructured, group string) error
	Update(clusterName string, desireObj, clusterObj *unstructured.Unstructured, group string) error
	Delete(clusterName string, desireObj *unstructured.Unstructured) error
	NeedsUpdate(clusterName string, oldObj, currentObj *unstructured.Unstructured) bool
}

// ClientSetFunc is used to generate client set of member cluster
type ClientSetFunc func(c string, client client.Client) (*util.DynamicClusterClient, error)

type versionFunc func() (objectVersion string, err error)

type versionWithLock struct {
	lock    sync.RWMutex
	version string
}

type objectWatcherImpl struct {
	Lock                 sync.RWMutex
	RESTMapper           meta.RESTMapper
	KubeClientSet        client.Client
	VersionRecord        map[string]map[string]*versionWithLock
	ClusterClientSetFunc ClientSetFunc
	resourceInterpreter  resourceinterpreter.ResourceInterpreter
}

// NewObjectWatcher returns an instance of ObjectWatcher
func NewObjectWatcher(kubeClientSet client.Client, restMapper meta.RESTMapper, clusterClientSetFunc ClientSetFunc, interpreter resourceinterpreter.ResourceInterpreter) ObjectWatcher {
	return &objectWatcherImpl{
		KubeClientSet:        kubeClientSet,
		VersionRecord:        make(map[string]map[string]*versionWithLock),
		RESTMapper:           restMapper,
		ClusterClientSetFunc: clusterClientSetFunc,
		resourceInterpreter:  interpreter,
	}
}

func (o *objectWatcherImpl) Create(clusterName string, desireObj *unstructured.Unstructured, group string) error {
	dynamicClusterClient, err := o.ClusterClientSetFunc(clusterName, o.KubeClientSet)
	if err != nil {
		klog.Errorf("Failed to build dynamic cluster client for cluster %s.", clusterName)
		return err
	}

	gvr, err := restmapper.GetGroupVersionResource(o.RESTMapper, desireObj.GroupVersionKind())
	if err != nil {
		klog.Errorf("Failed to create resource(kind=%s, %s/%s) in cluster %s as mapping GVK to GVR failed: %v", desireObj.GetKind(), desireObj.GetNamespace(), desireObj.GetName(), clusterName, err)
		return err
	}

	clusterObj, err := dynamicClusterClient.DynamicClientSet.Resource(gvr).Namespace(desireObj.GetNamespace()).Create(context.TODO(), desireObj, metav1.CreateOptions{})
	if err != nil {
		klog.Errorf("[Group %s] Failed to create resource(kind=%s, %s/%s) in cluster %s, err is %v ", group, desireObj.GetKind(), desireObj.GetNamespace(), desireObj.GetName(), clusterName, err)
		return err
	}

	klog.Infof("[Group %s] Created resource(kind=%s, %s/%s) on cluster: %s", group, desireObj.GetKind(), desireObj.GetNamespace(), desireObj.GetName(), clusterName)

	// record version
	return o.recordVersionWithVersionFunc(clusterObj, dynamicClusterClient.ClusterName, group, func() (string, error) { return lifted.ObjectVersion(clusterObj), nil })
}

func (o *objectWatcherImpl) retainClusterFields(desired, observed *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	// Pass the same ResourceVersion as in the cluster object for update operation, otherwise operation will fail.
	desired.SetResourceVersion(observed.GetResourceVersion())

	// Retain finalizers since they will typically be set by
	// controllers in a member cluster.  It is still possible to set the fields
	// via overrides.
	desired.SetFinalizers(observed.GetFinalizers())

	// Retain ownerReferences since they will typically be set by controllers in a member cluster.
	desired.SetOwnerReferences(observed.GetOwnerReferences())

	// Retain annotations since they will typically be set by controllers in a member cluster
	// and be set by user in karmada-controller-plane.
	util.RetainAnnotations(desired, observed)

	// Retain labels since they will typically be set by controllers in a member cluster
	// and be set by user in karmada-controller-plane.
	util.RetainLabels(desired, observed)

	if o.resourceInterpreter.HookEnabled(desired.GroupVersionKind(), configv1alpha1.InterpreterOperationRetain) {
		return o.resourceInterpreter.Retain(desired, observed)
	}

	return desired, nil
}

func (o *objectWatcherImpl) Update(clusterName string, desireObj, clusterObj *unstructured.Unstructured, group string) error {
	updateAllowed := o.allowUpdate(clusterName, desireObj, clusterObj)
	if !updateAllowed {
		return nil
	}

	dynamicClusterClient, err := o.ClusterClientSetFunc(clusterName, o.KubeClientSet)
	if err != nil {
		klog.Errorf("Failed to build dynamic cluster client for cluster %s.", clusterName)
		return err
	}

	gvr, err := restmapper.GetGroupVersionResource(o.RESTMapper, desireObj.GroupVersionKind())
	if err != nil {
		klog.Errorf("Failed to update resource(kind=%s, %s/%s) in cluster %s as mapping GVK to GVR failed: %v", desireObj.GetKind(), desireObj.GetNamespace(), desireObj.GetName(), clusterName, err)
		return err
	}

	var errMsg string
	var desireCopy, resource *unstructured.Unstructured
	err = retry.RetryOnConflict(backoff.Retry, func() error {
		desireCopy = desireObj.DeepCopy()
		if err != nil {
			clusterObj, err = dynamicClusterClient.DynamicClientSet.Resource(gvr).Namespace(desireObj.GetNamespace()).Get(context.TODO(), desireObj.GetName(), metav1.GetOptions{})
			if err != nil {
				errMsg = fmt.Sprintf("[Group %s] Failed to get resource(kind=%s, %s/%s) in cluster %s, err is %v ",
					group, desireObj.GetKind(), desireObj.GetNamespace(), desireObj.GetName(), clusterName, err)
				return err
			}
		}

		desireCopy, err = o.retainClusterFields(desireCopy, clusterObj)
		if err != nil {
			errMsg = fmt.Sprintf("[Group %s] Failed to retain fields for resource(kind=%s, %s/%s) in cluster %s: %v",
				group, clusterObj.GetKind(), clusterObj.GetNamespace(), clusterObj.GetName(), clusterName, err)
			return err
		}

		versionFuncWithUpdate := func() (string, error) {
			resource, err = dynamicClusterClient.DynamicClientSet.Resource(gvr).Namespace(desireObj.GetNamespace()).Update(context.TODO(), desireCopy, metav1.UpdateOptions{})
			if err != nil {
				return "", err
			}

			return lifted.ObjectVersion(resource), nil
		}
		err = o.recordVersionWithVersionFunc(desireCopy, clusterName, group, versionFuncWithUpdate)
		if err == nil {
			return nil
		}

		errMsg = fmt.Sprintf("[Group %s] Failed to update resource(kind=%s, %s/%s) in cluster %s, err is %v ",
			group, desireObj.GetKind(), desireObj.GetNamespace(), desireObj.GetName(), clusterName, err)
		return err
	})

	if err != nil {
		klog.Errorf(errMsg)
		return err
	}

	klog.Infof("[Group %s] Updated resource(kind=%s, %s/%s) on cluster: %s, ResourceVersion: OLD: %s, CUR: %s; Diff: %s",
		group, desireObj.GetKind(), desireObj.GetNamespace(), desireObj.GetName(), clusterName, desireObj.GetResourceVersion(), resource.GetResourceVersion(), util.TellDiffForObjects(desireObj, resource))

	return nil
}

func (o *objectWatcherImpl) Delete(clusterName string, desireObj *unstructured.Unstructured) error {
	dynamicClusterClient, err := o.ClusterClientSetFunc(clusterName, o.KubeClientSet)
	if err != nil {
		klog.Errorf("Failed to build dynamic cluster client for cluster %s.", clusterName)
		return err
	}

	gvr, err := restmapper.GetGroupVersionResource(o.RESTMapper, desireObj.GroupVersionKind())
	if err != nil {
		klog.Errorf("Failed to delete resource(kind=%s, %s/%s) in cluster %s as mapping GVK to GVR failed: %v", desireObj.GetKind(), desireObj.GetNamespace(), desireObj.GetName(), clusterName, err)
		return err
	}

	// Set deletion strategy to background explicitly even though it's the default strategy for most of the resources.
	// The reason for this is to fix the exception case that Kubernetes does on Job(batch/v1).
	// In kubernetes, the Job's default deletion strategy is "Orphan", that will cause the "Pods" created by "Job"
	// still exist after "Job" has been deleted.
	// Refer to https://github.com/karmada-io/karmada/issues/969 for more details.
	deleteBackground := metav1.DeletePropagationBackground
	deleteOption := metav1.DeleteOptions{
		PropagationPolicy: &deleteBackground,
	}

	err = dynamicClusterClient.DynamicClientSet.Resource(gvr).Namespace(desireObj.GetNamespace()).Delete(context.TODO(), desireObj.GetName(), deleteOption)
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("Failed to delete resource %v in cluster %s, err is %v ", desireObj.GetName(), clusterName, err)
		return err
	}
	klog.Infof("Deleted resource(kind=%s, %s/%s) on cluster: %s", desireObj.GetKind(), desireObj.GetNamespace(), desireObj.GetName(), clusterName)

	o.deleteVersionRecord(desireObj, dynamicClusterClient.ClusterName)

	return nil
}

func (o *objectWatcherImpl) genObjectKey(obj *unstructured.Unstructured) string {
	return obj.GroupVersionKind().String() + "/" + obj.GetNamespace() + "/" + obj.GetName()
}

// recordVersion will add or update resource version records with the version returned by versionFunc
func (o *objectWatcherImpl) recordVersionWithVersionFunc(obj *unstructured.Unstructured, clusterName, group string, fn versionFunc) error {
	objectKey := o.genObjectKey(obj)
	return o.addOrUpdateVersionRecordWithVersionFunc(clusterName, objectKey, group, fn)
}

// getVersionRecord will return the recorded version of given resource(if exist)
func (o *objectWatcherImpl) getVersionRecord(clusterName, resourceName string) (string, bool) {
	versionLock, exist := o.getVersionWithLockRecord(clusterName, resourceName)
	if !exist {
		return "", false
	}

	versionLock.lock.RLock()
	defer versionLock.lock.RUnlock()

	return versionLock.version, true
}

// getVersionRecordWithLock will return the recorded versionWithLock of given resource(if exist)
func (o *objectWatcherImpl) getVersionWithLockRecord(clusterName, resourceName string) (*versionWithLock, bool) {
	o.Lock.RLock()
	defer o.Lock.RUnlock()

	versionLock, exist := o.VersionRecord[clusterName][resourceName]
	return versionLock, exist
}

// newVersionWithLockRecord will add new versionWithLock record of given resource
func (o *objectWatcherImpl) newVersionWithLockRecord(clusterName, resourceName string) *versionWithLock {
	o.Lock.Lock()
	defer o.Lock.Unlock()

	v, exist := o.VersionRecord[clusterName][resourceName]
	if exist {
		return v
	}

	v = &versionWithLock{}
	if o.VersionRecord[clusterName] == nil {
		o.VersionRecord[clusterName] = map[string]*versionWithLock{}
	}

	o.VersionRecord[clusterName][resourceName] = v

	return v
}

// addOrUpdateVersionRecordWithVersionFunc will add or update the recorded version of given resource with version returned by versionFunc
func (o *objectWatcherImpl) addOrUpdateVersionRecordWithVersionFunc(clusterName, resourceName, group string, fn versionFunc) error {
	versionLock, exist := o.getVersionWithLockRecord(clusterName, resourceName)

	if !exist {
		versionLock = o.newVersionWithLockRecord(clusterName, resourceName)
	}

	versionLock.lock.Lock()
	defer versionLock.lock.Unlock()

	version, err := fn()
	if err != nil {
		return err
	}

	klog.Infof("[Group %s] Update version record in objectWatcher from %s to %s for %s/%s", group, versionLock.version, version, clusterName, resourceName)
	versionLock.version = version

	return nil
}

// deleteVersionRecord will delete the recorded version of given resource
func (o *objectWatcherImpl) deleteVersionRecord(obj *unstructured.Unstructured, clusterName string) {
	objectKey := o.genObjectKey(obj)

	o.Lock.Lock()
	defer o.Lock.Unlock()
	delete(o.VersionRecord[clusterName], objectKey)
}

func (o *objectWatcherImpl) NeedsUpdate(clusterName string, desiredObj, clusterObj *unstructured.Unstructured) (need bool) {
	// just for log
	objectKey := o.genObjectKey(clusterObj)
	recordedVersion, _ := o.getVersionRecord(clusterName, objectKey)
	clusterVersion := lifted.ObjectVersion(clusterObj)
	genRes, rvSame, _ := lifted.CompareObjectVersion(clusterVersion, recordedVersion)
	genStr := "<nil>"
	if genRes != nil {
		genStr = fmt.Sprint(*genRes)
	}

	need = o.needsUpdate(clusterName, desiredObj, clusterObj)
	klog.Infof("NeedUpdate check resource(kind=%s, %s/%s) need: %v, desiredVersion: %v, clusterVersion: %v, clusterRV: %v, clusterGen: %v, genRes: %v, rvSame: %v",
		desiredObj.GetKind(), desiredObj.GetNamespace(), desiredObj.GetName(),
		need, recordedVersion, clusterVersion, clusterObj.GetResourceVersion(), clusterObj.GetGeneration(), genStr, rvSame)
	return need
}

func (o *objectWatcherImpl) needsUpdate(clusterName string, desiredObj, clusterObj *unstructured.Unstructured) bool {
	// get resource version
	objectKey := o.genObjectKey(clusterObj)
	recordedVersion, exist := o.getVersionRecord(clusterName, objectKey)
	if !exist {
		return true
	}

	return ObjectNeedsUpdate(desiredObj, clusterObj, recordedVersion)
}

/*
// dynamicContentCheck get the newest object, compares non metadata/status fields with desiredObj

	func (o *objectWatcherImpl) dynamicContentCheck(clusterName string, desiredObj, clusterObj *unstructured.Unstructured) (needUpdate bool, newestClusterObj *unstructured.Unstructured, err error) {
		gvr, err := restmapper.GetGroupVersionResource(o.RESTMapper, clusterObj.GetObjectKind().GroupVersionKind())
		if err != nil {
			return false, clusterObj, err
		}
		dynamicClient := o.InformerManager.GetSingleClusterManager(clusterName).GetClient()
		var c dynamic.ResourceInterface
		if clusterObj.GetNamespace() != "" {
			c = dynamicClient.Resource(gvr).Namespace(clusterObj.GetNamespace())
		} else {
			c = dynamicClient.Resource(gvr)
		}
		newestClusterObj, err = c.Get(context.TODO(), clusterObj.GetName(), metav1.GetOptions{})
		if err != nil {
			return false, clusterObj, err
		}

		// if we never updated this cluster object or cluster object got delete-recreate, we needUpdate a content check
		desiredContent := copyNonMetadataAndStatus(desiredObj.Object)
		clusterContent := copyNonMetadataAndStatus(newestClusterObj.Object)

		if !apiequality.Semantic.DeepEqual(desiredContent, clusterContent) {
			needUpdate = true
		}
		return needUpdate, newestClusterObj, nil
	}

	func copyNonMetadataAndStatus(original map[string]interface{}) map[string]interface{} {
		ret := make(map[string]interface{})
		for key, val := range original {
			if key == "metadata" || key == "status" {
				continue
			}
			ret[key] = val
		}
		return ret
	}
*/
func (o *objectWatcherImpl) allowUpdate(clusterName string, desiredObj, clusterObj *unstructured.Unstructured) bool {
	// If the existing resource is managed by Karmada, then the updating is allowed.
	if util.GetLabelValue(desiredObj.GetLabels(), workv1alpha1.WorkNameLabel) == util.GetLabelValue(clusterObj.GetLabels(), workv1alpha1.WorkNameLabel) &&
		util.GetLabelValue(desiredObj.GetLabels(), workv1alpha1.WorkNamespaceLabel) == util.GetLabelValue(clusterObj.GetLabels(), workv1alpha1.WorkNamespaceLabel) {
		return true
	}

	// This happens when promoting workload to the Karmada control plane
	conflictResolution := util.GetAnnotationValue(desiredObj.GetAnnotations(), workv1alpha2.ResourceConflictResolutionAnnotation)
	if conflictResolution == workv1alpha2.ResourceConflictResolutionOverwrite {
		return true
	}

	// The existing resource is not managed by Karmada, and no conflict resolution found, avoid updating the existing resource by default.
	klog.Warningf("resource(kind=%s, %s/%s) already exist in cluster %v and the %s strategy value is empty, karmada will not manage this resource",
		desiredObj.GetKind(), desiredObj.GetNamespace(), desiredObj.GetName(), clusterName, workv1alpha2.ResourceConflictResolutionAnnotation,
	)
	return false
}

// ObjectNeedsUpdate determines whether the 2 objects provided cluster
// object needs to be updated according to the desired object and the
// recorded version.
func ObjectNeedsUpdate(desiredObj, clusterObj *unstructured.Unstructured, recordedVersion string) bool {
	targetVersion := lifted.ObjectVersion(clusterObj)

	if recordedVersion != targetVersion {
		return true
	}

	// If versions match and the version is sourced from the
	// generation field, a further check of metadata equivalency is
	// required.
	return strings.HasPrefix(targetVersion, "gen:") && !objectMetaObjEquivalent(desiredObj, clusterObj)
}

func objectMetaObjEquivalent(desired, cluster *unstructured.Unstructured) bool {
	if desired.GetName() != cluster.GetName() {
		return false
	}
	if desired.GetNamespace() != cluster.GetNamespace() {
		return false
	}
	desiredCopy := copyOnlyLabelsAndAnnotations(desired)
	clusterCopy := copyOnlyLabelsAndAnnotations(cluster)

	// Retain annotations since they will typically be set by controllers in a member cluster
	// and be set by user in karmada-controller-plane.
	util.RetainAnnotations(desiredCopy, clusterCopy)

	// Retain labels since they will typically be set by controllers in a member cluster
	// and be set by user in karmada-controller-plane.
	util.RetainLabels(desiredCopy, clusterCopy)

	return reflect.DeepEqual(desiredCopy.GetLabels(), cluster.GetLabels()) &&
		reflect.DeepEqual(desiredCopy.GetAnnotations(), cluster.GetAnnotations())
}

func copyOnlyLabelsAndAnnotations(a *unstructured.Unstructured) *unstructured.Unstructured {
	ret := &unstructured.Unstructured{}
	ret.SetAnnotations(a.GetAnnotations())
	ret.SetLabels(a.GetLabels())
	return ret
}
