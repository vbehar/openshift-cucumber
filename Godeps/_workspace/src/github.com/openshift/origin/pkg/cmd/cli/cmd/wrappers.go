package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	kcmd "k8s.io/kubernetes/pkg/kubectl/cmd"
	kvalidation "k8s.io/kubernetes/pkg/util/validation"

	"github.com/openshift/origin/pkg/cmd/cli/describe"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
)

func tab(original string) string {
	lines := []string{}
	scanner := bufio.NewScanner(strings.NewReader(original))
	for scanner.Scan() {
		lines = append(lines, "  "+scanner.Text())
	}
	return strings.Join(lines, "\n")
}

const (
	getLong = `Display one or many resources

Possible resources include builds, buildConfigs, services, pods, etc.
Some resources may omit advanced details that you can see with '-o wide'.
If you want an even more detailed view, use '%[1]s describe'.`

	getExample = `  # List all pods in ps output format.
  $ %[1]s get pods

  # List a single replication controller with specified ID in ps output format.
  $ %[1]s get rc redis

  # List all pods and show more details about them.
  $ %[1]s get -o wide pods

  # List a single pod in JSON output format.
  $ %[1]s get -o json pod redis-pod

  # Return only the status value of the specified pod.
  $ %[1]s get -o template pod redis-pod --template={{.currentState.status}}`
)

// NewCmdGet is a wrapper for the Kubernetes cli get command
func NewCmdGet(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdGet(f.Factory, out)
	cmd.Long = fmt.Sprintf(getLong, fullName)
	cmd.Example = fmt.Sprintf(getExample, fullName)
	cmd.SuggestFor = []string{"list"}
	return cmd
}

const (
	replaceLong = `Replace a resource by filename or stdin

JSON and YAML formats are accepted.`

	replaceExample = `  # Replace a pod using the data in pod.json.
  $ %[1]s replace -f pod.json

  # Replace a pod based on the JSON passed into stdin.
  $ cat pod.json | %[1]s replace -f -

  # Force replace, delete and then re-create the resource
  $ %[1]s replace --force -f pod.json`
)

// NewCmdReplace is a wrapper for the Kubernetes cli replace command
func NewCmdReplace(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdReplace(f.Factory, out)
	cmd.Long = replaceLong
	cmd.Example = fmt.Sprintf(replaceExample, fullName)
	return cmd
}

const (
	patchLong = `Update field(s) of a resource using strategic merge patch

JSON and YAML formats are accepted.`

	patchExample = `  # Partially update a node using strategic merge patch
  $ %[1]s patch node k8s-node-1 -p '{"spec":{"unschedulable":true}}'`
)

// NewCmdPatch is a wrapper for the Kubernetes cli patch command
func NewCmdPatch(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdPatch(f.Factory, out)
	cmd.Long = patchLong
	cmd.Example = fmt.Sprintf(patchExample, fullName)
	return cmd
}

const (
	deleteLong = `Delete a resource

JSON and YAML formats are accepted.

If both a filename and command line arguments are passed, the command line
arguments are used and the filename is ignored.

Note that the delete command does NOT do resource version checks, so if someone
submits an update to a resource right when you submit a delete, their update
will be lost along with the rest of the resource.`

	deleteExample = `  # Delete a pod using the type and ID specified in pod.json.
  $ %[1]s delete -f pod.json

  # Delete a pod based on the type and ID in the JSON passed into stdin.
  $ cat pod.json | %[1]s delete -f -

  # Delete pods and services with label name=myLabel.
  $ %[1]s delete pods,services -l name=myLabel

  # Delete a pod with ID 1234-56-7890-234234-456456.
  $ %[1]s delete pod 1234-56-7890-234234-456456

  # Delete all pods
  $ %[1]s delete pods --all`
)

// NewCmdDelete is a wrapper for the Kubernetes cli delete command
func NewCmdDelete(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdDelete(f.Factory, out)
	cmd.Long = deleteLong
	cmd.Example = fmt.Sprintf(deleteExample, fullName)
	cmd.SuggestFor = []string{"remove"}
	return cmd
}

const (
	createLong = `Create a resource by filename or stdin

JSON and YAML formats are accepted.`

	createExample = `  # Create a pod using the data in pod.json.
  $ %[1]s create -f pod.json

  # Create a pod based on the JSON passed into stdin.
  $ cat pod.json | %[1]s create -f -`
)

// NewCmdCreate is a wrapper for the Kubernetes cli create command
func NewCmdCreate(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdCreate(f.Factory, out)
	cmd.Long = createLong
	cmd.Example = fmt.Sprintf(createExample, fullName)
	return cmd
}

