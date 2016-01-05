package steps

import (
	"fmt"
	"net/http"

	"github.com/stretchr/testify/assert"
)

// registers all HTTP-check related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`^I should get an HTTP response code (\d+) on path "(.+?)" through the tunnel "(.+?)"$`, func(expectedResponseCode int, path string, tunnelName string) {
			tunnel := c.GetTunnel(tunnelName)
			if tunnel == nil {
				c.Fail("Could not find a tunnel named '%s'", tunnelName)
				return
			}

			url := fmt.Sprintf("http://%s:%v%s", "localhost", tunnel.LocalPort, path)
			resp, err := execHttpGetRequest(url)
			if err != nil {
				c.Fail("HTTP request on %s failed: %v", url, err)
				return
			}

			assert.Equal(c.T, expectedResponseCode, resp.StatusCode)
			resp.Body.Close()
		})

	})
}

// execHttpGetRequest executes an HTTP GET request on the given URL
// and returns the response or an error
func execHttpGetRequest(url string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
