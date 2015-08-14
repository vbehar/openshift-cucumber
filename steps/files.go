package steps

import (
	"os"

	"github.com/lsegal/gucumber"
	"github.com/stretchr/testify/assert"
)

// registers all files related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Given(`^I have a file "(.+?)"$`, func(fileName string) {
			expandedFileName := os.ExpandEnv(fileName)
			if expandedFileName == "" {
				assert.Fail(gucumber.T, "Empty file name", "File name '%s' (expanded to '%s') is empty !", fileName, expandedFileName)
			}
			if _, err := os.Stat(expandedFileName); err != nil {
				assert.Fail(gucumber.T, "File existance", "File '%s' (expanded to '%s') does not exists: %v", fileName, expandedFileName, err)
			}
		})

	})
}
