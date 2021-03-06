package steps

import (
	"os"

	templateapi "github.com/openshift/origin/pkg/template/api"

	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"

	"github.com/stretchr/testify/assert"
)

// registers all template related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.When(`^I create a new template for the file "(.+?)"$`, func(templateFileName string) {
			expandedTemplateFileName := os.ExpandEnv(templateFileName)
			if expandedTemplateFileName == "" {
				c.Fail("Template file name '%s' (expanded to '%s') is empty !", templateFileName, expandedTemplateFileName)
				return
			}
			if _, err := os.Stat(expandedTemplateFileName); err != nil {
				c.Fail("Template file '%s' (expanded to '%s') does not exists: %v", templateFileName, expandedTemplateFileName, err)
				return
			}

			r, err := c.ParseResource(expandedTemplateFileName)
			if err != nil {
				c.Fail("Failed to parse template file '%s' (expanded to '%s'): %v", templateFileName, expandedTemplateFileName, err)
				return
			}

			err = r.Visit(CreateResource)
			if err != nil {
				c.Fail("Failed to create template for file '%s' (expanded to '%s'): %v", templateFileName, expandedTemplateFileName, err)
				return
			}
		})

		c.Then(`^I should have a template "(.+?)"$`, func(templateName string) {
			template, err := c.GetTemplate(templateName)
			if err != nil {
				c.Fail("Failed to get Template '%s': %v", templateName, err)
				return
			}

			assert.Equal(c.T, templateName, template.Name)
		})

		c.Then(`^I should have a template "(.+?)" with (\d+) objects and (\d+) parameters$`, func(templateName string, expectedObjects int, expectedParameters int) {
			template, err := c.GetTemplate(templateName)
			if err != nil {
				c.Fail("Failed to get Template '%s': %v", templateName, err)
				return
			}

			assert.Equal(c.T, templateName, template.Name)
			assert.Equal(c.T, expectedObjects, len(template.Objects), "Template %s has %d objects, but expected number is %d !", template.Name, len(template.Objects), expectedObjects)
			assert.Equal(c.T, expectedParameters, len(template.Parameters), "Template %s has %d parameters, but expected number is %d !", template.Name, len(template.Parameters), expectedParameters)
		})

		c.Then(`^I should not have a template "(.+?)"$`, func(templateName string) {
			found, err := c.TemplateExists(templateName)
			if err != nil {
				c.Fail("Failed to check for template '%s' existance: %v", templateName, err)
				return
			}

			if found {
				c.Fail("Template %s should not exists", templateName)
				return
			}
		})

		c.Given(`^I have a template "(.+?)"$`, func(templateName string) {
			if _, err := c.GetTemplate(templateName); err != nil {
				c.Fail("Template '%s' does not exists: %v", templateName, err)
			}
		})

	})
}

// TemplateExists checks if a template with the given name exists.
func (c *Context) TemplateExists(tmplName string) (bool, error) {
	client, _, err := c.Clients()
	if err != nil {
		return false, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return false, err
	}

	tmplList, err := client.Templates(namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return false, err
	}

	for _, t := range tmplList.Items {
		if t.Name == tmplName {
			return true, nil
		}
	}
	return false, nil
}

// GetTemplate gets the Template with the given name, or returns an error
func (c *Context) GetTemplate(templateName string) (*templateapi.Template, error) {
	client, _, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	template, err := client.Templates(namespace).Get(templateName)
	if err != nil {
		return nil, err
	}

	return template, nil
}
