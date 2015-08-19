package cmd

import (
	"fmt"
	"io"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/fields"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"

	"github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/cmd/cli/describe"
	imageapi "github.com/openshift/origin/pkg/image/api"
	"github.com/spf13/cobra"

	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
)

const (
	importImageLong = `
Import tag and image information from an external Docker image repository

Only image streams that have a value set for spec.dockerImageRepository may
have tag and image information imported.`

	importImageExample = `  $ %[1]s import-image mystream`
)

// NewCmdImportImage implements the OpenShift cli import-image command.
func NewCmdImportImage(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "import-image IMAGESTREAM",
		Short:   "Imports images from a Docker registry",
		Long:    importImageLong,
		Example: fmt.Sprintf(importImageExample, fullName),
		Run: func(cmd *cobra.Command, args []string) {
			err := RunImportImage(f, out, cmd, args)
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().String("from", "", "A Docker image repository to import images from")
	cmd.Flags().Bool("confirm", false, "If true, allow the image stream import location to be set or changed")

	return cmd
}

// RunImportImage contains all the necessary functionality for the OpenShift cli import-image command.
func RunImportImage(f *clientcmd.Factory, out io.Writer, cmd *cobra.Command, args []string) error {
	if len(args) == 0 || len(args[0]) == 0 {
		return cmdutil.UsageError(cmd, "you must specify the name of an image stream")
	}

	streamName := args[0]
	namespace, _, err := f.DefaultNamespace()
	if err != nil {
		return err
	}

	osClient, _, err := f.Clients()
	if err != nil {
		return err
	}

	from := cmdutil.GetFlagString(cmd, "from")
	confirm := cmdutil.GetFlagBool(cmd, "confirm")

	imageStreamClient := osClient.ImageStreams(namespace)
	stream, err := imageStreamClient.Get(streamName)
	if err != nil {
		if len(from) == 0 || !errors.IsNotFound(err) {
			return err
		}
		if !confirm {
			return fmt.Errorf("the image stream does not exist, pass --confirm to create")
		}
		stream = &imageapi.ImageStream{
			ObjectMeta: kapi.ObjectMeta{Name: streamName},
			Spec:       imageapi.ImageStreamSpec{DockerImageRepository: from},
		}
	} else {
		if len(stream.Spec.DockerImageRepository) == 0 {
			if len(from) == 0 {
				return fmt.Errorf("only image streams with spec.dockerImageRepository set may have images imported")
			}
			if !confirm {
				return fmt.Errorf("the image stream already has an import repository set, pass --confirm to update")
			}
			stream.Spec.DockerImageRepository = from
		} else {
			if len(from) != 0 {
				if from != stream.Spec.DockerImageRepository {
					if !confirm {
						return fmt.Errorf("the image stream has a different import spec %q, pass --confirm to update", stream.Spec.DockerImageRepository)
					}
				}
			}
		}
	}

	if stream.Annotations != nil {
		delete(stream.Annotations, imageapi.DockerImageRepositoryCheckAnnotation)
	}

	if stream.CreationTimestamp.IsZero() {
		stream, err = imageStreamClient.Create(stream)
	} else {
		stream, err = imageStreamClient.Update(stream)
	}
	if err != nil {
		return err
	}

	resourceVersion := stream.ResourceVersion

	fmt.Fprintln(cmd.Out(), "Waiting for the import to complete, CTRL+C to stop waiting.")

	stream, err = waitForImport(imageStreamClient, stream.Name, resourceVersion)
	if err != nil {
		return fmt.Errorf("unable to determine if the import completed successfully - please run 'oc describe -n %s imagestream/%s' to see if the tags were updated as expected: %v", stream.Namespace, stream.Name, err)
	}

	fmt.Fprintln(cmd.Out(), "The import completed successfully.\n")

	d := describe.ImageStreamDescriber{osClient}
	info, err := d.Describe(stream.Namespace, stream.Name)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, info)
	return nil
}

func hasImportAnnotation(stream *imageapi.ImageStream) bool {
	return stream.Annotations != nil && len(stream.Annotations[imageapi.DockerImageRepositoryCheckAnnotation]) != 0
}

func waitForImport(imageStreamClient client.ImageStreamInterface, name, resourceVersion string) (*imageapi.ImageStream, error) {
	streamWatch, err := imageStreamClient.Watch(labels.Everything(), fields.SelectorFromSet(fields.Set{"name": name}), resourceVersion)
	if err != nil {
		return nil, err
	}
	defer streamWatch.Stop()

	for {
		select {
		case event, ok := <-streamWatch.ResultChan():
			if !ok {
				return nil, fmt.Errorf("image stream watch ended prematurely")
			}

			switch event.Type {
			case watch.Modified:
				s, ok := event.Object.(*imageapi.ImageStream)
				if !ok {
					continue
				}

				if hasImportAnnotation(s) {
					return s, nil
				}
			case watch.Deleted:
				return nil, fmt.Errorf("the image stream was deleted")
			case watch.Error:
				return nil, fmt.Errorf("error watching image stream")
			}
		}
	}
}
