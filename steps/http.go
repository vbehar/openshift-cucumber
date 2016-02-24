package steps

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
)

// registers all HTTP-check related steps
func init() {
	var currentResp *http.Response
	requestHeaders := make(http.Header)

	RegisterSteps(func(c *Context) {

		c.Then(`^I should get an HTTP response code (\d+) on path "(.+?)" through the tunnel "(.+?)"$`, func(expectedResponseCode int, path string, tunnelName string) {
			tunnel := c.GetTunnel(tunnelName)
			if tunnel == nil {
				c.Fail("Could not find a tunnel named '%s'", tunnelName)
				return
			}

			url := fmt.Sprintf("http://%s:%v%s", "localhost", tunnel.LocalPort, path)
			resp, err := c.execHttpGetRequest(url, requestHeaders)
			currentResp = resp
			if err != nil {
				c.Fail("HTTP request on %s failed: %v", url, err)
				return
			}

			assert.Equal(c.T, expectedResponseCode, resp.StatusCode)
			resp.Body.Close()
		})

		c.Then(`^This response has header "(.+?)" equals to "(.+?)"$`, func(name, expectedValue string) {
			assert.Equal(c.T, expectedValue, currentResp.Header.Get(name))
		})

		c.Then(`^This response has no header "(.+?)"$`, func(name string) {
			assert.NotContains(c.T, name, currentResp.Header)
		})

		c.Then(`^I should have the text "(.+?)" on path "(.+?)" through the tunnel "(.+?)"$`, func(expectedContentText string, path string, tunnelName string) {
			tunnel := c.GetTunnel(tunnelName)
			if tunnel == nil {
				c.Fail("Could not find a tunnel named '%s'", tunnelName)
				return
			}

			url := fmt.Sprintf("http://%s:%v%s", "localhost", tunnel.LocalPort, path)
			resp, err := c.execHttpGetRequest(url, requestHeaders)
			if err != nil {
				c.Fail("HTTP request on %s failed: %v", url, err)
				return
			}
			data, err := ioutil.ReadAll(resp.Body)

			assert.Contains(c.T, string(data), expectedContentText)
			resp.Body.Close()
		})

		c.Given(`^Using request header "(.+?)": "(.+?)"$`, func(name, value string) {
			for _, v := range strings.Split(value, ";") {
				requestHeaders.Add(name, v)
			}
		})

	})
}

// execHttpGetRequest executes an HTTP GET request on the given URL
// and returns the response or an error
// It uses an exponential backoff retry
func (c *Context) execHttpGetRequest(url string, headers http.Header) (*http.Response, error) {
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
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	var resp *http.Response
	err = c.ExecWithExponentialBackoff(func() (err error) {
		resp, err = client.Do(req)
		if err == nil {
			if resp.StatusCode >= 500 {
				err = fmt.Errorf("Invalid status: %s", resp.Status)
			}
		}
		return
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// See 2 (end of page 4) http://www.ietf.org/rfc/rfc2617.txt
// "To receive authorization, the client sends the userid and password,
// separated by a single colon (":") character, within a base64
// encoded string in the credentials."
// It is not meant to be urlencoded.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
