package steps

import (
	"fmt"
	"io/ioutil"
	"time"

	deployapi "github.com/openshift/origin/pkg/deploy/api"
	deployutil "github.com/openshift/origin/pkg/deploy/util"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"

	"github.com/stretchr/testify/assert"
)

// registers all deployment related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`^I should not have a deploymentconfig "(.+?)"$`, func(dcName string) {
			found, err := c.DeploymentConfigExists(dcName)
			if err != nil {
				c.Fail("Failed to check for Deployment Config '%s' existance: %v", dcName, err)
				return
			}

			if found {
				c.Fail("Deployment Config %s should not exists", dcName)
				return
			}
		})

		c.Then(`^I should have a deploymentconfig "(.+?)"$`, func(dcName string) {
			dc, err := c.GetDeploymentConfig(dcName)
			if err != nil {
				c.Fail("Failed to get Deployment Config '%s': %v", dcName, err)
				return
			}

			assert.Equal(c.T, dcName, dc.Name)
		})

		c.Given(`^I have a deploymentconfig "(.+?)"$`, func(dcName string) {
			dc, err := c.GetDeploymentConfig(dcName)
			if err != nil {
				c.Fail("Failed to get Deployment Config '%s': %v", dcName, err)
				return
			}

			assert.Equal(c.T, dcName, dc.Name)
		})

		c.When(`^the deploymentconfig "(.+?)" has at least (\d+) deployments?$`, func(dcName string, requiredDeployments int) {
			dc, err := c.GetDeploymentConfig(dcName)
			if err != nil {
				c.Fail("Failed to get Deployment Config '%s': %v", dcName, err)
				return
			}

			if dc.Status.LatestVersion < requiredDeployments {
				c.Fail("The Deployment Config '%s' has only %v deployments, instead of the %v deployments required", dcName, dc.Status.LatestVersion, requiredDeployments)
				return
			}
		})

		c.Then(`^the latest deployment of "(.+?)" should succeed in less than "(.+?)"$`, func(dcName string, timeout string) {
			timeoutDuration, err := time.ParseDuration(timeout)
			if err != nil {
				c.Fail("Failed to parse duration '%s': %v", timeout, err)
				return
			}

			dc, err := c.GetDeploymentConfig(dcName)
			if err != nil {
				c.Fail("Failed to get Deployment Config '%s': %v", dcName, err)
				return
			}

			latestDeploymentName := fmt.Sprintf("%s-%d", dc.Name, dc.Status.LatestVersion)

			success, err := c.IsDeploymentComplete(latestDeploymentName, timeoutDuration)
			if err != nil {
				c.Fail("Failed to check status of the deployment '%s': %v", latestDeploymentName, err)
				return
			}

			if !success {
				logs, err := c.GetDeploymentLogs(latestDeploymentName)
				if err != nil {
					fmt.Printf("Failed to get deployment logs '%v'\n", err)
				} else {
					fmt.Printf("Deployment logs '%v'\n", logs)
				}
				c.Fail("Deployment '%s' was not successful!", latestDeploymentName)
				return
			}
		})

		c.When(`^I have a successful deployment of "(.+?)"$`, func(dcName string) {
			rcList, err := c.GetReplicationControllers(deployutil.ConfigSelector(dcName))
			if err != nil {
				c.Fail("Failed to get Deployment Config '%s': %v", dcName, err)
				return
			}

			var successfulDeployment bool
			for _, rc := range rcList.Items {
				if status, ok := rc.Annotations[deployapi.DeploymentStatusAnnotation]; ok {
					switch status {
					case string(deployapi.DeploymentStatusComplete):
						successfulDeployment = true
					default:
					}
				}
			}

			if !successfulDeployment {
				logs, err := c.GetDeploymentLogs(dcName)
				if err != nil {
					fmt.Printf("Failed to get deployment logs '%v'\n", err)
				} else {
					fmt.Printf("Deployment logs '%v'\n", logs)
				}
				c.Fail("No successful deployment for '%s'", dcName)
				return
			}
		})

		c.When(`^I delete the deploymentconfig "(.+?)"$`, func(dcName string) {
			if err := c.DeleteDeploymentConfig(dcName); err != nil {
				c.Fail("Failed to delete deployment config %s", dcName)
			}
		})

	})
}

// DeploymentConfigExists checks if a DeploymentConfig with the given name exists.
func (c *Context) DeploymentConfigExists(dcName string) (bool, error) {
	client, _, err := c.Clients()
	if err != nil {
		return false, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return false, err
	}

	dcList, err := client.DeploymentConfigs(namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return false, err
	}

	for _, dc := range dcList.Items {
		if dc.Name == dcName {
			return true, nil
		}
	}
	return false, nil
}

// GetDeploymentConfig gets the DeploymentConfig with the given name, or returns an error
func (c *Context) GetDeploymentConfig(dcName string) (*deployapi.DeploymentConfig, error) {
	client, _, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	dc, err := client.DeploymentConfigs(namespace).Get(dcName)
	if err != nil {
		return nil, err
	}

	return dc, nil
}

// GetReplicationControllers gets a ReplicationControllerList from the given label selector, or returns an error
func (c *Context) GetReplicationControllers(labelSelector labels.Selector) (*kapi.ReplicationControllerList, error) {
	_, kclient, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	rcList, err := kclient.ReplicationControllers(namespace).List(labelSelector, fields.Everything())
	if err != nil {
		return nil, err
	}

	return rcList, nil
}

// DeleteDeploymentConfig deletes the DeploymentConfig with the given name, or returns an error
func (c *Context) DeleteDeploymentConfig(dcName string) error {
	client, _, err := c.Clients()
	if err != nil {
		return err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return err
	}

	if err = client.DeploymentConfigs(namespace).Delete(dcName); err != nil {
		return err
	}

	return nil
}

// IsDeploymentComplete checks if the deployment with the given name is complete.
//
// If the deployment is still running, it will wait for the given timeout duration.
//
// It returns true if the deployment completed, or false if it failed.
func (c *Context) IsDeploymentComplete(deploymentName string, timeout time.Duration) (bool, error) {
	_, kclient, err := c.Clients()
	if err != nil {
		return false, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return false, err
	}

	startTime := time.Now()

	// TODO use Watch instead of manually polling
	for time.Now().Sub(startTime) < timeout {
		var rc *kapi.ReplicationController

		err = c.ExecWithExponentialBackoff(func() (err error) {
			rc, err = kclient.ReplicationControllers(namespace).Get(deploymentName)
			return
		})
		if err != nil {
			return false, err
		}

		if status, ok := rc.Annotations[deployapi.DeploymentStatusAnnotation]; ok {
			switch status {
			case string(deployapi.DeploymentStatusNew), string(deployapi.DeploymentStatusPending), string(deployapi.DeploymentStatusRunning):
				time.Sleep(5 * time.Second)
			case string(deployapi.DeploymentStatusComplete):
				return true, nil
			case string(deployapi.DeploymentStatusFailed):
				return false, nil
			default:
				return false, fmt.Errorf("Unknown status %v", status)
			}
		}
	}

	return false, nil
}

func (c *Context) GetDeploymentLogs(name string) (string, error) {
	client, _, err := c.Clients()
	if err != nil {
		return "", err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return "", err
	}

	readCloser, err := client.DeploymentLogs(namespace).Get(name, deployapi.DeploymentLogOptions{}).Stream()
	if err != nil {
		return "", err
	}
	defer readCloser.Close()

	bytes, err := ioutil.ReadAll(readCloser)

	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
