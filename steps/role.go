package steps

import (
	authapi "github.com/openshift/origin/pkg/authorization/api"

	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

// registers all role related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`I should have the "(.+?)" role$`, func(roleName string) {
			rb, err := c.GetRoleBindingForRole(roleName)
			if err != nil {
				c.Fail("Failed to get Role Binding for role '%s': %v", roleName, err)
				return
			}
			if rb == nil {
				c.Fail("Could not find a Role Binding for role '%s'", roleName)
				return
			}

			user, err := c.GetCurrentUser()
			if err != nil {
				c.Fail("Failed to get the current User: %v", err)
				return
			}

			if !rb.Users.Has(user.Name) {
				c.Fail("The current user '%s' does not have the '%s' role !", user.Name, roleName)
			}
		})

	})
}

// GetRoleBinding gets the RoleBinding with the given role name, or returns an error
func (c *Context) GetRoleBindingForRole(roleName string) (*authapi.RoleBinding, error) {
	oclient, _, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	rbList, err := oclient.RoleBindings(namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return nil, err
	}
	for _, rb := range rbList.Items {
		if rb.RoleRef.Name == roleName {
			return &rb, nil
		}
	}

	return nil, nil
}