const (
	execLong = `Execute a command in a container`

	execExample = `  # Get output from running 'date' in ruby-container from pod 123456-7890
  $ %[1]s exec -p 123456-7890 -c ruby-container date

  # Switch to raw terminal mode, sends stdin to 'bash' in ruby-container from pod 123456-780 and sends stdout/stderr from 'bash' back to the client
  $ %[1]s exec -p 123456-7890 -c ruby-container -i -t -- bash -il`
)

// NewCmdExec is a wrapper for the Kubernetes cli exec command
func NewCmdExec(fullName string, f *clientcmd.Factory, cmdIn io.Reader, cmdOut, cmdErr io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdExec(f.Factory, cmdIn, cmdOut, cmdErr)
	cmd.Use = "exec POD [-c CONTAINER] [options] -- COMMAND [args...]"
	cmd.Long = execLong
	cmd.Example = fmt.Sprintf(execExample, fullName)
	return cmd
}

const (
	portForwardLong = `Forward 1 or more local ports to a pod`

	portForwardExample = `  # Listens on ports 5000 and 6000 locally, forwarding data to/from ports 5000 and 6000 in the pod
  $ %[1]s port-forward -p mypod 5000 6000

  # Listens on port 8888 locally, forwarding to 5000 in the pod
  $ %[1]s port-forward -p mypod 8888:5000

  # Listens on a random port locally, forwarding to 5000 in the pod
  $ %[1]s port-forward -p mypod :5000

  # Listens on a random port locally, forwarding to 5000 in the pod
  $ %[1]s port-forward -p mypod 0:5000`
)

// NewCmdPortForward is a wrapper for the Kubernetes cli port-forward command
func NewCmdPortForward(fullName string, f *clientcmd.Factory) *cobra.Command {
	cmd := kcmd.NewCmdPortForward(f.Factory)
	cmd.Long = portForwardLong
	cmd.Example = fmt.Sprintf(portForwardExample, fullName)
	return cmd
}

const (
	describeLong = `Show details of a specific resource

This command joins many API calls together to form a detailed description of a
given resource.`

	describeExample = `  # Provide details about the ruby-22-centos7 image repository
  $ %[1]s describe imageRepository ruby-22-centos7

  # Provide details about the ruby-sample-build build configuration
  $ %[1]s describe bc ruby-sample-build`
)

// NewCmdDescribe is a wrapper for the Kubernetes cli describe command
func NewCmdDescribe(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdDescribe(f.Factory, out)
	cmd.Long = describeLong
	cmd.Example = fmt.Sprintf(describeExample, fullName)
	cmd.ValidArgs = describe.DescribableResources()
	return cmd
}

const (
	proxyLong = `Run a proxy to the Kubernetes API server`

	proxyExample = `  # Run a proxy to kubernetes apiserver on port 8011, serving static content from ./local/www/
  $ %[1]s proxy --port=8011 --www=./local/www/

  # Run a proxy to kubernetes apiserver, changing the api prefix to k8s-api
  # This makes e.g. the pods api available at localhost:8011/k8s-api/v1beta3/pods/
  $ %[1]s proxy --api-prefix=k8s-api`
)

// NewCmdProxy is a wrapper for the Kubernetes cli proxy command
func NewCmdProxy(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdProxy(f.Factory, out)
	cmd.Long = proxyLong
	cmd.Example = fmt.Sprintf(proxyExample, fullName)
	return cmd
}

const (
	scaleLong = `Set a new size for a deployment or replication controller

Scale also allows users to specify one or more preconditions for the scale action.
If --current-replicas or --resource-version is specified, it is validated before the
scale is attempted, and it is guaranteed that the precondition holds true when the
scale is sent to the server.

Note that scaling a deployment configuration with no deployments will update the
desired replicas in the configuration template.`

	scaleExample = `  # Scale replication controller named 'foo' to 3.
  $ %[1]s scale --replicas=3 replicationcontrollers foo

  # If the replication controller named foo's current size is 2, scale foo to 3.
  $ %[1]s scale --current-replicas=2 --replicas=3 replicationcontrollers foo

  # Scale the latest deployment of 'bar'. In case of no deployment, bar's template
  # will be scaled instead.
  $ %[1]s scale --replicas=10 dc bar`
)

// NewCmdScale is a wrapper for the Kubernetes cli scale command
func NewCmdScale(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdScale(f.Factory, out)
	cmd.Short = "Change the number of pods in a deployment"
	cmd.Long = scaleLong
	cmd.Example = fmt.Sprintf(scaleExample, fullName)
	cmd.ValidArgs = []string{"deploymentconfig", "job", "replicationcontroller"}
	return cmd
}

