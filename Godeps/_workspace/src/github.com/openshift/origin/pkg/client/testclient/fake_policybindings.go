package testclient

import (
	ktestclient "k8s.io/kubernetes/pkg/client/unversioned/testclient"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"

	authorizationapi "github.com/openshift/origin/pkg/authorization/api"
)

// FakePolicyBindings implements PolicyBindingInterface. Meant to be embedded into a struct to get a default
// implementation. This makes faking out just the methods you want to test easier.
type FakePolicyBindings struct {
	Fake      *Fake
	Namespace string
}

func (c *FakePolicyBindings) Get(name string) (*authorizationapi.PolicyBinding, error) {
	obj, err := c.Fake.Invokes(ktestclient.NewGetAction("policybindings", c.Namespace, name), &authorizationapi.PolicyBinding{})
	if obj == nil {
		return nil, err
	}

	return obj.(*authorizationapi.PolicyBinding), err
}

func (c *FakePolicyBindings) List(label labels.Selector, field fields.Selector) (*authorizationapi.PolicyBindingList, error) {
	obj, err := c.Fake.Invokes(ktestclient.NewListAction("policybindings", c.Namespace, label, field), &authorizationapi.PolicyBindingList{})
	if obj == nil {
		return nil, err
	}

	return obj.(*authorizationapi.PolicyBindingList), err
}

func (c *FakePolicyBindings) Create(inObj *authorizationapi.PolicyBinding) (*authorizationapi.PolicyBinding, error) {
	obj, err := c.Fake.Invokes(ktestclient.NewCreateAction("policybindings", c.Namespace, inObj), inObj)
	if obj == nil {
		return nil, err
	}

	return obj.(*authorizationapi.PolicyBinding), err
}

func (c *FakePolicyBindings) Delete(name string) error {
	_, err := c.Fake.Invokes(ktestclient.NewDeleteAction("policybindings", c.Namespace, name), &authorizationapi.PolicyBinding{})
	return err
}

func (c *FakePolicyBindings) Watch(label labels.Selector, field fields.Selector, resourceVersion string) (watch.Interface, error) {
	return c.Fake.InvokesWatch(ktestclient.NewWatchAction("policybindings", c.Namespace, label, field, resourceVersion))
}
