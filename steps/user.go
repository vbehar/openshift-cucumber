package steps

import userapi "github.com/openshift/origin/pkg/user/api"

// GetCurrentUser gets the current User, or returns an error
func (c *Context) GetCurrentUser() (*userapi.User, error) {
	return c.GetUser("~")
}

// GetUser gets the User with the given name, or returns an error
func (c *Context) GetUser(userName string) (*userapi.User, error) {
	oclient, _, err := c.Clients()
	if err != nil {
		return nil, err
	}

	user, err := oclient.Users().Get(userName)
	if err != nil {
		return nil, err
	}

	return user, nil
}
