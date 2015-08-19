package steps

import (
	"errors"

	"github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"

	kclient "k8s.io/kubernetes/pkg/client"

	"github.com/lsegal/gucumber"
	"github.com/stretchr/testify/assert"
)

// Context shared by all steps
// Used to access the openshift client factory
// and the underlying gucumber context
type Context struct {
	*gucumber.Context

	factory   *clientcmd.Factory
	namespace string
}

// NewContext build a new context based on the given gucumber context
// It will register all known steps on the gucumber context
func NewContext(gc *gucumber.Context) *Context {
	c := &Context{
		Context: gc,
	}

	// register all steps with this context
	for _, registerer := range stepsRegisterers {
		registerer(c)
	}

	return c
}

// SetFactory stores a client factory in the context
func (c *Context) setFactory(factory *clientcmd.Factory) {
	c.factory = factory
}

// Factory returns the available client factory (if any)
// or return an error if no factory is available (meaning we are not logged in)
func (c *Context) Factory() (*clientcmd.Factory, error) {
	if c.factory == nil {
		return nil, errors.New("No factory (not logged in ?)")
	}
	return c.factory, nil
}

// SetNamespace stores the current namespace in the context
func (c *Context) setNamespace(namespace string) {
	c.namespace = namespace
}

// Namespace returns the current namespace (if defined)
// or return an error if no namespace is defined
func (c *Context) Namespace() (string, error) {
	if len(c.namespace) == 0 {
		return "", errors.New("No namespace defined !")
	}
	return c.namespace, nil
}

// Clients is a shortcut to the factory Clients
// It returns the openshift and k8s clients if available
// otherwise it returns an error (for example if we are not logged in)
func (c *Context) Clients() (*client.Client, *kclient.Client, error) {
	factory, err := c.Factory()
	if err != nil {
		return nil, nil, err
	}
	return factory.Clients()
}

// Fail fails the current step
// It will display the given message and optional arguments
// Note that it will not stop the step, but only record the failure
// so it is recommended to return from your step directly after calling this method
func (c *Context) Fail(msgAndArgs ...interface{}) bool {
	return assert.Fail(c.T, "", msgAndArgs...)
}
