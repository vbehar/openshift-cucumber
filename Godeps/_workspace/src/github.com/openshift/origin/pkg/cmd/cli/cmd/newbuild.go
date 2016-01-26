package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/util/errors"

	buildapi "github.com/openshift/origin/pkg/build/api"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	configcmd "github.com/openshift/origin/pkg/config/cmd"
	newapp "github.com/openshift/origin/pkg/generate/app"
	newcmd "github.com/openshift/origin/pkg/generate/app/cmd"
	"k8s.io/kubernetes/pkg/labels"
)

const (
	newBuildLong = `
Create a new build by specifying source code

This command will try to create a build configuration for your application using images and
code that has a public repository. It will lookup the images on the local Docker installation
(if available), a Docker registry, or an image stream.

If you specify a source code URL, it will set up a build that takes your source code and converts
it into an image that can run inside of a pod. Local source must be in a git repository that has a
remote repository that the server can see.

Once the build configuration is created a new build will be automatically triggered.
You can use '%[1]s status' to check the progress.`

	newBuildExample = `
  # Create a build config based on the source code in the current git repository (with a public
  # remote) and a Docker image
  $ %[1]s new-build . --docker-image=repo/langimage

  # Create a NodeJS build config based on the provided [image]~[source code] combination
  $ %[1]s new-build openshift/nodejs-010-centos7~https://github.com/openshift/nodejs-ex.git

  # Create a build config from a remote repository using its beta2 branch
  $ %[1]s new-build https://github.com/openshift/ruby-hello-world#beta2

  # Create a build config using a Dockerfile specified as an argument
  $ %[1]s new-build -D $'FROM centos:7\nRUN yum install -y httpd'

  # Create a build config from a remote repository and add custom environment variables
  $ %[1]s new-build https://github.com/openshift/ruby-hello-world RACK_ENV=development`

	newBuildNoInput = `You must specify one or more images, image streams, or source code locations to create a build.

To build from an existing image stream tag or Docker image, provide the name of the image and
the source code location:

  $ %[1]s new-build openshift/nodejs-010-centos7~https://github.com/openshift/nodejs-ex.git

If you only specify the source repository location (local or remote), the command will look at
the repo to determine the type, and then look for a matching image on your server or on the
default Docker registry.

  $ %[1]s new-build https://github.com/openshift/nodejs-ex.git

will look for an image called "nodejs" in your current project, the 'openshift' project, or
on the Docker Hub.
`
)

// NewCmdNewBuild implements the OpenShift cli new-build command
func NewCmdNewBuild(fullName string, f *clientcmd.Factory, in io.Reader, out io.Writer) *cobra.Command {
	config := newcmd.NewAppConfig()
	config.ExpectToBuild = true

	cmd := &cobra.Command{
		Use:        "new-build (IMAGE | IMAGESTREAM | PATH | URL ...)",
		Short:      "Create a new build configuration",
		Long:       fmt.Sprintf(newBuildLong, fullName),
		Example:    fmt.Sprintf(newBuildExample, fullName),
		SuggestFor: []string{"build", "builds"},
		Run: func(c *cobra.Command, args []string) {
			mapper, typer := f.Object()
			config.SetMapper(mapper)
			config.SetTyper(typer)
			config.SetClientMapper(f.ClientMapperForCommand())

			config.AddEnvironmentToBuild = true
			err := RunNewBuild(fullName, f, out, in, c, args, config)
			if err == errExit {
				os.Exit(1)
			}
			cmdutil.CheckErr(err)
		},
	}

	cmd.Flags().StringSliceVar(&config.SourceRepositories, "code", config.SourceRepositories, "Source code in the build configuration.")
	cmd.Flags().StringSliceVarP(&config.ImageStreams, "image", "i", config.ImageStreams, "Name of an image stream to to use as a builder.")
	cmd.Flags().StringSliceVar(&config.DockerImages, "docker-image", config.DockerImages, "Name of a Docker image to use as a builder.")
	cmd.Flags().StringVar(&config.Name, "name", "", "Set name to use for generated build artifacts.")
	cmd.Flags().StringVar(&config.To, "to", "", "Push built images to this image stream tag (or Docker image repository if --to-docker is set).")
	cmd.Flags().BoolVar(&config.OutputDocker, "to-docker", false, "Have the build output push to a Docker repository.")
	cmd.Flags().StringSliceVarP(&config.Environment, "env", "e", config.Environment, "Specify key value pairs of environment variables to set into resulting image.")
	cmd.Flags().StringVar(&config.Strategy, "strategy", "", "Specify the build strategy to use if you don't want to detect (docker|source).")
	cmd.Flags().StringVarP(&config.Dockerfile, "dockerfile", "D", "", "Specify the contents of a Dockerfile to build directly, implies --strategy=docker. Pass '-' to read from STDIN.")
	cmd.Flags().BoolVar(&config.BinaryBuild, "binary", false, "Instead of expecting a source URL, set the build to expect binary contents. Will disable triggers.")
	cmd.Flags().StringP("labels", "l", "", "Label to set in all generated resources.")
	cmd.Flags().BoolVar(&config.AllowMissingImages, "allow-missing-images", false, "If true, indicates that referenced Docker images that cannot be found locally or in a registry should still be used.")
	cmd.Flags().StringVar(&config.ContextDir, "context-dir", "", "Context directory to be used for the build.")
	cmd.Flags().BoolVar(&config.DryRun, "dry-run", false, "If true, do not actually create resources.")
	cmd.Flags().BoolVar(&config.NoOutput, "no-output", false, "If true, the build output will not be pushed anywhere.")
	cmdutil.AddPrinterFlags(cmd)

	return cmd
}

