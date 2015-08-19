package steps

import (
	kapi "k8s.io/kubernetes/pkg/api"

	"github.com/stretchr/testify/assert"
)

// registers all service account related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`the serviceaccount "(.+?)" should have a secret "(.+?)"$`, func(saName string, secretName string) {
			sa, err := c.GetServiceAccount(saName)
			if err != nil {
				c.Fail("Failed to get Service Account '%s': %v", saName, err)
				return
			}

			assert.Equal(c.T, saName, sa.Name)

			var found bool
			for _, secret := range sa.Secrets {
				if secret.Name == secretName {
					found = true
					break
				}
			}
			if !found {
				c.Fail("The service account '%s' has no secret '%s' !", saName, secretName)
			}
		})

	})
}

// GetServiceAccount gets the ServiceAccount with the given name, or returns an error
func (c *Context) GetServiceAccount(saName string) (*kapi.ServiceAccount, error) {
	_, kclient, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	sa, err := kclient.ServiceAccounts(namespace).Get(saName)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
