package steps

import (
	"fmt"
	"net"
	"time"

	kclientapi "k8s.io/kubernetes/pkg/client"
	"k8s.io/kubernetes/pkg/client/portforward"
)

// Tunnel is a wrapper around an HTTP tunnel
// implemented by port forwarding
type Tunnel struct {
	Name      string
	LocalPort int
	stopChan  chan struct{}
}

// NewTunnel build a new Tunnel with the given name
// The tunnel will need to be started with StartForwardingToPod
func NewTunnel(name string) *Tunnel {
	return &Tunnel{
		Name:     name,
		stopChan: make(chan struct{}, 1),
	}
}

// StartForwardingToPod starts forwarding requests to the given pod on the given target port
// If no localPort has been defined on the tunnel, a random available port will be assigned
// The tunnel is started in the background (using a goroutine), and will need to be stopped with Stop()
// It returns an error if it can't start the tunnel.
func (tunnel *Tunnel) StartForwardingToPod(podName string, namespace string, targetPort int, restClient *kclientapi.RESTClient, clientConfig *kclientapi.Config) error {
	req := restClient.Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	if tunnel.LocalPort == 0 {
		port, err := getRandomAvailableLocalPort()
		if err != nil {
			return err
		}
		tunnel.LocalPort = port
	}

	port := fmt.Sprintf("%v:%v", tunnel.LocalPort, targetPort)
	ports := []string{port}

	fw, err := portforward.New(req, clientConfig, ports, tunnel.stopChan)
	if err != nil {
		return err
	}

	go func(localPort int) {
		err = fw.ForwardPorts()
		if err != nil {
			fmt.Printf("Failed to forward localPort %v to remotePort %v on pod %s: %v\n", localPort, targetPort, podName, err)
		}
	}(tunnel.LocalPort)

	// FIXME wait a little to make sure the port forwarding has been set up
	time.Sleep(200 * time.Millisecond)

	return nil
}

// StopForwarding stop forwarding to the pod, and close the tunnel.
func (tunnel *Tunnel) StopForwarding() {
	close(tunnel.stopChan)
}

// Close is an alias of StopForwarding
func (tunnel *Tunnel) Close() {
	tunnel.StopForwarding()
}

// OpenTunnel opens a new Tunnel with the given name, targeting the given pod and port.
// It returns the tunnel object (to get the local port), or an error.
func (c *Context) OpenTunnel(tunnelName string, podName string, targetPort int) (*Tunnel, error) {
	_, kclient, err := c.Clients()
	if err != nil {
		return nil, err
	}

	namespace, err := c.Namespace()
	if err != nil {
		return nil, err
	}

	config, err := c.ClientConfig()
	if err != nil {
		return nil, err
	}

	tunnel := NewTunnel(tunnelName)
	if err = tunnel.StartForwardingToPod(podName, namespace, targetPort, kclient.RESTClient, config); err != nil {
		return nil, err
	}

	c.tunnels[tunnelName] = *tunnel

	return tunnel, nil
}

// CloseTunnel closes the tunnel with the given name
func (c *Context) CloseTunnel(tunnelName string) {
	if tunnel, found := c.tunnels[tunnelName]; found {
		tunnel.Close()
		delete(c.tunnels, tunnelName)
	}
}

// getRandomAvailableLocalPort find an available TCP local port
// and return it - or an error
func getRandomAvailableLocalPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}

	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	return port, nil
}
