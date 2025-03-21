/*
Copyright 2020 The Karmada Authors.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ResourceKindPropagationPolicy is kind name of PropagationPolicy.
	ResourceKindPropagationPolicy = "PropagationPolicy"
	// ResourceSingularPropagationPolicy is singular name of PropagationPolicy.
	ResourceSingularPropagationPolicy = "propagationpolicy"
	// ResourcePluralPropagationPolicy is kind plural name of PropagationPolicy.
	ResourcePluralPropagationPolicy = "propagationpolicies"
	// ResourceNamespaceScopedPropagationPolicy indicates if PropagationPolicy is NamespaceScoped.
	ResourceNamespaceScopedPropagationPolicy = true

	// ResourceKindClusterPropagationPolicy is kind name of ClusterPropagationPolicy.
	ResourceKindClusterPropagationPolicy = "ClusterPropagationPolicy"
	// ResourceSingularClusterPropagationPolicy is singular name of ClusterPropagationPolicy.
	ResourceSingularClusterPropagationPolicy = "clusterpropagationpolicy"
	// ResourcePluralClusterPropagationPolicy is plural name of ClusterPropagationPolicy.
	ResourcePluralClusterPropagationPolicy = "clusterpropagationpolicies"
	// ResourceNamespaceScopedClusterPropagationPolicy indicates if ClusterPropagationPolicy is NamespaceScoped.
	ResourceNamespaceScopedClusterPropagationPolicy = false
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:path=propagationpolicies,scope=Namespaced,shortName=pp,categories={karmada-io}
// +kubebuilder:printcolumn:JSONPath=`.spec.conflictResolution`,name="CONFLICT-RESOLUTION",type=string
// +kubebuilder:printcolumn:JSONPath=`.spec.priority`,name="PRIORITY",type=string
// +kubebuilder:printcolumn:JSONPath=`.metadata.creationTimestamp`,name="AGE",type=date

// PropagationPolicy represents the policy that propagates a group of resources to one or more clusters.
type PropagationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired behavior of PropagationPolicy.
	// +required
	Spec PropagationSpec `json:"spec"`
}

// PropagationSpec represents the desired behavior of PropagationPolicy.
type PropagationSpec struct {
	// ResourceSelectors used to select resources.
	// Nil or empty selector is not allowed and doesn't mean match all kinds
	// of resources for security concerns that sensitive resources(like Secret)
	// might be accidentally propagated.
	// +required
	// +kubebuilder:validation:MinItems=1
	ResourceSelectors []ResourceSelector `json:"resourceSelectors"`

	// Association tells if relevant resources should be selected automatically.
	// e.g. a ConfigMap referred by a Deployment.
	// default false.
	// Deprecated: in favor of PropagateDeps.
	// +optional
	Association bool `json:"association,omitempty"`

	// PropagateDeps tells if relevant resources should be propagated automatically.
	// Take 'Deployment' which referencing 'ConfigMap' and 'Secret' as an example, when 'propagateDeps' is 'true',
	// the referencing resources could be omitted(for saving config effort) from 'resourceSelectors' as they will be
	// propagated along with the Deployment. In addition to the propagating process, the referencing resources will be
	// migrated along with the Deployment in the fail-over scenario.
	//
	// Defaults to false.
	// +optional
	PropagateDeps bool `json:"propagateDeps,omitempty"`

	// Placement represents the rule for select clusters to propagate resources.
	// +optional
	Placement Placement `json:"placement,omitempty"`

	// Priority indicates the importance of a policy(PropagationPolicy or ClusterPropagationPolicy).
	// A policy will be applied for the matched resource templates if there is
	// no other policies with higher priority at the point of the resource
	// template be processed.
	// Once a resource template has been claimed by a policy, by default it will
	// not be preempted by following policies even with a higher priority.
	// See Preemption for more details.
	//
	// In case of two policies have the same priority, the one with a more precise
	// matching rules in ResourceSelectors wins:
	// - matching by name(resourceSelector.name) has higher priority than
	//   by selector(resourceSelector.labelSelector)
	// - matching by selector(resourceSelector.labelSelector) has higher priority
	//   than by APIVersion(resourceSelector.apiVersion) and Kind(resourceSelector.kind).
	// If there is still no winner at this point, the one with the lower alphabetic
	// order wins, e.g. policy 'bar' has higher priority than 'foo'.
	//
	// The higher the value, the higher the priority. Defaults to zero.
	// +optional
	// +kubebuilder:default=0
	Priority *int32 `json:"priority,omitempty"`

	// Preemption declares the behaviors for preempting.
	// Valid options are "Always" and "Never".
	//
	// +kubebuilder:default="Never"
	// +kubebuilder:validation:Enum=Always;Never
	// +optional
	Preemption PreemptionBehavior `json:"preemption,omitempty"`

	// DependentOverrides represents the list of overrides(OverridePolicy)
	// which must present before the current PropagationPolicy takes effect.
	//
	// It used to explicitly specify overrides which current PropagationPolicy rely on.
	// A typical scenario is the users create OverridePolicy(ies) and resources at the same time,
	// they want to ensure the new-created policies would be adopted.
	//
	// Note: For the overrides, OverridePolicy(ies) in current namespace and ClusterOverridePolicy(ies),
	// which not present in this list will still be applied if they matches the resources.
	// +optional
	DependentOverrides []string `json:"dependentOverrides,omitempty"`

	// SchedulerName represents which scheduler to proceed the scheduling.
	// If specified, the policy will be dispatched by specified scheduler.
	// If not specified, the policy will be dispatched by default scheduler.
	// +kubebuilder:default="default-scheduler"
	// +optional
	SchedulerName string `json:"schedulerName,omitempty"`

	// Failover indicates how Karmada migrates applications in case of failures.
	// If this value is nil, failover is disabled.
	// +optional
	Failover *FailoverBehavior `json:"failover,omitempty"`

	// ConflictResolution declares how potential conflict should be handled when
	// a resource that is being propagated already exists in the target cluster.
	//
	// It defaults to "Abort" which means stop propagating to avoid unexpected
	// overwrites. The "Overwrite" might be useful when migrating legacy cluster
	// resources to Karmada, in which case conflict is predictable and can be
	// instructed to Karmada take over the resource by overwriting.
	//
	// +kubebuilder:default="Abort"
	// +kubebuilder:validation:Enum=Abort;Overwrite
	// +optional
	ConflictResolution ConflictResolution `json:"conflictResolution,omitempty"`

	// ActivationPreference indicates how the referencing resource template will
	// be propagated, in case of policy changes.
	//
	// If empty, the resource template will respond to policy changes
	// immediately, in other words, any policy changes will drive the resource
	// template to be propagated immediately as per the current propagation rules.
	//
	// If the value is 'Lazy' means the policy changes will not take effect for now
	// but defer to the resource template changes, in other words, the resource
	// template will not be propagated as per the current propagation rules until
	// there is an update on it.
	// This is an experimental feature that might help in a scenario where a policy
	// manages huge amount of resource templates, changes to a policy typically
	// affect numerous applications simultaneously. A minor misconfiguration
	// could lead to widespread failures. With this feature, the change can be
	// gradually rolled out through iterative modifications of resource templates.
	//
	// +kubebuilder:validation:Enum=Lazy
	// +optional
	ActivationPreference ActivationPreference `json:"activationPreference,omitempty"`

	// Suspension declares the policy for suspending different aspects of propagation.
	// nil means no suspension. no default values.
	// +optional
	Suspension *Suspension `json:"suspension,omitempty"`

	// PreserveResourcesOnDeletion controls whether resources should be preserved on the
	// member clusters when the resource template is deleted.
	// If set to true, resources will be preserved on the member clusters.
	// Default is false, which means resources will be deleted along with the resource template.
	//
	// This setting is particularly useful during workload migration scenarios to ensure
	// that rollback can occur quickly without affecting the workloads running on the
	// member clusters.
	//
	// Additionally, this setting applies uniformly across all member clusters and will not
	// selectively control preservation on only some clusters.
	//
	// Note: This setting does not apply to the deletion of the policy itself.
	// When the policy is deleted, the resource templates and their corresponding
	// propagated resources in member clusters will remain unchanged unless explicitly deleted.
	//
	// +optional
	PreserveResourcesOnDeletion *bool `json:"preserveResourcesOnDeletion,omitempty"`

	// SchedulePriority defines how Karmada should resolve the priority and preemption policy
	// for workload scheduling.
	//
	// This setting is useful for controlling the scheduling behavior of offline workloads.
	// By setting a higher or lower priority, users can control which workloads are scheduled first.
	// Additionally, it allows specifying a preemption policy where higher-priority workloads can
	// preempt lower-priority ones in scenarios of resource contention.
	//
	// Note: This feature is currently in the alpha stage. The priority-based scheduling functionality is
	// controlled by the PriorityBasedScheduling feature gate, and preemption is controlled by the
	// PriorityBasedPreemptiveScheduling feature gate. Currently, only priority-based scheduling is
	// supported. Preemption functionality is not yet available and will be introduced in future
	// releases as the feature matures.
	//
	// +optional
	SchedulePriority *SchedulePriority `json:"schedulePriority,omitempty"`
}

// ResourceSelector the resources will be selected.
type ResourceSelector struct {
	// APIVersion represents the API version of the target resources.
	// +required
	APIVersion string `json:"apiVersion"`

	// Kind represents the Kind of the target resources.
	// +required
	Kind string `json:"kind"`

	// Namespace of the target resource.
	// Default is empty, which means inherit from the parent object scope.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the target resource.
	// Default is empty, which means selecting all resources.
	// +optional
	Name string `json:"name,omitempty"`

	// A label query over a set of resources.
	// If name is not empty, labelSelector will be ignored.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// FieldSelector is a field filter.
type FieldSelector struct {
	// A list of field selector requirements.
	MatchExpressions []corev1.NodeSelectorRequirement `json:"matchExpressions,omitempty"`
}

// Suspension defines the policy for suspending different aspects of propagation.
type Suspension struct {
	// Dispatching controls whether dispatching should be suspended.
	// nil means not suspend, no default value, only accepts 'true'.
	// Note: true means stop propagating to all clusters. Can not co-exist
	// with DispatchingOnClusters which is used to suspend particular clusters.
	// +optional
	Dispatching *bool `json:"dispatching,omitempty"`

	// DispatchingOnClusters declares a list of clusters to which the dispatching
	// should be suspended.
	// Note: Can not co-exist with Dispatching which is used to suspend all.
	// +optional
	DispatchingOnClusters *SuspendClusters `json:"dispatchingOnClusters,omitempty"`
}

// SuspendClusters represents a group of clusters that should be suspended from propagating.
// Note: No plan to introduce the label selector or field selector to select clusters yet, as it
// would make the system unpredictable.
type SuspendClusters struct {
	// ClusterNames is the list of clusters to be selected.
	// +optional
	ClusterNames []string `json:"clusterNames,omitempty"`
}

// PurgeMode represents how to deal with the legacy application on the
// cluster from which the application is migrated.
type PurgeMode string

const (
	// Immediately represents that Karmada will immediately evict the legacy
	// application. This is useful in scenarios where an application can not
	// tolerate two instances running simultaneously.
	// For example, the Flink application supports exactly-once state consistency,
	// which means it requires that no two instances of the application are running
	// at the same time. During a failover, it is crucial to ensure that the old
	// application is removed before creating a new one to avoid duplicate
	// processing and maintaining state consistency.
	Immediately PurgeMode = "Immediately"
	// Graciously represents that Karmada will wait for the application to
	// come back to healthy on the new cluster or after a timeout is reached
	// before evicting the application.
	Graciously PurgeMode = "Graciously"
	// Never represents that Karmada will not evict the application and
	// users manually confirms how to clean up redundant copies.
	Never PurgeMode = "Never"
)

// FailoverBehavior indicates failover behaviors in case of an application or
// cluster failure.
type FailoverBehavior struct {
	// Application indicates failover behaviors in case of application failure.
	// If this value is nil, failover is disabled.
	// If set, the PropagateDeps should be true so that the dependencies could
	// be migrated along with the application.
	// +optional
	Application *ApplicationFailoverBehavior `json:"application,omitempty"`

	// Cluster indicates failover behaviors in case of cluster failure.
	// If this value is nil, failover is disabled.
	// +optional
	// Cluster *ClusterFailoverBehavior `json:"cluster,omitempty"`
}

// ApplicationFailoverBehavior indicates application failover behaviors.
type ApplicationFailoverBehavior struct {
	// DecisionConditions indicates the decision conditions of performing the failover process.
	// Only when all conditions are met can the failover process be performed.
	// Currently, DecisionConditions includes several conditions:
	// - TolerationSeconds (optional)
	// +required
	DecisionConditions DecisionConditions `json:"decisionConditions"`

	// PurgeMode represents how to deal with the legacy applications on the
	// cluster from which the application is migrated.
	// Valid options are "Immediately", "Graciously" and "Never".
	// Defaults to "Graciously".
	// +kubebuilder:validation:Enum=Immediately;Graciously;Never
	// +kubebuilder:default=Graciously
	// +optional
	PurgeMode PurgeMode `json:"purgeMode,omitempty"`

	// GracePeriodSeconds is the maximum waiting duration in seconds before
	// application on the migrated cluster should be deleted.
	// Required only when PurgeMode is "Graciously" and defaults to 600s.
	// If the application on the new cluster cannot reach a Healthy state,
	// Karmada will delete the application after GracePeriodSeconds is reached.
	// Value must be positive integer.
	// +optional
	GracePeriodSeconds *int32 `json:"gracePeriodSeconds,omitempty"`

	// StatePreservation defines the policy for preserving and restoring state data
	// during failover events for stateful applications.
	//
	// When an application fails over from one cluster to another, this policy enables
	// the extraction of critical data from the original resource configuration.
	// Upon successful migration, the extracted data is then re-injected into the new
	// resource, ensuring that the application can resume operation with its previous
	// state intact.
	// This is particularly useful for stateful applications where maintaining data
	// consistency across failover events is crucial.
	// If not specified, means no state data will be preserved.
	//
	// Note: This requires the StatefulFailoverInjection feature gate to be enabled,
	// which is alpha.
	// +optional
	StatePreservation *StatePreservation `json:"statePreservation,omitempty"`
}

// DecisionConditions represents the decision conditions of performing the failover process.
type DecisionConditions struct {
	// TolerationSeconds represents the period of time Karmada should wait
	// after reaching the desired state before performing failover process.
	// If not specified, Karmada will immediately perform failover process.
	// Defaults to 300s.
	// +kubebuilder:default=300
	// +optional
	TolerationSeconds *int32 `json:"tolerationSeconds,omitempty"`
}

// StatePreservation defines the policy for preserving state during failover events.
type StatePreservation struct {
	// Rules contains a list of StatePreservationRule configurations.
	// Each rule specifies a JSONPath expression targeting specific pieces of
	// state data to be preserved during failover events. An AliasLabelName is associated
	// with each rule, serving as a label key when the preserved data is passed
	// to the new cluster.
	// +required
	Rules []StatePreservationRule `json:"rules"`
}

// StatePreservationRule defines a single rule for state preservation.
// It includes a JSONPath expression and an alias name that will be used
// as a label key when passing state information to the new cluster.
type StatePreservationRule struct {
	// AliasLabelName is the name that will be used as a label key when the preserved
	// data is passed to the new cluster. This facilitates the injection of the
	// preserved state back into the application resources during recovery.
	// +required
	AliasLabelName string `json:"aliasLabelName"`

	// JSONPath is the JSONPath template used to identify the state data
	// to be preserved from the original resource configuration.
	// The JSONPath syntax follows the Kubernetes specification:
	// https://kubernetes.io/docs/reference/kubectl/jsonpath/
	//
	// Note: The JSONPath expression will start searching from the "status" field of
	// the API resource object by default. For example, to extract the "availableReplicas"
	// from a Deployment, the JSONPath expression should be "{.availableReplicas}", not
	// "{.status.availableReplicas}".
	//
	// +required
	JSONPath string `json:"jsonPath"`
}

// Placement represents the rule for select clusters.
type Placement struct {
	// ClusterAffinity represents scheduling restrictions to a certain set of clusters.
	// Note:
	//   1. ClusterAffinity can not co-exist with ClusterAffinities.
	//   2. If both ClusterAffinity and ClusterAffinities are not set, any cluster
	//      can be scheduling candidates.
	// +optional
	ClusterAffinity *ClusterAffinity `json:"clusterAffinity,omitempty"`

	// ClusterAffinities represents scheduling restrictions to multiple cluster
	// groups that indicated by ClusterAffinityTerm.
	//
	// The scheduler will evaluate these groups one by one in the order they
	// appear in the spec, the group that does not satisfy scheduling restrictions
	// will be ignored which means all clusters in this group will not be selected
	// unless it also belongs to the next group(a cluster could belong to multiple
	// groups).
	//
	// If none of the groups satisfy the scheduling restrictions, then scheduling
	// fails, which means no cluster will be selected.
	//
	// Note:
	//   1. ClusterAffinities can not co-exist with ClusterAffinity.
	//   2. If both ClusterAffinity and ClusterAffinities are not set, any cluster
	//      can be scheduling candidates.
	//
	// Potential use case 1:
	// The private clusters in the local data center could be the main group, and
	// the managed clusters provided by cluster providers could be the secondary
	// group. So that the Karmada scheduler would prefer to schedule workloads
	// to the main group and the second group will only be considered in case of
	// the main group does not satisfy restrictions(like, lack of resources).
	//
	// Potential use case 2:
	// For the disaster recovery scenario, the clusters could be organized to
	// primary and backup groups, the workloads would be scheduled to primary
	// clusters firstly, and when primary cluster fails(like data center power off),
	// Karmada scheduler could migrate workloads to the backup clusters.
	//
	// +optional
	ClusterAffinities []ClusterAffinityTerm `json:"clusterAffinities,omitempty"`

	// ClusterTolerations represents the tolerations.
	// +optional
	ClusterTolerations []corev1.Toleration `json:"clusterTolerations,omitempty"`

	// SpreadConstraints represents a list of the scheduling constraints.
	// +optional
	SpreadConstraints []SpreadConstraint `json:"spreadConstraints,omitempty"`

	// ReplicaScheduling represents the scheduling policy on dealing with the number of replicas
	// when propagating resources that have replicas in spec (e.g. deployments, statefulsets) to member clusters.
	// +optional
	ReplicaScheduling *ReplicaSchedulingStrategy `json:"replicaScheduling,omitempty"`
}

// SpreadFieldValue is the type to define valid values for SpreadConstraint.SpreadByField
type SpreadFieldValue string

// Available fields for spreading are: cluster, region, zone, and provider.
const (
	SpreadByFieldCluster  SpreadFieldValue = "cluster"
	SpreadByFieldRegion   SpreadFieldValue = "region"
	SpreadByFieldZone     SpreadFieldValue = "zone"
	SpreadByFieldProvider SpreadFieldValue = "provider"
)

// SpreadConstraint represents the spread constraints on resources.
type SpreadConstraint struct {
	// SpreadByField represents the fields on Karmada cluster API used for
	// dynamically grouping member clusters into different groups.
	// Resources will be spread among different cluster groups.
	// Available fields for spreading are: cluster, region, zone, and provider.
	// SpreadByField should not co-exist with SpreadByLabel.
	// If both SpreadByField and SpreadByLabel are empty, SpreadByField will be set to "cluster" by system.
	// +kubebuilder:validation:Enum=cluster;region;zone;provider
	// +optional
	SpreadByField SpreadFieldValue `json:"spreadByField,omitempty"`

	// SpreadByLabel represents the label key used for
	// grouping member clusters into different groups.
	// Resources will be spread among different cluster groups.
	// SpreadByLabel should not co-exist with SpreadByField.
	// +optional
	SpreadByLabel string `json:"spreadByLabel,omitempty"`

	// MaxGroups restricts the maximum number of cluster groups to be selected.
	// +optional
	MaxGroups int `json:"maxGroups,omitempty"`

	// MinGroups restricts the minimum number of cluster groups to be selected.
	// Defaults to 1.
	// +optional
	MinGroups int `json:"minGroups,omitempty"`
}

// ClusterAffinity represents the filter to select clusters.
type ClusterAffinity struct {
	// LabelSelector is a filter to select member clusters by labels.
	// If non-nil and non-empty, only the clusters match this filter will be selected.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// FieldSelector is a filter to select member clusters by fields.
	// The key(field) of the match expression should be 'provider', 'region', or 'zone',
	// and the operator of the match expression should be 'In' or 'NotIn'.
	// If non-nil and non-empty, only the clusters match this filter will be selected.
	// +optional
	FieldSelector *FieldSelector `json:"fieldSelector,omitempty"`

	// ClusterNames is the list of clusters to be selected.
	// +optional
	ClusterNames []string `json:"clusterNames,omitempty"`

	// ExcludedClusters is the list of clusters to be ignored.
	// +optional
	ExcludeClusters []string `json:"exclude,omitempty"`
}

// ClusterAffinityTerm selects a set of cluster.
type ClusterAffinityTerm struct {
	// AffinityName is the name of the cluster group.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=32
	// +required
	AffinityName string `json:"affinityName"`

	ClusterAffinity `json:",inline"`
}

// ReplicaSchedulingType describes scheduling methods for the "replicas" in a resource.
type ReplicaSchedulingType string

const (
	// ReplicaSchedulingTypeDuplicated means when propagating a resource,
	// each candidate member cluster will directly apply the original replicas.
	ReplicaSchedulingTypeDuplicated ReplicaSchedulingType = "Duplicated"
	// ReplicaSchedulingTypeDivided means when propagating a resource,
	// each candidate member cluster will get only a part of original replicas.
	ReplicaSchedulingTypeDivided ReplicaSchedulingType = "Divided"
)

// ReplicaDivisionPreference describes options of how replicas can be scheduled.
type ReplicaDivisionPreference string

const (
	// ReplicaDivisionPreferenceAggregated divides replicas into clusters as few as possible,
	// while respecting clusters' resource availabilities during the division.
	ReplicaDivisionPreferenceAggregated ReplicaDivisionPreference = "Aggregated"
	// ReplicaDivisionPreferenceWeighted divides replicas by weight according to WeightPreference.
	ReplicaDivisionPreferenceWeighted ReplicaDivisionPreference = "Weighted"
)

// ReplicaSchedulingStrategy represents the assignment strategy of replicas.
type ReplicaSchedulingStrategy struct {
	// ReplicaSchedulingType determines how the replicas is scheduled when karmada propagating
	// a resource. Valid options are Duplicated and Divided.
	// "Duplicated" duplicates the same replicas to each candidate member cluster from resource.
	// "Divided" divides replicas into parts according to number of valid candidate member
	// clusters, and exact replicas for each cluster are determined by ReplicaDivisionPreference.
	// +kubebuilder:validation:Enum=Duplicated;Divided
	// +kubebuilder:default=Divided
	// +optional
	ReplicaSchedulingType ReplicaSchedulingType `json:"replicaSchedulingType,omitempty"`

	// ReplicaDivisionPreference determines how the replicas is divided
	// when ReplicaSchedulingType is "Divided". Valid options are Aggregated and Weighted.
	// "Aggregated" divides replicas into clusters as few as possible,
	// while respecting clusters' resource availabilities during the division.
	// "Weighted" divides replicas by weight according to WeightPreference.
	// +kubebuilder:validation:Enum=Aggregated;Weighted
	// +optional
	ReplicaDivisionPreference ReplicaDivisionPreference `json:"replicaDivisionPreference,omitempty"`

	// WeightPreference describes weight for each cluster or for each group of cluster
	// If ReplicaDivisionPreference is set to "Weighted", and WeightPreference is not set, scheduler will weight all clusters the same.
	// +optional
	WeightPreference *ClusterPreferences `json:"weightPreference,omitempty"`
}

// ClusterPreferences describes weight for each cluster or for each group of cluster.
type ClusterPreferences struct {
	// StaticWeightList defines the static cluster weight.
	// +optional
	StaticWeightList []StaticClusterWeight `json:"staticWeightList,omitempty"`
	// DynamicWeight specifies the factor to generates dynamic weight list.
	// If specified, StaticWeightList will be ignored.
	// +kubebuilder:validation:Enum=AvailableReplicas
	// +optional
	DynamicWeight DynamicWeightFactor `json:"dynamicWeight,omitempty"`
}

// StaticClusterWeight defines the static cluster weight.
type StaticClusterWeight struct {
	// TargetCluster describes the filter to select clusters.
	// +required
	TargetCluster ClusterAffinity `json:"targetCluster"`

	// Weight expressing the preference to the cluster(s) specified by 'TargetCluster'.
	// +kubebuilder:validation:Minimum=1
	// +required
	Weight int64 `json:"weight"`
}

// DynamicWeightFactor represents the weight factor.
// For now only support 'AvailableReplicas', more factors could be extended if there is a need.
type DynamicWeightFactor string

const (
	// DynamicWeightByAvailableReplicas represents the cluster weight list should be generated according to
	// available resource (available replicas).
	// Example:
	//   The scheduler selected 3 clusters (A/B/C) and should divide 12 replicas to them.
	//   Workload:
	//     Desired replica: 12
	//   Cluster:
	//     A: Max available replica: 6
	//     B: Max available replica: 12
	//     C: Max available replica: 18
	//   The weight of cluster A:B:C will be 6:12:18 (equals to 1:2:3). At last, the assignment would be 'A: 2, B: 4, C: 6'.
	DynamicWeightByAvailableReplicas DynamicWeightFactor = "AvailableReplicas"
)

// PreemptionBehavior describes whether and how to preempt resources that are
// claimed by lower-priority PropagationPolicy(ClusterPropagationPolicy).
// +enum
type PreemptionBehavior string

const (
	// PreemptAlways means that preemption is allowed.
	//
	// If it is applied to a PropagationPolicy, it can preempt any resource as
	// per Priority, regardless of whether it has been claimed by a PropagationPolicy
	// or a ClusterPropagationPolicy, as long as it can match the rules defined
	// in ResourceSelector. In addition, if a resource has already been claimed
	// by a ClusterPropagationPolicy, the PropagationPolicy can still preempt it
	// without considering Priority.
	//
	// If it is applied to a ClusterPropagationPolicy, it can only preempt from
	// ClusterPropagationPolicy, and from PropagationPolicy is not allowed.
	PreemptAlways PreemptionBehavior = "Always"

	// PreemptNever means that a PropagationPolicy(ClusterPropagationPolicy) never
	// preempts resources.
	PreemptNever PreemptionBehavior = "Never"
)

// ConflictResolution describes how to resolve the conflict during the process
// of propagation especially the resource already in a member cluster.
type ConflictResolution string

const (
	// ConflictOverwrite means that resolve the conflict by overwriting the
	// resource with the propagating resource template.
	ConflictOverwrite ConflictResolution = "Overwrite"

	// ConflictAbort means that do not resolve the conflict and stop propagating.
	ConflictAbort ConflictResolution = "Abort"
)

// ActivationPreference indicates how the referencing resource template will be propagated, in case of policy changes.
type ActivationPreference string

const (
	// LazyActivation means the policy changes will not take effect for now but defer to the resource template changes,
	// in other words, the resource template will not be propagated as per the current propagation rules until
	// there is an update on it.
	LazyActivation ActivationPreference = "Lazy"
)

// SchedulePriority defines how Karmada should resolve the priority and preemption policy
// for workload scheduling.
type SchedulePriority struct {
	// PriorityClassSource specifies where Karmada should look for the PriorityClass definition.
	// Available options:
	// - KubePriorityClass: Uses Kubernetes PriorityClass (scheduling.k8s.io/v1)
	// - PodPriorityClass: Uses PriorityClassName from PodTemplate: PodSpec.PriorityClassName (not yet implemented)
	// - FederatedPriorityClass: Uses Karmada FederatedPriorityClass (not yet implemented)
	//
	// +kubebuilder:validation:Enum=KubePriorityClass
	// +required
	PriorityClassSource PriorityClassSource `json:"priorityClassSource"`

	// PriorityClassName specifies which PriorityClass to use. Its behavior depends on PriorityClassSource:
	//
	// Behavior of PriorityClassName:
	//
	// For KubePriorityClass:
	// - When specified: Uses the named Kubernetes PriorityClass.
	//
	// For PodPriorityClass:
	// - Uses PriorityClassName from the PodTemplate.
	// - Not yet implemented.
	//
	// For FederatedPriorityClass:
	// - Not yet implemented.
	//
	// +required
	PriorityClassName string `json:"priorityClassName"`
}

// PriorityClassSource defines the type for PriorityClassSource field.
type PriorityClassSource string

const (
	// FederatedPriorityClass specifies to use Karmada FederatedPriorityClass for priority resolution.
	// This feature is planned for future releases and is currently not implemented.
	FederatedPriorityClass PriorityClassSource = "FederatedPriorityClass"

	// KubePriorityClass specifies to use Kubernetes native PriorityClass (scheduling.k8s.io/v1)
	// for priority resolution. This is the default source.
	KubePriorityClass PriorityClassSource = "KubePriorityClass"

	// PodPriorityClass specifies to use the PriorityClassName defined in the workload's
	// PodTemplate for priority resolution.
	PodPriorityClass PriorityClassSource = "PodPriorityClass"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PropagationPolicyList contains a list of PropagationPolicy.
type PropagationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PropagationPolicy `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:path=clusterpropagationpolicies,scope="Cluster",shortName=cpp,categories={karmada-io}
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:JSONPath=`.spec.conflictResolution`,name="CONFLICT-RESOLUTION",type=string
// +kubebuilder:printcolumn:JSONPath=`.spec.priority`,name="PRIORITY",type=string
// +kubebuilder:printcolumn:JSONPath=`.metadata.creationTimestamp`,name="AGE",type=date

// ClusterPropagationPolicy represents the cluster-wide policy that propagates a group of resources to one or more clusters.
// Different with PropagationPolicy that could only propagate resources in its own namespace, ClusterPropagationPolicy
// is able to propagate cluster level resources and resources in any namespace other than system reserved ones.
// System reserved namespaces are: karmada-system, karmada-cluster, karmada-es-*.
type ClusterPropagationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired behavior of ClusterPropagationPolicy.
	// +required
	Spec PropagationSpec `json:"spec"`
}

// +kubebuilder:resource:scope="Cluster"
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPropagationPolicyList contains a list of ClusterPropagationPolicy.
type ClusterPropagationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPropagationPolicy `json:"items"`
}
