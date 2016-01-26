package steps

import (
	"fmt"

	deployapi "github.com/openshift/origin/pkg/deploy/api"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"

	"github.com/stretchr/testify/assert"
)

// registers all pod related steps
func init() {
	RegisterSteps(func(c *Context) {

		c.Then(`^I should have a pod "(.+?)"$`, func(podName string) {
			pod, err := c.GetPod(podName)
			if err != nil {
				c.Fail("Failed to get Pod '%s': %v", podName, err)
				return
			}

			assert.Equal(c.T, podName, pod.Name)
		})

		c.When(`^I open a tunnel "(.+?)" to a pod of the latest deployment of "(.+?)" on port (\d+)$`, func(tunnelName string, dcName string, targetPort int) {
			dc, err := c.GetDeploymentConfig(dcName)
			if err != nil {
				c.Fail("Failed to get Deployment Config '%s': %v", dcName, err)
				return
			}

			deployment := fmt.Sprintf("%v-%v", dc.Name, dc.Status.LatestVersion)
			deploymentSelector := labels.Set{deployapi.DeploymentLabel: deployment}.AsSelector()
			pods, err := c.GetPods(deploymentSelector)
			if err != nil {
				c.Fail("Failed to get pods for label selector %+v: %v", deploymentSelector, err)
				return
			}

			var pod *kapi.Pod
			for _, p := range pods.Items {
				if p.Status.Phase == kapi.PodRunning {
					pod = &p
					break
				}
			}
			if pod == nil {
				c.Fail("Could not find any running pod for label selector %+v: %v", deploymentSelector, err)
				return
			}

			_, err = c.OpenTunnel(tunnelName, pod.Name, targetPort)
			if err != nil {
				c.Fail("Failed to open tunnel %s: %v", tunnelName, err)
				return
			}
		})

		c.Then(`^I close the tunnel "(.+?)"$`, func(tunnelName string) {
			c.CloseTunnel(tunnelName)
		})

	})
}

// GetPod gets the Pod with the given name, or returns an error
func (c *Context) GetPod(podName string) (*kapi.Pod, error) {
	_, kclient, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	pod, err := kclient.Pods(namespace).Get(podName)
	if err != nil {
		return nil, err
	}

	return pod, nil
}

// GetPods gets the PodList from the given label selector, or returns an error
func (c *Context) GetPods(labelSelector labels.Selector) (*kapi.PodList, error) {
	_, kclient, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	pods, err := kclient.Pods(namespace).List(labelSelector, fields.Everything())
	if err != nil {
		return nil, err
	}

	return pods, nil
}
