package steps

import (
	"os"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/validation"
	"k8s.io/kubernetes/pkg/kubectl"
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

		c.When(`^I delete all resources with "(.+?)"$`, func(selector string) {
			err := c.DeleteResourcesBySelector(selector)
			if err != nil {
				c.Fail("Failed to delete resources with '%s': %v", selector, err)
				return
			}
		})

	})
}

// DeleteResourcesBySelector deletes all resources matching the given label selector
func (c *Context) DeleteResourcesBySelector(selector string) error {
	factory, err := c.Factory()
	if err != nil {
		return err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return err
	}

	mapper, typer := factory.Object()
	clientMapper := factory.ClientMapperForCommand()

	r := resource.
		NewBuilder(mapper, typer, clientMapper).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(true).
		SelectorParam(selector).
		SelectAllParam(false).
		ResourceTypeOrNameArgs(false, "template,buildconfig,build,imagestream,deploymentconfig,replicationcontroller,pod,service,route").
		RequireObject(false).
		Flatten().
		Do()

	if err = r.Err(); err != nil {
		return err
	}

	return r.Visit(func(info *resource.Info) error {
		reaper, err := factory.Reaper(info.Mapping)
		if err != nil {
			if kubectl.IsNoSuchReaperError(err) {
				return resource.NewHelper(info.Client, info.Mapping).Delete(info.Namespace, info.Name)
			}
			return err
		}

		_, err = reaper.Stop(info.Namespace, info.Name, 5*time.Second, api.NewDeleteOptions(int64(0)))
		return err
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
		NamespaceParam(namespace).DefaultNamespace().RequireNamespace().
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
