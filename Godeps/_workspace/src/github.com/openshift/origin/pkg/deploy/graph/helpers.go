package graph

import (
	"sort"

	kapi "k8s.io/kubernetes/pkg/api"

	osgraph "github.com/openshift/origin/pkg/api/graph"
	kubegraph "github.com/openshift/origin/pkg/api/kubegraph/nodes"
	deployapi "github.com/openshift/origin/pkg/deploy/api"
	deploygraph "github.com/openshift/origin/pkg/deploy/graph/nodes"
	deployutil "github.com/openshift/origin/pkg/deploy/util"
	imageapi "github.com/openshift/origin/pkg/image/api"
)

// RelevantDeployments returns the active deployment and a list of inactive deployments (in order from newest to oldest)
func RelevantDeployments(g osgraph.Graph, dcNode *deploygraph.DeploymentConfigNode) (*kubegraph.ReplicationControllerNode, []*kubegraph.ReplicationControllerNode) {
	allDeployments := []*kubegraph.ReplicationControllerNode{}
	uncastDeployments := g.SuccessorNodesByEdgeKind(dcNode, DeploymentEdgeKind)
	if len(uncastDeployments) == 0 {
		return nil, []*kubegraph.ReplicationControllerNode{}
	}

	for i := range uncastDeployments {
		allDeployments = append(allDeployments, uncastDeployments[i].(*kubegraph.ReplicationControllerNode))
	}

	sort.Sort(RecentDeploymentReferences(allDeployments))

	if dcNode.DeploymentConfig.Status.LatestVersion == deployutil.DeploymentVersionFor(allDeployments[0]) {
		return allDeployments[0], allDeployments[1:]
	}

	return nil, allDeployments
}

func BelongsToDeploymentConfig(config *deployapi.DeploymentConfig, b *kapi.ReplicationController) bool {
	if b.Annotations != nil {
		return config.Name == deployutil.DeploymentConfigNameFor(b)
	}
	return false
}

type RecentDeploymentReferences []*kubegraph.ReplicationControllerNode

func (m RecentDeploymentReferences) Len() int      { return len(m) }
func (m RecentDeploymentReferences) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m RecentDeploymentReferences) Less(i, j int) bool {
	return deployutil.DeploymentVersionFor(m[i].ReplicationController) > deployutil.DeploymentVersionFor(m[j].ReplicationController)
}

// TODO: move to deploy/api/helpers.go
type TemplateImage struct {
	Image string

	Ref *imageapi.DockerImageReference

	From *kapi.ObjectReference
}

func EachTemplateImage(pod *kapi.PodSpec, triggerFn TriggeredByFunc, fn func(TemplateImage, error)) {
	for _, container := range pod.Containers {
		var ref imageapi.DockerImageReference
		if trigger, ok := triggerFn(&container); ok {
			trigger.Image = container.Image
			fn(trigger, nil)
			continue
		}
		ref, err := imageapi.ParseDockerImageReference(container.Image)
		if err != nil {
			fn(TemplateImage{Image: container.Image}, err)
			continue
		}
		fn(TemplateImage{Image: container.Image, Ref: &ref}, nil)
	}
}

type TriggeredByFunc func(container *kapi.Container) (TemplateImage, bool)

func DeploymentConfigHasTrigger(config *deployapi.DeploymentConfig) TriggeredByFunc {
	return func(container *kapi.Container) (TemplateImage, bool) {
		for _, trigger := range config.Spec.Triggers {
			params := trigger.ImageChangeParams
			if params == nil {
				continue
			}
			for _, name := range params.ContainerNames {
				if container.Name == name {
					if len(params.From.Name) == 0 {
						continue
					}
					from := params.From
					if len(from.Namespace) == 0 {
						from.Namespace = config.Namespace
					}
					return TemplateImage{
						Image: container.Image,
						From:  &from,
					}, true
				}
			}
		}
		return TemplateImage{}, false
	}
}
