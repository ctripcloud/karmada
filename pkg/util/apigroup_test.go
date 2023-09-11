package util

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestManagedResourceConfigGVKParse(t *testing.T) {
	tests := []struct {
		input string
		out   []schema.GroupVersionKind
	}{
		{
			input: "v1/Node,Pod;networking.k8s.io/v1beta1/Ingress,IngressClass",
			out: []schema.GroupVersionKind{
				{
					Group:   "",
					Version: "v1",
					Kind:    "Node",
				},
				{
					Group:   "",
					Version: "v1",
					Kind:    "Pod",
				},
				{
					Group:   "networking.k8s.io",
					Version: "v1beta1",
					Kind:    "Ingress",
				},
				{
					Group:   "networking.k8s.io",
					Version: "v1beta1",
					Kind:    "IngressClass",
				},
			}},
	}
	for _, test := range tests {
		r := NewManagedResourceConfig()
		if err := r.Parse(test.input); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		for i, o := range test.out {
			ok := r.GroupVersionKindEnabled(o)
			if !ok {
				t.Errorf("%d: unexpected error: %v", i, o)
			}
		}
	}
}
func TestResourceConfigGVParse(t *testing.T) {
	tests := []struct {
		input string
		out   []schema.GroupVersion
	}{
		{
			input: "networking.k8s.io/v1;test/v1beta1",
			out: []schema.GroupVersion{
				{
					Group:   "networking.k8s.io",
					Version: "v1",
				},
				{
					Group:   "networking.k8s.io",
					Version: "v1",
				},
				{
					Group:   "test",
					Version: "v1beta1",
				},
				{
					Group:   "test",
					Version: "v1beta1",
				},
			}},
	}
	for _, test := range tests {
		r := NewManagedResourceConfig()
		if err := r.Parse(test.input); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		for i, o := range test.out {
			ok := r.GroupVersionEnabled(o)
			if !ok {
				t.Errorf("%d: unexpected error: %v", i, o)
			}
		}
	}
}

func TestManagedResourceConfigGroupParse(t *testing.T) {
	tests := []struct {
		input string
		out   []string
	}{
		{
			input: "networking.k8s.io;test",
			out: []string{
				"networking.k8s.io", "test",
			}},
	}
	for _, test := range tests {
		r := NewManagedResourceConfig()
		if err := r.Parse(test.input); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		for i, o := range test.out {
			ok := r.GroupEnabled(o)
			if !ok {
				t.Errorf("%d: unexpected error: %v", i, o)
			}
		}
	}
}

func TestManagedResourceConfigMixedParse(t *testing.T) {
	tests := []struct {
		input string
		out   ManagedResourceConfig
	}{
		{
			input: "v1/Node,Pod;networking.k8s.io/v1beta1/Ingress,IngressClass;networking.k8s.io;test.com/v1",
			out: ManagedResourceConfig{
				Groups: map[string]struct{}{
					"networking.k8s.io": {},
				},
				GroupVersions: map[schema.GroupVersion]struct{}{
					{
						Group:   "test.com",
						Version: "v1",
					}: {},
				},
				GroupVersionKinds: map[schema.GroupVersionKind]struct{}{
					{
						Group:   "",
						Version: "v1",
						Kind:    "Node",
					}: {},
					{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					}: {},
					{
						Group:   "networking.k8s.io",
						Version: "v1beta1",
						Kind:    "Ingress",
					}: {},
					{
						Group:   "networking.k8s.io",
						Version: "v1beta1",
						Kind:    "IngressClass",
					}: {},
				},
			}},
	}
	for i, test := range tests {
		r := NewManagedResourceConfig()
		if err := r.Parse(test.input); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		for g := range r.Groups {
			ok := r.GroupEnabled(g)
			if !ok {
				t.Errorf("%d: unexpected error: %v", i, g)
			}
		}

		for gv := range r.GroupVersions {
			ok := r.GroupVersionEnabled(gv)
			if !ok {
				t.Errorf("%d: unexpected error: %v", i, gv)
			}
		}

		for gvk := range r.GroupVersionKinds {
			ok := r.GroupVersionKindEnabled(gvk)
			if !ok {
				t.Errorf("%d: unexpected error: %v", i, gvk)
			}
		}
	}
}
