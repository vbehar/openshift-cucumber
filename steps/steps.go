// Package steps provides cucumber steps implementations on top of the OpenShift API
package steps

import (
	"log"
	"os"

	"github.com/openshift/origin/pkg/cmd/util/clientcmd"

	"github.com/spf13/pflag"
)

const (
	// Name of the env var that contains the OpenShift server used to login
	// example: "https://localhost:8443"
	OpenShiftServerEnvVarName = "OPENSHIFT_HOST"

	// Name of the env var that contains the OpenShift username used to login
	OpenShiftUsernameEnvVarName = "OPENSHIFT_USER"

	// Name of the env var that contains the OpenShift password used to login
	OpenShiftPasswordEnvVarName = "OPENSHIFT_PASSWD"
)

// StepsRegisterer allows to register steps on a Context
// using the context Given/When/Then methods
type StepsRegisterer func(c *Context)

// stepsRegisterers contains all the registerers we should call
// in order to register all the steps on a given Context
var stepsRegisterers []StepsRegisterer

// RegisterSteps allows to register steps on a Context
// using the context Given/When/Then methods
func RegisterSteps(registerer StepsRegisterer) {
	stepsRegisterers = append(stepsRegisterers, registerer)
}

// register the tag handlers
func init() {
	RegisterSteps(func(c *Context) {

		c.Before("@offline", func() {
			c.setNamespace("offline")
			c.setFactory(func() *clientcmd.Factory {
				flags := pflag.NewFlagSet("openshift-cucumber", pflag.ContinueOnError)
				return clientcmd.New(flags)
			}())
		})

		// @loggedInFromEnvVars performs a login using server/username/password from env vars
		// it sets a factory on the context, ready to be used by other steps
		c.Before("@loggedInFromEnvVars", func() {
			server := os.Getenv(OpenShiftServerEnvVarName)
			username := os.Getenv(OpenShiftUsernameEnvVarName)
			password := os.Getenv(OpenShiftPasswordEnvVarName)

			factory, err := Login(server, username, password)
			if err != nil {
				log.Fatalf("Could not loggin on server '%s' (from env var '%s') with username '%s' (from env var '%s'): %v",
					server, OpenShiftServerEnvVarName, username, OpenShiftUsernameEnvVarName, err)
			}
			c.setFactory(factory)
		})

	})
}