// RunNewBuild contains all the necessary functionality for the OpenShift cli new-build command
func RunNewBuild(fullName string, f *clientcmd.Factory, out io.Writer, in io.Reader, c *cobra.Command, args []string, config *newcmd.AppConfig) error {
	output := cmdutil.GetFlagString(c, "output")
	shortOutput := output == "name"

	if config.Dockerfile == "-" {
		data, err := ioutil.ReadAll(in)
		if err != nil {
			return err
		}
		config.Dockerfile = string(data)
	}

	if err := setupAppConfig(f, out, c, args, config); err != nil {
		return err
	}

	if err := setAppConfigLabels(c, config); err != nil {
		return err
	}
	result, err := config.Run()
	if err != nil {
		return handleBuildError(c, err, fullName)
	}

	if len(config.Labels) == 0 && len(result.Name) > 0 {
		config.Labels = map[string]string{"build": result.Name}
	}

	if err := setLabels(config.Labels, result); err != nil {
		return err
	}
	if err := setAnnotations(map[string]string{newcmd.GeneratedByNamespace: newcmd.GeneratedByNewBuild}, result); err != nil {
		return err
	}

	indent := "    "
	switch {
	case shortOutput:
		indent = ""
	case len(output) != 0:
		return f.Factory.PrintObject(c, result.List, out)
	default:
		if len(config.Labels) > 0 {
			fmt.Fprintf(out, "--> Creating resources with label %s ...\n", labels.SelectorFromSet(config.Labels).String())
		} else {
			fmt.Fprintf(out, "--> Creating resources ...\n")
		}
	}
	if config.DryRun {
		fmt.Fprintf(out, "--> Success (DRY RUN)\n")
		return nil
	}

	mapper, _ := f.Object()
	if err := createObjects(f, configcmd.NewPrintNameOrErrorAfterIndent(mapper, shortOutput, "created", out, c.Out(), indent), result); err != nil {
		return err
	}

	if shortOutput {
		return nil
	}

	fmt.Fprintf(out, "--> Success\n")
	for _, item := range result.List.Items {
		switch t := item.(type) {
		case *buildapi.BuildConfig:
			if len(t.Spec.Triggers) > 0 && t.Spec.Source.Binary == nil {
				fmt.Fprintf(out, "%sBuild configuration %q created and build triggered.\n", indent, t.Name)
				fmt.Fprintf(out, "%sRun '%s logs -f bc/%s' to stream the build progress.\n", indent, fullName, t.Name)
			}
		}
	}

	return nil
}

func handleBuildError(c *cobra.Command, err error, fullName string) error {
	if err == nil {
		return nil
	}
	if errs, ok := err.(errors.Aggregate); ok {
		if len(errs.Errors()) == 1 {
			err = errs.Errors()[0]
		}
	}
	switch t := err.(type) {
	case newapp.ErrNoMatch:
		return fmt.Errorf(`%[1]v

The '%[2]s' command will match arguments to the following types:

  1. Images tagged into image streams in the current project or the 'openshift' project
     - if you don't specify a tag, we'll add ':latest'
  2. Images in the Docker Hub, on remote registries, or on the local Docker engine
  3. Git repository URLs or local paths that point to Git repositories

--allow-missing-images can be used to point to an image that does not exist yet
or is only on the local system.

See '%[2]s' for examples.
`, t, c.Name())
	}
	switch err {
	case newcmd.ErrNoInputs:
		// TODO: suggest things to the user
		return cmdutil.UsageError(c, newBuildNoInput, fullName)
	default:
		return err
	}
}
