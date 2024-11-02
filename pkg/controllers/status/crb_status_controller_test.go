/*
Copyright 2023 The Karmada Authors.

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

package status

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	workv1alpha1 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha1"
	workv1alpha2 "github.com/karmada-io/karmada/pkg/apis/work/v1alpha2"
	"github.com/karmada-io/karmada/pkg/resourceinterpreter/default/native"
	"github.com/karmada-io/karmada/pkg/util/fedinformer/genericmanager"
	"github.com/karmada-io/karmada/pkg/util/gclient"
	utilhelper "github.com/karmada-io/karmada/pkg/util/helper"
)

func generateCRBStatusController() *CRBStatusController {
	stopCh := make(chan struct{})
	defer close(stopCh)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme.Scheme,
		&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "cr1"}})
	m := genericmanager.NewSingleClusterInformerManager(dynamicClient, 0, stopCh)
	m.Lister(rbacv1.SchemeGroupVersion.WithResource("clusterroles"))
	m.Start()
	m.WaitForCacheSync()

	c := &CRBStatusController{
		Client: fake.NewClientBuilder().WithScheme(gclient.NewSchema()).WithIndex(
			&workv1alpha1.Work{},
			workv1alpha2.ClusterResourceBindingPermanentIDLabel,
			utilhelper.GenOneLabelEqualIndexerFunc(workv1alpha2.ClusterResourceBindingPermanentIDLabel),
		).Build(),
		DynamicClient:   dynamicClient,
		InformerManager: m,
		RESTMapper: func() meta.RESTMapper {
			m := meta.NewDefaultRESTMapper([]schema.GroupVersion{rbacv1.SchemeGroupVersion})
			m.Add(rbacv1.SchemeGroupVersion.WithKind("ClusterRole"), meta.RESTScopeRoot)
			return m
		}(),
		EventRecorder: &record.FakeRecorder{},
	}
	return c
}

func TestCRBStatusController_Reconcile(t *testing.T) {
	preTime := metav1.Date(2023, 0, 0, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		binding     *workv1alpha2.ClusterResourceBinding
		expectRes   controllerruntime.Result
		expectError bool
	}{
		{
			name: "failed in syncBindingStatus",
			binding: &workv1alpha2.ClusterResourceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "binding",
				},
				Spec: workv1alpha2.ResourceBindingSpec{
					Resource: workv1alpha2.ObjectReference{
						APIVersion: "rbac.authorization.k8s.io/v1",
						Kind:       "ClusterRole",
						Name:       "cr",
					},
				},
			},
			expectRes:   controllerruntime.Result{},
			expectError: false,
		},
		{
			name:        "binding not found in client",
			expectRes:   controllerruntime.Result{},
			expectError: false,
		},
		{
			name: "failed in syncBindingStatus",
			binding: &workv1alpha2.ClusterResourceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "binding",
					// finalizers field is required when deletionTimestamp is defined, otherwise will encounter the
					// error: `refusing to create obj binding with metadata.deletionTimestamp but no finalizers`.
					Finalizers:        []string{"test"},
					DeletionTimestamp: &preTime,
				},
				Spec: workv1alpha2.ResourceBindingSpec{
					Resource: workv1alpha2.ObjectReference{
						APIVersion: "rbac.authorization.k8s.io/v1",
						Kind:       "ClusterRole",
						Name:       "cr",
					},
				},
			},
			expectRes:   controllerruntime.Result{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := generateCRBStatusController()
			c.ResourceInterpreter = FakeResourceInterpreter{DefaultInterpreter: native.NewDefaultInterpreter()}

			// Prepare req
			req := controllerruntime.Request{
				NamespacedName: types.NamespacedName{
					Name: "binding",
				},
			}

			// Prepare binding and create it in client
			if tt.binding != nil {
				c.Client = fake.NewClientBuilder().WithScheme(gclient.NewSchema()).WithObjects(tt.binding).WithStatusSubresource(tt.binding).
					WithIndex(&workv1alpha1.Work{}, workv1alpha2.ClusterResourceBindingPermanentIDLabel, utilhelper.GenOneLabelEqualIndexerFunc(workv1alpha2.ClusterResourceBindingPermanentIDLabel)).
					Build()
			}

			res, err := c.Reconcile(context.Background(), req)
			assert.Equal(t, tt.expectRes, res)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCRBStatusController_syncBindingStatus(t *testing.T) {
	tests := []struct {
		name                           string
		resource                       workv1alpha2.ObjectReference
		clusterRoleNameInDynamicClient string
		resourceExistInClient          bool
		expectedError                  bool
	}{
		{
			name: "failed in FetchResourceTemplate, err is NotFound",
			resource: workv1alpha2.ObjectReference{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
				Name:       "cr",
			},
			clusterRoleNameInDynamicClient: "cr1",
			resourceExistInClient:          true,
			expectedError:                  false,
		},
		{
			name:                           "failed in FetchResourceTemplate, err is not NotFound",
			resource:                       workv1alpha2.ObjectReference{},
			clusterRoleNameInDynamicClient: "cr",
			resourceExistInClient:          true,
			expectedError:                  true,
		},
		{
			name: "failed in AggregateClusterResourceBindingWorkStatus",
			resource: workv1alpha2.ObjectReference{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
				Name:       "cr",
			},
			clusterRoleNameInDynamicClient: "cr",
			resourceExistInClient:          false,
			expectedError:                  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := generateCRBStatusController()
			c.DynamicClient = dynamicfake.NewSimpleDynamicClient(scheme.Scheme,
				&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: tt.clusterRoleNameInDynamicClient}})
			c.ResourceInterpreter = FakeResourceInterpreter{DefaultInterpreter: native.NewDefaultInterpreter()}

			binding := &workv1alpha2.ClusterResourceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "binding",
				},
				Spec: workv1alpha2.ResourceBindingSpec{
					Resource: tt.resource,
				},
			}

			if tt.resourceExistInClient {
				c.Client = fake.NewClientBuilder().WithScheme(gclient.NewSchema()).WithObjects(binding).WithStatusSubresource(binding).
					WithIndex(&workv1alpha1.Work{}, workv1alpha2.ClusterResourceBindingPermanentIDLabel, utilhelper.GenOneLabelEqualIndexerFunc(workv1alpha2.ClusterResourceBindingPermanentIDLabel)).
					Build()
			}

			err := c.syncBindingStatus(context.Background(), binding)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
