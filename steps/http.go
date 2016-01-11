package steps

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

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
			resp, err := c.execHttpGetRequest(url)
			if err != nil {
				c.Fail("HTTP request on %s failed: %v", url, err)
				return
			}

			assert.Equal(c.T, expectedResponseCode, resp.StatusCode)
			resp.Body.Close()
		})

		c.Then(`^I should have the text "(.+?)" on path "(.+?)" through the tunnel "(.+?)"$`, func(expectedContentText string, path string, tunnelName string) {
			tunnel := c.GetTunnel(tunnelName)
			if tunnel == nil {
				c.Fail("Could not find a tunnel named '%s'", tunnelName)
				return
			}

			url := fmt.Sprintf("http://%s:%v%s", "localhost", tunnel.LocalPort, path)
			resp, err := c.execHttpGetRequest(url)
			if err != nil {
				c.Fail("HTTP request on %s failed: %v", url, err)
				return
			}
			data, err := ioutil.ReadAll(resp.Body)

			assert.Contains(c.T, string(data), expectedContentText)
			resp.Body.Close()
		})

	})
}

// execHttpGetRequest executes an HTTP GET request on the given URL
// and returns the response or an error
// It uses an exponential backoff retry
func (c *Context) execHttpGetRequest(url string) (*http.Response, error) {
	transport := &http.Transport{
		DisableKeepAlives:     true,
		MaxIdleConnsPerHost:   5,
		ResponseHeaderTimeout: 10 * time.Second,
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var resp *http.Response
	err = c.ExecWithExponentialBackoff(func() error {
		var err error
		resp, err = client.Do(req)
		return err
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}
