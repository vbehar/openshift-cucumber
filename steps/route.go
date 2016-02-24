package steps

import (
	"fmt"
	"net/http"

	routeapi "github.com/openshift/origin/pkg/route/api"

	"github.com/stretchr/testify/assert"
)

// registers all route related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`^I should have a route "(.+?)"$`, func(routeName string) {
			route, err := c.GetRoute(routeName)
			if err != nil {
				c.Fail("Failed to get Route '%s': %v", routeName, err)
				return
			}

			assert.Equal(c.T, routeName, route.Name)
		})

		c.Then(`^I can access the application through the route "(.+?)"$`, func(routeName string) {
			route, err := c.GetRoute(routeName)
			if err != nil {
				c.Fail("Failed to get Route '%s': %v", routeName, err)
				return
			}
			if len(route.Spec.Host) == 0 {
				c.Fail("The Route '%s' has no host !", routeName)
				return
			}

			host := fmt.Sprintf("http://%s/", route.Spec.Host)
			resp, err := c.execHttpGetRequest(host, make(http.Header))
			if err != nil {
				c.Fail("Failed to access the route '%s' at %s: %v", routeName, host, err)
				return
			}

			assert.True(c.T, resp.StatusCode >= 200 && resp.StatusCode < 400, "Status code should be either 2xx or 3xx, but it is %d", resp.StatusCode)
			resp.Body.Close()
		})

		c.Then(`^I can access the application with the credentials "(.+?)":"(.+?)" through the route "(.+?)"$`, func(login string, password string, routeName string) {
			route, err := c.GetRoute(routeName)
			if err != nil {
				c.Fail("Failed to get Route '%s': %v", routeName, err)
				return
			}
			if len(route.Spec.Host) == 0 {
				c.Fail("The Route '%s' has no host !", routeName)
				return
			}

			host := fmt.Sprintf("http://%s/", route.Spec.Host)
			requestHeaders := make(http.Header)
			requestHeaders.Set("Authorization", "Basic "+basicAuth(login, password))

			resp, err := c.execHttpGetRequest(host, requestHeaders)
			if err != nil {
				c.Fail("Failed to access the route '%s' at %s: %v", routeName, host, err)
				return
			}

			assert.True(c.T, resp.StatusCode >= 200 && resp.StatusCode < 400, "Status code should be either 2xx or 3xx, but it is %d", resp.StatusCode)
			resp.Body.Close()
		})

	})
}

// GetRoute gets the Route with the given name, or returns an error
func (c *Context) GetRoute(routeName string) (*routeapi.Route, error) {
	client, _, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	route, err := client.Routes(namespace).Get(routeName)
	if err != nil {
		return nil, err
	}

	return route, nil
}
