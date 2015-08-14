package api

import (
	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
)

// ProjectList is a list of Project objects.
type ProjectList struct {
	kapi.TypeMeta
	kapi.ListMeta
	Items []Project
}

const (
	// These are internal finalizer values to Origin
	FinalizerOrigin kapi.FinalizerName = "openshift.io/origin"
)

// ProjectSpec describes the attributes on a Project
type ProjectSpec struct {
	// Finalizers is an opaque list of values that must be empty to permanently remove object from storage
	Finalizers []kapi.FinalizerName
}

// ProjectStatus is information about the current status of a Project
type ProjectStatus struct {
	Phase kapi.NamespacePhase
}

// Project is a logical top-level container for a set of origin resources
type Project struct {
	kapi.TypeMeta
	kapi.ObjectMeta

	Spec   ProjectSpec
	Status ProjectStatus
}

type ProjectRequest struct {
	kapi.TypeMeta
	kapi.ObjectMeta
	DisplayName string
	Description string
}

// These constants represent annotations keys affixed to projects
const (
	// ProjectDisplayName is an annotation that stores the name displayed when querying for projects
	ProjectDisplayName = "openshift.io/display-name"
	// ProjectDescription is an annotatoion that holds the description of the project
	ProjectDescription = "openshift.io/description"
	// ProjectNodeSelector is an annotation that holds the node selector;
	// the node selector annotation determines which nodes will have pods from this project scheduled to them
	ProjectNodeSelector = "openshift.io/node-selector"
)
