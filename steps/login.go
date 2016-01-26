package steps

import (
	"io/ioutil"
	"os"

	api "github.com/openshift/origin/pkg/api/latest"
	"github.com/openshift/origin/pkg/cmd/cli/cmd"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"

	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	kclientcmd "k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	kclientcmdapi "k8s.io/kubernetes/pkg/client/unversioned/clientcmd/api"
	kcmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"

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

		c.And(`^I have a token "(.+?)"$`, func(token string) {
			expandedToken := os.ExpandEnv(token)
			if len(expandedToken) == 0 {
				c.Fail("Token '%s' (expanded to '%s') is empty !", token, expandedToken)
				return
			}
			loginOptions.Token = expandedToken
		})

		c.When(`^I login on "(.+?)"$`, func(host string) {
			expandedHost := os.ExpandEnv(host)
			if len(expandedHost) == 0 {
				c.Fail("Host '%s' (expanded to '%s') is empty !", host, expandedHost)
				return
			}
			loginOptions.Server = expandedHost

			var config *kclient.Config
			var err error
			if len(loginOptions.Token) > 0 {
				config, err = ValidateToken(loginOptions.Server, loginOptions.Token)
				if err != nil {
					c.Fail("Failed to validate token: %v", err)
					return
				}
			} else {
				config, err = Login(loginOptions.Server, loginOptions.Username, loginOptions.Password)
				if err != nil {
					c.Fail("Failed to login: %v", err)
					return
				}
			}

			factory := NewFactory(config)
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
// It returns a client config if successful, or an error
func Login(server string, username string, password string) (*kclient.Config, error) {
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

	return opts.Config, nil
}

// ValidateToken validates that the given token is valid on the given server
//
// It returns a client config if successful, or an error
func ValidateToken(server string, token string) (*kclient.Config, error) {
	opts := &cmd.LoginOptions{
		Server:             server,
		Token:              token,
		InsecureTLS:        true,
		APIVersion:         api.Version,
		StartingKubeConfig: kclientcmdapi.NewConfig(),
		PathOptions:        kcmdconfig.NewDefaultPathOptions(),
		Out:                ioutil.Discard,
	}

	// check the token
	if err := opts.GatherInfo(); err != nil {
		return nil, err
	}

	return opts.Config, nil
}

// NewFactory builds a new openshift client factory from the given config
func NewFactory(config *kclient.Config) *clientcmd.Factory {
	// keep only what we need to initialize a factory
	clientConfig := kclientcmd.NewDefaultClientConfig(
		*kclientcmdapi.NewConfig(),
		&kclientcmd.ConfigOverrides{
			ClusterInfo: kclientcmdapi.Cluster{
				Server:                config.Host,
				APIVersion:            config.Version,
				InsecureSkipTLSVerify: config.Insecure,
			},
			AuthInfo: kclientcmdapi.AuthInfo{
				Token: config.BearerToken,
			},
			Context: kclientcmdapi.Context{},
		})

	factory := clientcmd.NewFactory(clientConfig)

	return factory
}
