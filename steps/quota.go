package steps

import (
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"

	"github.com/stretchr/testify/assert"
)

// registers all resource quota related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`I should have a (\w+) quota of (\d+) (.+?)$`, func(resourceType string, quantity int64, resourceName string) {
			quotas, err := c.GetResourceQuotas()
			if err != nil {
				c.Fail("Failed to get the Resource Quotas: %v", err)
				return
			}

			var found bool
			for _, quota := range quotas {
				switch resourceType {
				case "hard":
					for rName, rQuantity := range quota.Spec.Hard {
						if rName.String() == resourceName {
							found = true
							assert.Equal(c.T, quantity, rQuantity.Value())
							break
						}
					}
				default:
					c.Fail("Unknown resource quota type '%s'", resourceType)
					return
				}
			}
			if !found {
				c.Fail("Could not find a quota with the resource name '%s'", resourceName)
			}
		})

	})
}

// GetResourceQuotas gets all the ResourceQuotas, or returns an error
func (c *Context) GetResourceQuotas() ([]kapi.ResourceQuota, error) {
	_, kclient, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	quotas, err := kclient.ResourceQuotas(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	return quotas.Items, nil
}
