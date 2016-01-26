package testclient

import (
	ktestclient "k8s.io/kubernetes/pkg/client/unversioned/testclient"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"

	authorizationapi "github.com/openshift/origin/pkg/authorization/api"
)

// FakeClusterRoles implements ClusterRoleInterface. Meant to be embedded into a struct to get a default
// implementation. This makes faking out just the methods you want to test easier.
type FakeClusterRoles struct {
	Fake *Fake
}

func (c *FakeClusterRoles) Get(name string) (*authorizationapi.ClusterRole, error) {
	obj, err := c.Fake.Invokes(ktestclient.NewRootGetAction("clusterroles", name), &authorizationapi.ClusterRole{})
	if obj == nil {
		return nil, err
	}

	return obj.(*authorizationapi.ClusterRole), err
}

func (c *FakeClusterRoles) List(label labels.Selector, field fields.Selector) (*authorizationapi.ClusterRoleList, error) {
	obj, err := c.Fake.Invokes(ktestclient.NewRootListAction("clusterroles", label, field), &authorizationapi.ClusterRoleList{})
	if obj == nil {
		return nil, err
	}

	return obj.(*authorizationapi.ClusterRoleList), err
}

func (c *FakeClusterRoles) Create(inObj *authorizationapi.ClusterRole) (*authorizationapi.ClusterRole, error) {
	obj, err := c.Fake.Invokes(ktestclient.NewRootCreateAction("clusterroles", inObj), inObj)
	if obj == nil {
		return nil, err
	}

	return obj.(*authorizationapi.ClusterRole), err
}

func (c *FakeClusterRoles) Update(inObj *authorizationapi.ClusterRole) (*authorizationapi.ClusterRole, error) {
	obj, err := c.Fake.Invokes(ktestclient.NewRootUpdateAction("clusterroles", inObj), inObj)
	if obj == nil {
		return nil, err
	}

	return obj.(*authorizationapi.ClusterRole), err
}

func (c *FakeClusterRoles) Delete(name string) error {
	_, err := c.Fake.Invokes(ktestclient.NewRootDeleteAction("clusterroles", name), &authorizationapi.ClusterRole{})
	return err
}
