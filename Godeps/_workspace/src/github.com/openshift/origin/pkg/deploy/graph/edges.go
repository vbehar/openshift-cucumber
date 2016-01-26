package graph

import (
	"github.com/gonum/graph"

	osgraph "github.com/openshift/origin/pkg/api/graph"
	kubegraph "github.com/openshift/origin/pkg/api/kubegraph/nodes"
	deploygraph "github.com/openshift/origin/pkg/deploy/graph/nodes"
	imageapi "github.com/openshift/origin/pkg/image/api"
	imagegraph "github.com/openshift/origin/pkg/image/graph/nodes"
)

const (
	TriggersDeploymentEdgeKind = "TriggersDeployment"
	UsedInDeploymentEdgeKind   = "UsedInDeployment"
	DeploymentEdgeKind         = "Deployment"
)

// AddTriggerEdges creates edges that point to named Docker image repositories for each image used in the deployment.
func AddTriggerEdges(g osgraph.MutableUniqueGraph, node *deploygraph.DeploymentConfigNode) *deploygraph.DeploymentConfigNode {
	podTemplate := node.DeploymentConfig.Spec.Template
	if podTemplate == nil {
		return node
	}

	EachTemplateImage(
		&podTemplate.Spec,
		DeploymentConfigHasTrigger(node.DeploymentConfig),
		func(image TemplateImage, err error) {
			if err != nil {
				return
			}
			if image.From != nil {
				if len(image.From.Name) == 0 {
					return
				}
				name, tag, _ := imageapi.SplitImageStreamTag(image.From.Name)
				in := imagegraph.FindOrCreateSyntheticImageStreamTagNode(g, imagegraph.MakeImageStreamTagObjectMeta(image.From.Namespace, name, tag))
				g.AddEdge(in, node, TriggersDeploymentEdgeKind)
				return
			}

			tag := image.Ref.Tag
			image.Ref.Tag = ""
			in := imagegraph.EnsureDockerRepositoryNode(g, image.Ref.String(), tag)
			g.AddEdge(in, node, UsedInDeploymentEdgeKind)
		})

	return node
}

func AddAllTriggerEdges(g osgraph.MutableUniqueGraph) {
	for _, node := range g.(graph.Graph).Nodes() {
		if dcNode, ok := node.(*deploygraph.DeploymentConfigNode); ok {
			AddTriggerEdges(g, dcNode)
		}
	}
}

func AddDeploymentEdges(g osgraph.MutableUniqueGraph, node *deploygraph.DeploymentConfigNode) *deploygraph.DeploymentConfigNode {
	for _, n := range g.(graph.Graph).Nodes() {
		if rcNode, ok := n.(*kubegraph.ReplicationControllerNode); ok {
			if BelongsToDeploymentConfig(node.DeploymentConfig, rcNode.ReplicationController) {
				g.AddEdge(node, rcNode, DeploymentEdgeKind)
			}
		}
	}

	return node
}

func AddAllDeploymentEdges(g osgraph.MutableUniqueGraph) {
	for _, node := range g.(graph.Graph).Nodes() {
		if dcNode, ok := node.(*deploygraph.DeploymentConfigNode); ok {
			AddDeploymentEdges(g, dcNode)
		}
	}
}
