package steps

import (
	"os"

	kapi "k8s.io/kubernetes/pkg/api"

	"github.com/stretchr/testify/assert"
)

// registers all endpoints related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`^I should have at least (\d+) endpoints "(.+?)"$`, func(minimumEndpoints int, epName string) {
			expandedEpName := os.ExpandEnv(epName)
			if len(expandedEpName) == 0 {
				c.Fail("Endpoints name '%s' (expanded to '%s') is empty !", epName, expandedEpName)
				return
			}

			ep, err := c.GetEndpoints(expandedEpName)
			if err != nil {
				c.Fail("Failed to get Endpoints '%s': %v", expandedEpName, err)
				return
			}

			endpoints := endpointsLen(ep)
			assert.True(c.T, endpoints >= minimumEndpoints,
				"Found %d endpoints for '%s' but expected at least %d", endpoints, expandedEpName, minimumEndpoints)
		})

	})
}

// GetEndpoints gets the Endpoints with the given name, or returns an error
func (c *Context) GetEndpoints(epName string) (*kapi.Endpoints, error) {
	_, kclient, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	ep, err := kclient.Endpoints(namespace).Get(epName)
	if err != nil {
		return nil, err
	}

	return ep, nil
}

// endpointsLen returns the number of endpoints
// (the Cartesian product of Addresses x Ports for each endpoint)
func endpointsLen(ep *kapi.Endpoints) int {
	var res int
	for _, subset := range ep.Subsets {
		res = res + len(subset.Addresses)*len(subset.Ports)
	}
	return res
}
