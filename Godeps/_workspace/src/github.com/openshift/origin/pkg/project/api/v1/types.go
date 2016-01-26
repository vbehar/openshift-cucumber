package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	kapi "k8s.io/kubernetes/pkg/api/v1"
)

// ProjectList is a list of Project objects.
type ProjectList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []Project `json:"items" description:"list of projects"`
}

const (
	// These are internal finalizer values to Origin
	FinalizerOrigin kapi.FinalizerName = "openshift.io/origin"
)

// ProjectSpec describes the attributes on a Project
type ProjectSpec struct {
	// Finalizers is an opaque list of values that must be empty to permanently remove object from storage
	Finalizers []kapi.FinalizerName `json:"finalizers,omitempty" description:"an opaque list of values that must be empty to permanently remove object from storage"`
}

// ProjectStatus is information about the current status of a Project
type ProjectStatus struct {
	Phase kapi.NamespacePhase `json:"phase,omitempty" description:"phase is the current lifecycle phase of the project"`
}

// Project is a logical top-level container for a set of origin resources
type Project struct {
	unversioned.TypeMeta `json:",inline"`
	kapi.ObjectMeta      `json:"metadata,omitempty"`

	// Spec defines the behavior of the Namespace.
	Spec ProjectSpec `json:"spec,omitempty" description:"spec defines the behavior of the Project"`

	// Status describes the current status of a Namespace
	Status ProjectStatus `json:"status,omitempty" description:"status describes the current status of a Project; read-only"`
}

type ProjectRequest struct {
	unversioned.TypeMeta `json:",inline"`
	kapi.ObjectMeta      `json:"metadata,omitempty"`
	DisplayName          string `json:"displayName,omitempty" description:"display name to apply to a project"`
	Description          string `json:"description,omitempty" description:"description to apply to a project"`
}