const (
	stopLong = `Gracefully shut down a resource by id or filename

The stop command is deprecated, all its functionalities are covered by the delete command.
See '%[1]s delete --help' for more details.`

	stopExample = `  # Shut down foo.
  $ %[1]s stop replicationcontroller foo

  # Stop pods and services with label name=myLabel.
  $ %[1]s stop pods,services -l name=myLabel

  # Shut down the service defined in service.json
  $ %[1]s stop -f service.json

  # Shut down all resources in the path/to/resources directory
  $ %[1]s stop -f path/to/resources`
)

// NewCmdStop is a wrapper for the Kubernetes cli stop command
func NewCmdStop(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdStop(f.Factory, out)
	cmd.Long = fmt.Sprintf(stopLong, fullName)
	cmd.Example = fmt.Sprintf(stopExample, fullName)
	return cmd
}

const (
	runLong = `Create and run a particular image, possibly replicated

Creates a deployment config to manage the created container(s). You can choose to run in the
foreground for an interactive container execution.  You may pass 'run-controller/v1' to
--generator to create a replication controller instead of a deployment config.`

	runExample = `  # Starts a single instance of nginx.
  $ %[1]s run nginx --image=nginx

  # Starts a replicated instance of nginx.
  $ %[1]s run nginx --image=nginx --replicas=5

  # Dry run. Print the corresponding API objects without creating them.
  $ %[1]s run nginx --image=nginx --dry-run

  # Start a single instance of nginx, but overload the spec of the replication
  # controller with a partial set of values parsed from JSON.
  $ %[1]s run nginx --image=nginx --overrides='{ "apiVersion": "v1", "spec": { ... } }'

  # Start a single instance of nginx and keep it in the foreground, don't restart it if it exits.
  $ %[1]s run -i --tty nginx --image=nginx --restart=Never`

	// TODO: uncomment these when arguments are delivered upstream

	// Start the nginx container using the default command, but use custom
	// arguments (arg1 .. argN) for that command.
	//$ %[1]s run nginx --image=nginx -- <arg1> <arg2> ... <argN>

	// Start the nginx container using a different command and custom arguments
	//$ %[1]s run nginx --image=nginx --command -- <cmd> <arg1> ... <argN>`
)

// NewCmdRun is a wrapper for the Kubernetes cli run command
func NewCmdRun(fullName string, f *clientcmd.Factory, in io.Reader, out, errout io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdRun(f.Factory, in, out, errout)
	cmd.Long = runLong
	cmd.Example = fmt.Sprintf(runExample, fullName)
	cmd.SuggestFor = []string{"image"}
	cmd.Flags().Set("generator", "")
	cmd.Flag("generator").Usage = "The name of the API generator to use.  Default is 'run/v1' if --restart=Always, otherwise the default is 'run-pod/v1'."
	cmd.Flag("generator").DefValue = ""
	return cmd
}

const (
	attachLong = `Attach to a running container

Attach the current shell to a remote container, returning output or setting up a full
terminal session. Can be used to debug containers and invoke interactive commands.`

	attachExample = `  # Get output from running pod 123456-7890, using the first container by default
  $ %[1]s attach 123456-7890

  # Get output from ruby-container from pod 123456-7890
  $ %[1]s attach 123456-7890 -c ruby-container

  # Switch to raw terminal mode, sends stdin to 'bash' in ruby-container from pod 123456-780
  # and sends stdout/stderr from 'bash' back to the client
  $ %[1]s attach 123456-7890 -c ruby-container -i -t`
)

// NewCmdAttach is a wrapper for the Kubernetes cli attach command
func NewCmdAttach(fullName string, f *clientcmd.Factory, in io.Reader, out, errout io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdAttach(f.Factory, in, out, errout)
	cmd.Long = attachLong
	cmd.Example = fmt.Sprintf(attachExample, fullName)
	return cmd
}

