// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PluginSpec struct {
	// Disabled indicates that the bundle should not be installed.
	Disabled bool `json:"disabled,omitempty"`

	// Kind bundle kind.
	Kind BundleKind `json:"kind,omitempty"`

	// URL is the URL of helm repository, git clone url, tarball url, s3 url, etc.
	// +kubebuilder:validation:Required
	URL string `json:"url,omitempty"`

	// Version is the version of helm chart, git revision, etc.
	// +kubebuilder:validation:Required
	Version string `json:"version,omitempty"`

	// Chart is the name of the chart to install.
	Chart string `json:"chart,omitempty"`

	// Path is the path in a tarball to the chart/kustomize.
	Path string `json:"path,omitempty"`

	// InstallNamespace is the namespace to install the bundle into.
	// If not specified, the bundle will be installed into the namespace of the bundle.
	InstallNamespace string `json:"installNamespace,omitempty"`

	// Dependencies is a list of bundles that this bundle depends on.
	// The bundle will be installed after all dependencies are exists.
	Dependencies []corev1.ObjectReference `json:"dependencies,omitempty"`

	// Values is a nested map of helm values.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Optional
	Values Values `json:"values,omitempty"`

	// ValuesFiles is a list of references to helm values files.
	// Ref can be a configmap or secret.
	// +kubebuilder:validation:Optional
	ValuesFrom []ValuesFrom `json:"valuesFrom,omitempty"`

	// FileOverrides is a list of file overrides/append to the helm chart.
	FileOverrides []FileOverride `json:"fileOverrides,omitempty"`
}

const (
	ValuesFromKindConfigmap = "ConfigMap"
	ValuesFromKindSecret    = "Secret"
)

type ValuesFrom struct {
	// Kind is the type of resource being referenced
	// +kubebuilder:validation:Enum=ConfigMap;Secret
	Kind string `json:"kind"`
	// Name is the name of resource being referenced
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
	// An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.
	// +kubebuilder:validation:Optional
	Prefix string `json:"prefix,omitempty"`
	// Optional set to true to ignore references not found error
	Optional bool `json:"optional,omitempty"`
}

type FileOverride struct {
	// Name is the name of the file to override
	Name string `json:"name"`
	// Content is the content of the file to override
	Content string `json:"content"`
}

type PluginStatus struct {
	// The generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase is the current state of the release
	Phase Phase `json:"phase,omitempty"`

	// Message is the message associated with the status
	// In helm, it's the notes contents.
	Message string `json:"message,omitempty"`

	// Values is a nested map of final helm values.
	// +kubebuilder:pruning:PreserveUnknownFields
	Values Values `json:"values,omitempty"`

	// Version is the version of the bundle.
	// In helm, Version is the version of the chart.
	Version string `json:"version,omitempty"`

	// AppVersion is the app version of the bundle.
	AppVersion string `json:"appVersion,omitempty"`

	// Namespace is the namespace where the bundle is installed.
	Namespace string `json:"namespace,omitempty"`

	// CreationTimestamp is the first creation timestamp of the bundle.
	CreationTimestamp metav1.Time `json:"creationTimestamp,omitempty"`

	// UpgradeTimestamp is the time when the bundle was last upgraded.
	UpgradeTimestamp metav1.Time `json:"upgradeTimestamp,omitempty"`

	// Resources is a list of resources created/managed by the bundle.
	Resources []ManagedResource `json:"resources,omitempty"`
}

type ManagedResource struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name,omitempty"`
}

type Phase string

// +kubebuilder:validation:Enum=helm;kustomize;template
type BundleKind string

const (
	BundleKindHelm      BundleKind = "helm"
	BundleKindKustomize BundleKind = "kustomize"
	BundleKindTemplate  BundleKind = "template"
)

const (
	PhaseDisabled  Phase = "Disabled"  // Bundle is disabled. the .spce.disbaled field is set to true or DeletionTimestamp is set.
	PhaseFailed    Phase = "Failed"    // Failed on install.
	PhaseInstalled Phase = "Installed" // Bundle is installed
)
