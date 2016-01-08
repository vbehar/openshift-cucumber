package steps

import (
	"time"

	projectapi "github.com/openshift/origin/pkg/project/api"

	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

// registers all project related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Given(`^There is no project "(.+?)"$`, func(projectName string) {
			found, err := c.ProjectExists(projectName)
			if err != nil {
				c.Fail("Failed to check for project '%s' existance: %v", projectName, err)
				return
			}

			if found {
				c.DeleteProject(projectName)

				stillPresent, err := c.ProjectExists(projectName)
				if err != nil {
					c.Fail("Failed to check for project '%s' existance: %v", projectName, err)
					return
				}

				if stillPresent {
					c.Fail("Failed to delete existing project %s", projectName)
					return
				}
			}
		})

		c.Given(`^I have an existing project "(.+?)"$`, func(projectName string) {
			found, err := c.ProjectExists(projectName)
			if err != nil {
				c.Fail("Failed to check for project '%s' existance: %v", projectName, err)
				return
			}

			if !found {
				if err := c.CreateNewProject(projectName); err != nil {
					c.Fail("Failed to create new project %s", projectName)
					return
				}
			}
		})

		c.Given(`^My current project is "(.+?)"$`, func(projectName string) {
			found, err := c.ProjectExists(projectName)
			if err != nil {
				c.Fail("Failed to check for project '%s' existance: %v", projectName, err)
				return
			}

			if !found {
				c.Fail("Project '%s' does not exists", projectName)
				return
			}

			c.setNamespace(projectName)
		})

		c.When(`^I create a new project "(.+?)"$`, func(projectName string) {
			if err := c.CreateNewProject(projectName); err != nil {
				c.Fail("Failed to create new project %s", projectName)
			}
		})

		c.When(`^I delete the project "(.+?)"$`, func(projectName string) {
			if err := c.DeleteProject(projectName); err != nil {
				c.Fail("Failed to delete project %s", projectName)
			}
		})

		c.Then(`^I should have a project "(.+?)"$`, func(projectName string) {
			found, err := c.ProjectExists(projectName)
			if err != nil {
				c.Fail("Failed to check for project '%s' existance: %v", projectName, err)
				return
			}

			if !found {
				c.Fail("Failed to find a project named %s", projectName)
				return
			}
		})

		c.Then(`^I should not have a project "(.+?)"$`, func(projectName string) {
			found, err := c.ProjectExists(projectName)
			if err != nil {
				c.Fail("Failed to check for project '%s' existance: %v", projectName, err)
				return
			}

			if found {
				c.Fail("Project %s should not exists", projectName)
				return
			}
		})

	})
}

// ProjectExists checks if a project with the given name exists.
func (c *Context) ProjectExists(projectName string) (bool, error) {
	client, _, err := c.Clients()
	if err != nil {
		return false, err
	}

	projectList, err := client.Projects().List(labels.Everything(), fields.Everything())
	if err != nil {
		return false, err
	}

	for _, p := range projectList.Items {
		if p.Name == projectName {
			return true, nil
		}
	}
	return false, nil
}

// DeleteProject deletes the project with the given name, or returns an error
func (c *Context) DeleteProject(projectName string) error {
	client, _, err := c.Clients()
	if err != nil {
		return err
	}

	if err = client.Projects().Delete(projectName); err != nil {
		return err
	}

	// FIXME wait a little to make sure the project has been deleted
	time.Sleep(1 * time.Second)
	return nil
}

// CreateNewProject creates a new project with the given name, or returns an error
func (c *Context) CreateNewProject(projectName string) error {
	client, _, err := c.Clients()
	if err != nil {
		return err
	}

	projectRequest := &projectapi.ProjectRequest{}
	projectRequest.Name = projectName
	if _, err = client.ProjectRequests().Create(projectRequest); err != nil {
		return err
	}

	// FIXME wait a little to make sure the project has been created
	time.Sleep(1 * time.Second)
	return nil
}
