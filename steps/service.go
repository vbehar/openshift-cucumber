package steps

import (
	kapi "k8s.io/kubernetes/pkg/api"

	"github.com/stretchr/testify/assert"
)

// registers all service related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`^I should have a service "(.+?)"$`, func(serviceName string) {
			service, err := c.GetService(serviceName)
			if err != nil {
				c.Fail("Failed to get Service '%s': %v", serviceName, err)
				return
			}

			assert.Equal(c.T, serviceName, service.Name)
		})

	})
}

// GetService gets the Service with the given name, or returns an error
func (c *Context) GetService(serviceName string) (*kapi.Service, error) {
	_, kclient, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	service, err := kclient.Services(namespace).Get(serviceName)
	if err != nil {
		return nil, err
	}

	return service, nil
}
