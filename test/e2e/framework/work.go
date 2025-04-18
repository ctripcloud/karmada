/*
Copyright 2024 The Karmada Authors.

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

package framework

import (
	"context"
	"fmt"

	"github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	karmada "github.com/karmada-io/karmada/pkg/generated/clientset/versioned"
)

// WaitForWorkToDisappear waiting for work to disappear util timeout
func WaitForWorkToDisappear(client karmada.Interface, namespace, name string) {
	klog.Infof("Waiting for work(%s/%s) to disappear", namespace, name)
	gomega.Eventually(func() error {
		_, err := client.WorkV1alpha1().Works(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("work(%s/%s) still exist", namespace, name)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get work(%s/%s), err: %w", namespace, name, err)
		}
		return nil
	}, PollTimeout, PollInterval).ShouldNot(gomega.HaveOccurred())
}
