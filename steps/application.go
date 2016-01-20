package steps

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	configcmd "github.com/openshift/origin/pkg/config/cmd"
	"github.com/openshift/origin/pkg/generate/app/cmd"

	kapi "k8s.io/kubernetes/pkg/api"
)

// registers all new-app related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.When(`^I create a new application based on the template "(.+?)"$`, func(templateName string) {
			if _, errs := c.NewAppFromTemplate(templateName, []string{}); len(errs) > 0 {
				c.Fail("Failed to create a new application based on the template '%s': %v", templateName, errs)
				return
			}
		})

		c.When(`^I create a new application based on the template "(.+?)" with parameters "(.+?)"$`, func(templateName string, parameters string) {
			parametersList, err := parseParameters(parameters)
			if err != nil {
				c.Fail("Failed to parse parameters '%s': %v", parameters, err)
				return
			}

			// expand the parameters values
			templateParameters := []string{}
			for k, v := range parametersList {
				v = os.ExpandEnv(v)
				templateParameters = append(templateParameters, fmt.Sprintf("%s=%s", k, v))
			}

			if _, errs := c.NewAppFromTemplate(templateName, templateParameters); len(errs) > 0 {
				c.Fail("Failed to create a new application based on the template '%s': %v", templateName, errs)
				return
			}
		})

	})
}

// NewAppFromTemplate creates a new application from the given template and parameters
// and returns the list of objects created, or the errors
//
// The template referenced should already have been created
func (c *Context) NewAppFromTemplate(templateName string, templateParameters []string) (*kapi.List, []error) {
	factory, err := c.Factory()
	if err != nil {
		return nil, []error{err}
	}

	client, _, err := factory.Clients()
	if err != nil {
		return nil, []error{err}
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, []error{err}
	}

	mapper, typer := factory.Object()
	clientMapper := factory.ClientMapperForCommand()

	appConfig := cmd.NewAppConfig(typer, mapper, clientMapper)
	appConfig.SetOpenShiftClient(client, namespace)
	appConfig.Components = append(appConfig.Components, templateName)
	if len(templateParameters) > 0 {
		appConfig.TemplateParameters = append(appConfig.TemplateParameters, templateParameters...)
	}

	// parse
	appResult, err := appConfig.RunAll(os.Stdout, os.Stderr)
	if err != nil {
		return nil, []error{err}
	}

	// and create all objects
	bulk := configcmd.Bulk{
		Mapper:            mapper,
		Typer:             typer,
		RESTClientFactory: factory.RESTClient,
	}
	if errs := bulk.Create(appResult.List, appResult.Namespace); len(errs) != 0 {
		return nil, errs
	}

	return appResult.List, []error{}
}

var parameterRegexp = regexp.MustCompile("^([\\w\\-_]+)\\=(.*)$")

func parseParameters(parameters string) (map[string]string, error) {
	ret := map[string]string{}
	for _, parameter := range strings.Split(parameters, ",") {
		switch matches := parameterRegexp.FindStringSubmatch(parameter); len(matches) {
		case 3:
			k, v := matches[1], matches[2]
			ret[k] = v
		default:
			return map[string]string{}, fmt.Errorf("Parameter '%s' should match the format 'key=value'", parameter)
		}

	}
	return ret, nil
}
