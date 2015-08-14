package steps

import (
	"io/ioutil"
	"os"

	api "github.com/openshift/origin/pkg/api/latest"
	"github.com/openshift/origin/pkg/cmd/cli/cmd"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"

	kclientcmd "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd"
	kclientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	kcmdconfig "github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/config"

	"github.com/lsegal/gucumber"
	"github.com/stretchr/testify/assert"
)

// registers all login related steps
func init() {
	RegisterSteps(func(c *Context) {
		loginOptions := &cmd.LoginOptions{}

		c.Given(`^I have a username "(.+?)"$`, func(username string) {
			expandedUsername := os.ExpandEnv(username)
			if len(expandedUsername) == 0 {
				c.Fail("Username '%s' (expanded to '%s') is empty !", username, expandedUsername)
				return
			}
			loginOptions.Username = expandedUsername
		})

		c.And(`^I have a password "(.+?)"$`, func(password string) {
			expandedPassword := os.ExpandEnv(password)
			if len(expandedPassword) == 0 {
				c.Fail("Password '%s' (expanded to '%s') is empty !", password, expandedPassword)
				return
			}
			loginOptions.Password = expandedPassword
		})

		c.When(`^I login on "(.+?)"$`, func(host string) {
			expandedHost := os.ExpandEnv(host)
			if len(expandedHost) == 0 {
				c.Fail("Host '%s' (expanded to '%s') is empty !", host, expandedHost)
				return
			}
			loginOptions.Server = expandedHost

			factory, err := Login(loginOptions.Server, loginOptions.Username, loginOptions.Password)
			if err != nil {
				c.Fail("Failed to login: %v", err)
				return
			}

			c.setFactory(factory)
		})

		c.Then(`^I should be logged in as user "(.+?)"$`, func(username string) {
			expandedUsername := os.ExpandEnv(username)
			if len(expandedUsername) == 0 {
				c.Fail("Username '%s' (expanded to '%s') is empty !", username, expandedUsername)
				return
			}

			oclient, _, err := c.Clients()
			if err != nil {
				c.Fail("No clients available: %v", err)
				return
			}

			user, err := oclient.Users().Get("~")
			if err != nil {
				c.Fail("Could not get the current user: %v", err)
				return
			}

			assert.Equal(gucumber.T, expandedUsername, user.Name)
		})

		c.Then(`^I should have a token$`, func() {
			factory, err := c.Factory()
			if err != nil {
				c.Fail(err)
				return
			}

			config, err := factory.OpenShiftClientConfig.ClientConfig()
			if err != nil {
				c.Fail("No client config available: %v", err)
				return
			}

			if len(config.BearerToken) == 0 {
				c.Fail("No token")
				return
			}
		})
	})
}

// Login uses the given server/username/password to login on an openshift instance
//
// It returns an openshift client factory if successful, or an error
func Login(server string, username string, password string) (*clientcmd.Factory, error) {
	opts := &cmd.LoginOptions{
		Server:             server,
		Username:           username,
		Password:           password,
		InsecureTLS:        true,
		APIVersion:         api.Version,
		StartingKubeConfig: kclientcmdapi.NewConfig(),
		PathOptions:        kcmdconfig.NewDefaultPathOptions(),
		Out:                ioutil.Discard,
	}

	// perform the login, and store a new token in opts
	if err := opts.GatherInfo(); err != nil {
		return nil, err
	}

	// keep only what we need to initialize a factory
	config := kclientcmd.NewDefaultClientConfig(
		*kclientcmdapi.NewConfig(),
		&kclientcmd.ConfigOverrides{
			ClusterInfo: kclientcmdapi.Cluster{
				Server:                opts.Config.Host,
				APIVersion:            opts.Config.Version,
				InsecureSkipTLSVerify: opts.Config.Insecure,
			},
			AuthInfo: kclientcmdapi.AuthInfo{
				Token: opts.Config.BearerToken,
			},
			Context: kclientcmdapi.Context{},
		})

	factory := clientcmd.NewFactory(config)

	return factory, nil
}
