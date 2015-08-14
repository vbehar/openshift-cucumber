package steps

import (
	"fmt"
	"os"

	"github.com/openshift/origin/pkg/cmd/cli/secrets"

	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"

	"github.com/stretchr/testify/assert"
)

// registers all secret related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.When(`^I create a new secret "(.+?)" with "(.+?)"="(.+?)"$`, func(secretName string, keyName string, fileName string) {
			expandedFileName := os.ExpandEnv(fileName)
			if expandedFileName == "" {
				c.Fail("File name '%s' (expanded to '%s') is empty !", fileName, expandedFileName)
				return
			}
			if _, err := os.Stat(expandedFileName); err != nil {
				c.Fail("File '%s' (expanded to '%s') does not exists: %v", fileName, expandedFileName, err)
				return
			}

			secretSources := []string{fmt.Sprintf("%s=%s", keyName, expandedFileName)}
			_, err := c.CreateSecret(secretName, kapi.SecretTypeOpaque, secretSources)
			if err != nil {
				c.Fail("Failed to create secret '%s': %v", secretName, err)
				return
			}
		})

		c.When(`^I create a new dockercfg secret "(.+?)" for "(.+?)"$`, func(secretName string, fileName string) {
			expandedFileName := os.ExpandEnv(fileName)
			if expandedFileName == "" {
				c.Fail("File name '%s' (expanded to '%s') is empty !", fileName, expandedFileName)
				return
			}
			if _, err := os.Stat(expandedFileName); err != nil {
				c.Fail("File '%s' (expanded to '%s') does not exists: %v", fileName, expandedFileName, err)
				return
			}

			// it's a dockercfg secret, so it requires a specific key
			secretSources := []string{fmt.Sprintf("%s=%s", kapi.DockerConfigKey, expandedFileName)}
			_, err := c.CreateSecret(secretName, kapi.SecretTypeDockercfg, secretSources)
			if err != nil {
				c.Fail("Failed to create secret '%s': %v", secretName, err)
				return
			}
		})

		c.Then(`I should have a secret "(.+?)" of type "(.+?)" with a key "(.+?)"`, func(secretName string, secretType string, keyName string) {
			secret, err := c.GetSecret(secretName)
			if err != nil {
				c.Fail("Failed to get secret '%s': %v", secretName, err)
				return
			}

			assert.Equal(c.T, secretType, string(secret.Type), "Secret %s expected to have type %s, but has type %s", secretName, secretType, secret.Type)

			if _, ok := secret.Data[keyName]; !ok {
				c.Fail("No key '%s' in secret '%s' !", keyName, secretName)
				return
			}
		})

		c.When(`I have a secret "(.+?)"`, func(secretName string) {
			if _, err := c.GetSecret(secretName); err != nil {
				c.Fail("Could not find secret '%s': %v", secretName, err)
			}
		})

		c.When(`I add the secret "(.+?)" to the serviceaccount "(.+?)"`, func(secretName string, serviceAccountName string) {
			if err := c.AddSecretToServiceAccount(secretName, serviceAccountName); err != nil {
				c.Fail("Failed to add secret '%s' to the service account '%s': %v", secretName, serviceAccountName, err)
			}
		})

	})
}

// GetSecret gets the Secret with the given name, or returns an error
func (c *Context) GetSecret(secretName string) (*kapi.Secret, error) {
	factory, err := c.Factory()
	if err != nil {
		return nil, err
	}

	_, kclient, err := factory.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	secret, err := kclient.Secrets(namespace).Get(secretName)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// CreateSecret creates a new secret with the given name and content (sources).
//
// The most common secret types are SecretTypeOpaque and SecretTypeDockercfg.
//
// It returns the newly created secret, or an error.
func (c *Context) CreateSecret(secretName string, secretType kapi.SecretType, sources []string) (*kapi.Secret, error) {
	factory, err := c.Factory()
	if err != nil {
		return nil, err
	}

	_, kclient, err := factory.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	opts := &secrets.CreateSecretOptions{
		Name:             secretName,
		SecretsInterface: kclient.Secrets(namespace),
		SecretTypeName:   string(secretType),
		Sources:          sources,
	}

	secret, err := opts.BundleSecret()
	if err != nil {
		return nil, err
	}

	secret, err = opts.SecretsInterface.Create(secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// AddSecretToServiceAccount adds the secret with the given name to the service account with the given name.
//
// Both secret and service account should already exist.
//
// It returns an error, or nil if the operation was successful.
func (c *Context) AddSecretToServiceAccount(secretName string, serviceAccountName string) error {
	factory, err := c.Factory()
	if err != nil {
		return err
	}

	_, kclient, err := factory.Clients()
	if err != nil {
		return err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return err
	}

	mapper, typer := factory.Object()
	clientMapper := factory.ClientMapperForCommand()
	opts := &secrets.AddSecretOptions{
		TargetName:      fmt.Sprintf("serviceaccount/%s", serviceAccountName),
		SecretNames:     []string{fmt.Sprintf("secret/%s", secretName)},
		Namespace:       namespace,
		ForMount:        true,
		ForPull:         true,
		ClientInterface: kclient,
		Mapper:          mapper,
		Typer:           typer,
		ClientMapper:    clientMapper,
	}

	if err = opts.AddSecrets(); err != nil {
		return err
	}

	return nil
}
