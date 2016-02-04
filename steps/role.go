package steps

import (
	"fmt"

	authapi "github.com/openshift/origin/pkg/authorization/api"

	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

// registers all role related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`The group "(.+?)" should have the "(.+?)" role$`, func(groupName string, roleName string) {
			groupHasRole, err := c.GroupHasRole(groupName, roleName)
			if err != nil {
				c.Fail("Failed to check the roles for group %s: %v", groupName, err)
				return
			}
			if !groupHasRole {
				c.Fail("The group '%s' does not have the '%s' role !", groupName, roleName)
			}
		})

		c.Then(`The user "(.+?)" should have the "(.+?)" role$`, func(userName string, roleName string) {
			userHasRole, err := c.UserHasRole(userName, roleName)
			if err != nil {
				c.Fail("Failed to check the roles for user %s: %v", userName, err)
				return
			}
			if !userHasRole {
				c.Fail("The user '%s' does not have the '%s' role !", userName, roleName)
			}
		})

		c.Then(`I should have the "(.+?)" role$`, func(roleName string) {
			userHasRole, err := c.UserHasRole("", roleName)
			if err != nil {
				c.Fail("Failed to check the roles for the current user: %v", err)
				return
			}
			if !userHasRole {
				c.Fail("The current user does not have the '%s' role !", roleName)
			}
		})

	})
}

// UserHasRole checks if the given user has the given role
// if the userName is empty, the current user will be used
func (c *Context) UserHasRole(userName string, roleName string) (bool, error) {
	roleBindings, err := c.GetRoleBindingsForRole(roleName)
	if err != nil {
		return false, err
	}
	if len(roleBindings) == 0 {
		return false, fmt.Errorf("Could not find a Role Binding for role '%s'", roleName)
	}

	if len(userName) == 0 {
		user, err := c.GetCurrentUser()
		if err != nil {
			return false, err
		}
		userName = user.Name
	}

	namespace, err := c.Namespace()
	if err != nil {
		return false, err
	}

	allUsers := []string{}
	for _, rb := range roleBindings {
		users, _ := authapi.StringSubjectsFor(namespace, rb.Subjects)
		allUsers = append(allUsers, users...)
	}

	return contains(userName, allUsers), nil
}

// GroupHasRole checks that the given group has the given role
func (c *Context) GroupHasRole(groupName string, roleName string) (bool, error) {
	roleBindings, err := c.GetRoleBindingsForRole(roleName)
	if err != nil {
		return false, err
	}
	if len(roleBindings) == 0 {
		return false, fmt.Errorf("Could not find a Role Binding for role '%s'", roleName)
	}

	namespace, err := c.Namespace()
	if err != nil {
		return false, err
	}

	allGroups := []string{}
	for _, rb := range roleBindings {
		_, groups := authapi.StringSubjectsFor(namespace, rb.Subjects)
		allGroups = append(allGroups, groups...)
	}

	return contains(groupName, allGroups), nil
}

// GetRoleBindingsForRole gets the RoleBindings with the given role name, or returns an error
func (c *Context) GetRoleBindingsForRole(roleName string) ([]authapi.RoleBinding, error) {
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

	roleBindings := []authapi.RoleBinding{}
	for _, rb := range rbList.Items {
		if rb.RoleRef.Name == roleName {
			roleBindings = append(roleBindings, rb)
		}
	}
	return roleBindings, nil
}

func contains(value string, elts []string) bool {
	for _, elt := range elts {
		if elt == value {
			return true
		}
	}
	return false
}
