package util

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMergeOwnerReferences(t *testing.T) {
	BoolTrue := true
	tests := []struct {
		name     string
		exist    []metav1.OwnerReference
		input    []metav1.OwnerReference
		expected []metav1.OwnerReference
	}{
		{
			name:  "nil exist ownerreferences",
			exist: nil,
			input: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment",
					UID:        "demo-deployment",
				},
			},
			expected: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment",
					UID:        "demo-deployment",
				},
			},
		},
		{
			name: "exist with one controller and input is the same object with no controller",
			exist: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment",
					UID:        "demo-deployment",
					Controller: &BoolTrue,
				},
			},
			input: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment",
					UID:        "demo-deployment",
				},
			},
			expected: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment",
					UID:        "demo-deployment",
				},
			},
		},
		{
			name: "exist with one controller and input other object with no controller",
			exist: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment",
					UID:        "demo-deployment",
					Controller: &BoolTrue,
				},
			},
			input: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment-2",
					UID:        "demo-deployment-2",
				},
			},
			expected: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment",
					UID:        "demo-deployment",
					Controller: &BoolTrue,
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment-2",
					UID:        "demo-deployment-2",
				},
			},
		},
		{
			name: "exist with one controller and input with one controller",
			exist: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment",
					UID:        "demo-deployment",
					Controller: &BoolTrue,
				},
			},
			input: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment-2",
					UID:        "demo-deployment-2",
					Controller: &BoolTrue,
				},
			},
			expected: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment",
					UID:        "demo-deployment",
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment-2",
					UID:        "demo-deployment-2",
					Controller: &BoolTrue,
				},
			},
		},
		{
			name: "exist with one controller and input with two controllers",
			exist: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment",
					UID:        "demo-deployment",
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment-2",
					UID:        "demo-deployment-2",
					Controller: &BoolTrue,
				},
			},
			input: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment-3",
					UID:        "demo-deployment-3",
					Controller: &BoolTrue,
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment-4",
					UID:        "demo-deployment-4",
					Controller: &BoolTrue,
				},
			},
			expected: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment",
					UID:        "demo-deployment",
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment-2",
					UID:        "demo-deployment-2",
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment-3",
					UID:        "demo-deployment-3",
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "demo-deployment-4",
					UID:        "demo-deployment-4",
					Controller: &BoolTrue,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := MergeOwnerReferences(tt.exist, tt.input)
			if !reflect.DeepEqual(res, tt.expected) {
				t.Errorf("MergeOwnerReferences() = %v, want %v", res, tt.expected)
			}
		})
	}
}
