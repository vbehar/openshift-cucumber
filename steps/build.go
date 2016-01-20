package steps

import (
	"fmt"
	"time"

	buildapi "github.com/openshift/origin/pkg/build/api"

	kapi "k8s.io/kubernetes/pkg/api"

	"github.com/stretchr/testify/assert"
)

// registers all build related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`^I should have a buildconfig "(.+?)"$`, func(bcName string) {
			bc, err := c.GetBuildConfig(bcName)
			if err != nil {
				c.Fail("Failed to get Build Config '%s': %v", bcName, err)
				return
			}

			assert.Equal(c.T, bcName, bc.Name)
		})

		c.Given(`^I have a buildconfig "(.+?)"$`, func(bcName string) {
			bc, err := c.GetBuildConfig(bcName)
			if err != nil {
				c.Fail("Failed to get Build Config '%s': %v", bcName, err)
				return
			}

			assert.Equal(c.T, bcName, bc.Name)
		})

		c.When(`^I start a new build of "(.+?)"$`, func(bcName string) {
			if _, err := c.StartNewBuild(bcName); err != nil {
				c.Fail("Failed to start a new build for '%s': %v", bcName, err)
			}
		})

		c.Then(`^the latest build of "(.+?)" should succeed in less than "(.+?)"$`, func(bcName string, timeout string) {
			timeoutDuration, err := time.ParseDuration(timeout)
			if err != nil {
				c.Fail("Failed to parse duration '%s': %v", timeout, err)
				return
			}

			bc, err := c.GetBuildConfig(bcName)
			if err != nil {
				c.Fail("Failed to get Build Config '%s': %v", bcName, err)
				return
			}

			latestBuildName := fmt.Sprintf("%s-%d", bc.Name, bc.Status.LastVersion)

			success, err := c.IsBuildComplete(latestBuildName, timeoutDuration)
			if err != nil {
				c.Fail("Failed to check status of the build '%s': %v", latestBuildName, err)
				return
			}

			if !success {
				c.Fail("Build '%s' was not successful!", latestBuildName)
				return
			}
		})

	})
}

// GetBuildConfig gets the BuildConfig with the given name, or returns an error
func (c *Context) GetBuildConfig(bcName string) (*buildapi.BuildConfig, error) {
	client, _, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	bc, err := client.BuildConfigs(namespace).Get(bcName)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// StartNewBuild starts a new build for the BuildConfig with the given name
// and returns the newly created Build, or an error
func (c *Context) StartNewBuild(bcName string) (*buildapi.Build, error) {
	client, _, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	request := &buildapi.BuildRequest{
		ObjectMeta: kapi.ObjectMeta{
			Name: bcName,
		},
	}

	build, err := client.BuildConfigs(namespace).Instantiate(request)
	if err != nil {
		return nil, err
	}

	return build, nil
}

// IsBuildComplete checks if the build with the given name is complete.
//
// If the build is still running, it will wait for the given timeout duration.
//
// It returns true if the build completed, or false if it failed (or was cancelled).
func (c *Context) IsBuildComplete(buildName string, timeout time.Duration) (bool, error) {
	client, _, err := c.Clients()
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
		var build *buildapi.Build

		err = c.ExecWithExponentialBackoff(func() error {
			var err error
			build, err = client.Builds(namespace).Get(buildName)
			return err
		})
		if err != nil {
			return false, err
		}

		switch build.Status.Phase {
		case buildapi.BuildPhaseNew, buildapi.BuildPhasePending, buildapi.BuildPhaseRunning:
			time.Sleep(5 * time.Second)
		case buildapi.BuildPhaseComplete:
			return true, nil
		case buildapi.BuildPhaseFailed, buildapi.BuildPhaseError, buildapi.BuildPhaseCancelled:
			return false, nil
		default:
			return false, fmt.Errorf("Unknown phase %v", build.Status.Phase)
		}
	}

	return false, nil
}
