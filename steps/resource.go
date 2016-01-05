package steps

import (
	"os"

	"k8s.io/kubernetes/pkg/api/validation"
	"k8s.io/kubernetes/pkg/kubectl/resource"
)

// registers all resource related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.When(`^I parse the file "(.+?)"$`, func(fileName string) {
			expandedFileName := os.ExpandEnv(fileName)
			if expandedFileName == "" {
				c.Fail("File name '%s' (expanded to '%s') is empty !", fileName, expandedFileName)
				return
			}
			if _, err := os.Stat(expandedFileName); err != nil {
				c.Fail("File '%s' (expanded to '%s') does not exists: %v", fileName, expandedFileName, err)
				return
			}

			_, err := c.ParseResource(expandedFileName)
			if err != nil {
				c.Fail("Failed to parse file '%s' (expanded to '%s'): %v", fileName, expandedFileName, err)
				return
			}
		})

		c.When(`^I create resources from the file "(.+?)"$`, func(fileName string) {
			expandedFileName := os.ExpandEnv(fileName)
			if expandedFileName == "" {
				c.Fail("File name '%s' (expanded to '%s') is empty !", fileName, expandedFileName)
				return
			}
			if _, err := os.Stat(expandedFileName); err != nil {
				c.Fail("File '%s' (expanded to '%s') does not exists: %v", fileName, expandedFileName, err)
				return
			}

			r, err := c.ParseResource(expandedFileName)
			if err != nil {
				c.Fail("Failed to parse file '%s' (expanded to '%s'): %v", fileName, expandedFileName, err)
				return
			}

			if err = r.Visit(CreateResource); err != nil {
				c.Fail("Failed to create resource from file '%s' (expanded to '%s'): %v", fileName, expandedFileName, err)
				return
			}
		})

	})
}

// ParseResource parses the resource stored in the given file,
// and returns the Result or an error.
//
// If you need to create resources on openshift, after parsing you will need
// to use the visitor pattern, and the CreateResource function.
func (c *Context) ParseResource(fileName string) (*resource.Result, error) {
	factory, err := c.Factory()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	mapper, typer := factory.Object()
	clientMapper := factory.ClientMapperForCommand()

	r := resource.
		NewBuilder(mapper, typer, clientMapper).
		Schema(validation.NullSchema{}).
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(true, fileName).
		Flatten().
		Do()

	if err = r.Err(); err != nil {
		return nil, err
	}

	return r, nil
}

// CreateResource creates the given resource on openshift
// and returns an error, or nil if successful
//
// Usage: first parse the resource with Context.ParseResource
// and then use the visitor pattern on the parsed resource:
//   r.Visit(CreateResource)
func CreateResource(info *resource.Info) error {
	data, err := info.Mapping.Codec.Encode(info.Object)
	if err != nil {
		return err
	}

	obj, err := resource.NewHelper(info.Client, info.Mapping).Create(info.Namespace, true, data)
	if err != nil {
		return err
	}

	info.Refresh(obj, true)
	return nil
}