const (
	annotateLong = `Update the annotations on one or more resources

An annotation is a key/value pair that can hold larger (compared to a label),
and possibly not human-readable, data. It is intended to store non-identifying
auxiliary data, especially data manipulated by tools and system extensions. If
--overwrite is true, then existing annotations can be overwritten, otherwise
attempting to overwrite an annotation will result in an error. If
--resource-version is specified, then updates will use this resource version,
otherwise the existing resource-version will be used.

Run '%[1]s types' for a list of valid resources.`

	annotateExample = `  # Update pod 'foo' with the annotation 'description' and the value 'my frontend'.
  # If the same annotation is set multiple times, only the last value will be applied
  $ %[1]s annotate pods foo description='my frontend'

  # Update pod 'foo' with the annotation 'description' and the value
  # 'my frontend running nginx', overwriting any existing value.
  $ %[1]s annotate --overwrite pods foo description='my frontend running nginx'

  # Update all pods in the namespace
  $ %[1]s annotate pods --all description='my frontend running nginx'

  # Update pod 'foo' only if the resource is unchanged from version 1.
  $ %[1]s annotate pods foo description='my frontend running nginx' --resource-version=1

  # Update pod 'foo' by removing an annotation named 'description' if it exists.
  # Does not require the --overwrite flag.
  $ %[1]s annotate pods foo description-`
)

// NewCmdAnnotate is a wrapper for the Kubernetes cli annotate command
func NewCmdAnnotate(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdAnnotate(f.Factory, out)
	cmd.Long = fmt.Sprintf(annotateLong, fullName)
	cmd.Example = fmt.Sprintf(annotateExample, fullName)
	return cmd
}

const (
	labelLong = `Update the labels on one or more resources

A valid label value is consisted of letters and/or numbers with a max length of %[1]d
characters. If --overwrite is true, then existing labels can be overwritten, otherwise
attempting to overwrite a label will result in an error. If --resource-version is
specified, then updates will use this resource version, otherwise the existing
resource-version will be used.`

	labelExample = `  # Update pod 'foo' with the label 'unhealthy' and the value 'true'.
  $ %[1]s label pods foo unhealthy=true

  # Update pod 'foo' with the label 'status' and the value 'unhealthy', overwriting any existing value.
  $ %[1]s label --overwrite pods foo status=unhealthy

  # Update all pods in the namespace
  $ %[1]s label pods --all status=unhealthy

  # Update pod 'foo' only if the resource is unchanged from version 1.
  $ %[1]s label pods foo status=unhealthy --resource-version=1

  # Update pod 'foo' by removing a label named 'bar' if it exists.
  # Does not require the --overwrite flag.
  $ %[1]s label pods foo bar-`
)

// NewCmdLabel is a wrapper for the Kubernetes cli label command
func NewCmdLabel(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdLabel(f.Factory, out)
	cmd.Long = fmt.Sprintf(labelLong, kvalidation.LabelValueMaxLength)
	cmd.Example = fmt.Sprintf(labelExample, fullName)
	return cmd
}

const (
	applyLong = `Apply a configuration to a resource by filename or stdin.

JSON and YAML formats are accepted.`

	applyExample = `# Apply the configuration in pod.json to a pod.
$ %[1]s apply -f ./pod.json

# Apply the JSON passed into stdin to a pod.
$ cat pod.json | %[1]s apply -f -`
)

func NewCmdApply(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdApply(f.Factory, out)
	cmd.Long = applyLong
	cmd.Example = fmt.Sprintf(applyExample, fullName)
	return cmd
}

const (
	explainLong = `Documentation of resources.

Possible resource types include: pods (po), services (svc),
replicationcontrollers (rc), nodes (no), events (ev), componentstatuses (cs),
limitranges (limits), persistentvolumes (pv), persistentvolumeclaims (pvc),
resourcequotas (quota), namespaces (ns) or endpoints (ep).`

	explainExample = `# Get the documentation of the resource and its fields
$ %[1]s explain pods

# Get the documentation of a specific field of a resource
$ %[1]s explain pods.spec.containers`
)

func NewCmdExplain(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdExplain(f.Factory, out)
	cmd.Long = explainLong
	cmd.Example = fmt.Sprintf(explainExample, fullName)
	return cmd
}

const (
	convertLong = `Convert config files between different API versions. Both YAML
and JSON formats are accepted.

The command takes filename, directory, or URL as input, and convert it into format
of version specified by --output-version flag. If target version is not specified or
not supported, convert to latest version.

The default output will be printed to stdout in YAML format. One can use -o option
to change to output destination.
`
	convertExample = `# Convert 'pod.yaml' to latest version and print to stdout.
$ %[1]s convert -f pod.yaml

# Convert the live state of the resource specified by 'pod.yaml' to the latest version
# and print to stdout in json format.
$ %[1]s convert -f pod.yaml --local -o json

# Convert all files under current directory to latest version and create them all.
$ %[1]s convert -f . | kubectl create -f -
`
)

func NewCmdConvert(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := kcmd.NewCmdConvert(f.Factory, out)
	cmd.Long = convertLong
	cmd.Example = fmt.Sprintf(convertExample, fullName)
	return cmd
}
