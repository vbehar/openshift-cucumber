package steps

import (
	"os"

	kapi "k8s.io/kubernetes/pkg/api"

	"github.com/stretchr/testify/assert"
)

// registers all service related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`^I should have a service "(.+?)"$`, func(serviceName string) {
			expandedServiceName := os.ExpandEnv(serviceName)
			if len(expandedServiceName) == 0 {
				c.Fail("Service name '%s' (expanded to '%s') is empty !", serviceName, expandedServiceName)
				return
			}

			service, err := c.GetService(expandedServiceName)
			if err != nil {
				c.Fail("Failed to get Service '%s': %v", expandedServiceName, err)
				return
			}

			assert.Equal(c.T, expandedServiceName, service.Name)
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
