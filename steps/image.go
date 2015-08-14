package steps

import (
	imageapi "github.com/openshift/origin/pkg/image/api"

	"github.com/stretchr/testify/assert"
)

// registers all images related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`^I should have an imagestream "(.+?)"$`, func(isName string) {
			is, err := c.GetImageStream(isName)
			if err != nil {
				c.Fail("Failed to get Image Stream '%s': %v", isName, err)
				return
			}

			assert.Equal(c.T, isName, is.Name)
		})

	})
}

// GetImageStream gets the ImageStream with the given name, or returns an error
func (c *Context) GetImageStream(isName string) (*imageapi.ImageStream, error) {
	client, _, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	is, err := client.ImageStreams(namespace).Get(isName)
	if err != nil {
		return nil, err
	}

	return is, nil
}
