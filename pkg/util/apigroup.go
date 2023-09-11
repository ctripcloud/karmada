package util

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ManagedResourceConfig represents the configuration that identifies the API resources should be managed from propagating.
type ManagedResourceConfig struct {
	// Groups holds a collection of API group, all resources under this group will be managed.
	Groups map[string]struct{}
	// GroupVersions holds a collection of API GroupVersion, all resource under this GroupVersion will be managed.
	GroupVersions map[schema.GroupVersion]struct{}
	// GroupVersionKinds holds a collection of resource that should be managed.
	GroupVersionKinds map[schema.GroupVersionKind]struct{}
}

// NewManagedResourceConfig to create ManagedResourceConfig, use nil default GVK to make managed apis visible in args
func NewManagedResourceConfig() *ManagedResourceConfig {
	r := &ManagedResourceConfig{
		Groups:            map[string]struct{}{},
		GroupVersions:     map[schema.GroupVersion]struct{}{},
		GroupVersionKinds: map[schema.GroupVersionKind]struct{}{},
	}

	return r
}

// Parse parses the --managed-propagating-apis input.
func (r *ManagedResourceConfig) Parse(c string) error {
	// default(empty) input
	if c == "" {
		return nil
	}

	tokens := strings.Split(c, ";")
	for _, token := range tokens {
		if err := r.parseSingle(token); err != nil {
			return fmt.Errorf("parse --managed-propagating-apis %w", err)
		}
	}

	return nil
}

func (r *ManagedResourceConfig) parseSingle(token string) error {
	switch strings.Count(token, "/") {
	// Assume user don't want to skip the 'core'(no group name) group.
	// So, it should be the case "<group>".
	case 0:
		r.Groups[token] = struct{}{}
	// it should be the case "<group>/<version>"
	case 1:
		// for core group which don't have the group name, the case should be "v1/<kind>" or "v1/<kind>,<kind>..."
		if strings.HasPrefix(token, "v1") {
			var kinds []string
			for _, k := range strings.Split(token, ",") {
				if strings.Contains(k, "/") { // "v1/<kind>"
					s := strings.Split(k, "/")
					kinds = append(kinds, s[1])
				} else {
					kinds = append(kinds, k)
				}
			}
			for _, k := range kinds {
				gvk := schema.GroupVersionKind{
					Version: "v1",
					Kind:    k,
				}
				r.GroupVersionKinds[gvk] = struct{}{}
			}
		} else { // case "<group>/<version>"
			parts := strings.Split(token, "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid token: %s", token)
			}
			gv := schema.GroupVersion{
				Group:   parts[0],
				Version: parts[1],
			}
			r.GroupVersions[gv] = struct{}{}
		}
	// parameter format: "<group>/<version>/<kind>" or "<group>/<version>/<kind>,<kind>..."
	case 2:
		g := ""
		v := ""
		var kinds []string
		for _, k := range strings.Split(token, ",") {
			if strings.Contains(k, "/") {
				s := strings.Split(k, "/")
				g = s[0]
				v = s[1]
				kinds = append(kinds, s[2])
			} else {
				kinds = append(kinds, k)
			}
		}
		for _, k := range kinds {
			gvk := schema.GroupVersionKind{
				Group:   g,
				Version: v,
				Kind:    k,
			}
			r.GroupVersionKinds[gvk] = struct{}{}
		}
	default:
		return fmt.Errorf("invalid parameter: %s", token)
	}

	return nil
}

// GroupVersionEnabled returns whether GroupVersion is enabled.
func (r *ManagedResourceConfig) GroupVersionEnabled(gv schema.GroupVersion) bool {
	if _, ok := r.GroupVersions[gv]; ok {
		return true
	}
	return false
}

// GroupVersionKindEnabled returns whether GroupVersionKind is enabled.
func (r *ManagedResourceConfig) GroupVersionKindEnabled(gvk schema.GroupVersionKind) bool {
	if _, ok := r.GroupVersionKinds[gvk]; ok {
		return true
	}
	return false
}

// GroupEnabled returns whether Group is enabled.
func (r *ManagedResourceConfig) GroupEnabled(g string) bool {
	if _, ok := r.Groups[g]; ok {
		return true
	}
	return false
}

// EnableGroup to enable group.
func (r *ManagedResourceConfig) EnableGroup(g string) {
	r.Groups[g] = struct{}{}
}

// EnableGroupVersion to enable GroupVersion.
func (r *ManagedResourceConfig) EnableGroupVersion(gv schema.GroupVersion) {
	r.GroupVersions[gv] = struct{}{}
}

// EnableGroupVersionKind to enable GroupVersionKind.
func (r *ManagedResourceConfig) EnableGroupVersionKind(gvk schema.GroupVersionKind) {
	r.GroupVersionKinds[gvk] = struct{}{}
}
