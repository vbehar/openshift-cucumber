package describe

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	kapi "k8s.io/kubernetes/pkg/api"
	kapierrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	utilerrors "k8s.io/kubernetes/pkg/util/errors"
	"k8s.io/kubernetes/pkg/util/sets"

	osgraph "github.com/openshift/origin/pkg/api/graph"
	"github.com/openshift/origin/pkg/api/graph/graphview"
	kubeedges "github.com/openshift/origin/pkg/api/kubegraph"
	kubeanalysis "github.com/openshift/origin/pkg/api/kubegraph/analysis"
	kubegraph "github.com/openshift/origin/pkg/api/kubegraph/nodes"
	buildapi "github.com/openshift/origin/pkg/build/api"
	buildedges "github.com/openshift/origin/pkg/build/graph"
	buildanalysis "github.com/openshift/origin/pkg/build/graph/analysis"
	buildgraph "github.com/openshift/origin/pkg/build/graph/nodes"
	"github.com/openshift/origin/pkg/client"
	deployapi "github.com/openshift/origin/pkg/deploy/api"
	deployedges "github.com/openshift/origin/pkg/deploy/graph"
	deployanalysis "github.com/openshift/origin/pkg/deploy/graph/analysis"
	deploygraph "github.com/openshift/origin/pkg/deploy/graph/nodes"
	deployutil "github.com/openshift/origin/pkg/deploy/util"
	imageapi "github.com/openshift/origin/pkg/image/api"
	imageedges "github.com/openshift/origin/pkg/image/graph"
	imagegraph "github.com/openshift/origin/pkg/image/graph/nodes"
	projectapi "github.com/openshift/origin/pkg/project/api"
	routeapi "github.com/openshift/origin/pkg/route/api"
	routeedges "github.com/openshift/origin/pkg/route/graph"
	routeanalysis "github.com/openshift/origin/pkg/route/graph/analysis"
	routegraph "github.com/openshift/origin/pkg/route/graph/nodes"
	"github.com/openshift/origin/pkg/util/errors"
	"github.com/openshift/origin/pkg/util/parallel"
)

const ForbiddenListWarning = "Forbidden"

// ProjectStatusDescriber generates extended information about a Project
type ProjectStatusDescriber struct {
	K       kclient.Interface
	C       client.Interface
	Server  string
	Suggest bool
}

func (d *ProjectStatusDescriber) MakeGraph(namespace string) (osgraph.Graph, sets.String, error) {
	g := osgraph.New()

	loaders := []GraphLoader{
		&serviceLoader{namespace: namespace, lister: d.K},
		&serviceAccountLoader{namespace: namespace, lister: d.K},
		&secretLoader{namespace: namespace, lister: d.K},
		&rcLoader{namespace: namespace, lister: d.K},
		&podLoader{namespace: namespace, lister: d.K},
		// TODO check swagger for feature enablement and selectively add bcLoader and buildLoader
		// then remove errors.TolerateNotFoundError method.
		&bcLoader{namespace: namespace, lister: d.C},
		&buildLoader{namespace: namespace, lister: d.C},
		&isLoader{namespace: namespace, lister: d.C},
		&dcLoader{namespace: namespace, lister: d.C},
		&routeLoader{namespace: namespace, lister: d.C},
	}
	loadingFuncs := []func() error{}
	for _, loader := range loaders {
		loadingFuncs = append(loadingFuncs, loader.Load)
	}

	forbiddenResources := sets.String{}
	if errs := parallel.Run(loadingFuncs...); len(errs) > 0 {
		actualErrors := []error{}
		for _, err := range errs {
			if kapierrors.IsForbidden(err) {
				forbiddenErr := err.(*kapierrors.StatusError)
				if (forbiddenErr.Status().Details != nil) && (len(forbiddenErr.Status().Details.Kind) > 0) {
					forbiddenResources.Insert(forbiddenErr.Status().Details.Kind)
				}
				continue
			}
			actualErrors = append(actualErrors, err)
		}

		if len(actualErrors) > 0 {
			return g, forbiddenResources, utilerrors.NewAggregate(actualErrors)
		}
	}

	for _, loader := range loaders {
		loader.AddToGraph(g)
	}

	kubeedges.AddAllExposedPodTemplateSpecEdges(g)
	kubeedges.AddAllExposedPodEdges(g)
	kubeedges.AddAllManagedByRCPodEdges(g)
	kubeedges.AddAllRequestedServiceAccountEdges(g)
	kubeedges.AddAllMountableSecretEdges(g)
	kubeedges.AddAllMountedSecretEdges(g)
	buildedges.AddAllInputOutputEdges(g)
	buildedges.AddAllBuildEdges(g)
	deployedges.AddAllTriggerEdges(g)
	deployedges.AddAllDeploymentEdges(g)
	imageedges.AddAllImageStreamRefEdges(g)
	routeedges.AddAllRouteEdges(g)

	return g, forbiddenResources, nil
}

// Describe returns the description of a project
func (d *ProjectStatusDescriber) Describe(namespace, name string) (string, error) {
	g, forbiddenResources, err := d.MakeGraph(namespace)
	if err != nil {
		return "", err
	}

	project, err := d.C.Projects().Get(namespace)
	if err != nil {
		return "", err
	}

	coveredNodes := graphview.IntSet{}

	services, coveredByServices := graphview.AllServiceGroups(g, coveredNodes)
	coveredNodes.Insert(coveredByServices.List()...)

	standaloneDCs, coveredByDCs := graphview.AllDeploymentConfigPipelines(g, coveredNodes)
	coveredNodes.Insert(coveredByDCs.List()...)

	standaloneRCs, coveredByRCs := graphview.AllReplicationControllers(g, coveredNodes)
	coveredNodes.Insert(coveredByRCs.List()...)

	standaloneImages, coveredByImages := graphview.AllImagePipelinesFromBuildConfig(g, coveredNodes)
	coveredNodes.Insert(coveredByImages.List()...)

	return tabbedString(func(out *tabwriter.Writer) error {
		indent := "  "
		fmt.Fprintf(out, describeProjectAndServer(project, d.Server))

		for _, service := range services {
			if !service.Service.Found() {
				continue
			}

			fmt.Fprintln(out)
			printLines(out, indent, 0, describeServiceInServiceGroup(service)...)

			for _, dcPipeline := range service.DeploymentConfigPipelines {
				printLines(out, indent, 1, describeDeploymentInServiceGroup(dcPipeline)...)
			}

		rcNode:
			for _, rcNode := range service.FulfillingRCs {
				for _, coveredDC := range service.FulfillingDCs {
					if deployedges.BelongsToDeploymentConfig(coveredDC.DeploymentConfig, rcNode.ReplicationController) {
						continue rcNode
					}
				}
				printLines(out, indent, 1, describeRCInServiceGroup(rcNode)...)
			}

		pod:
			for _, podNode := range service.FulfillingPods {
				// skip pods that have been displayed in a roll-up of RCs and DCs (by implicit usage of RCs)
				for _, coveredRC := range service.FulfillingRCs {
					if g.Edge(podNode, coveredRC) != nil {
						continue pod
					}
				}
				printLines(out, indent, 1, describePodInServiceGroup(podNode)...)
			}

			for _, routeNode := range service.ExposingRoutes {
				printLines(out, indent, 1, describeRouteInServiceGroup(routeNode)...)
			}
		}

		for _, standaloneDC := range standaloneDCs {
			fmt.Fprintln(out)
			printLines(out, indent, 0, describeDeploymentInServiceGroup(standaloneDC)...)
		}

		for _, standaloneImage := range standaloneImages {
			fmt.Fprintln(out)
			lines := describeStandaloneBuildGroup(standaloneImage, namespace)
			lines = append(lines, describeAdditionalBuildDetail(standaloneImage.Build, standaloneImage.LastSuccessfulBuild, standaloneImage.LastUnsuccessfulBuild, standaloneImage.ActiveBuilds, standaloneImage.DestinationResolved, true)...)
			printLines(out, indent, 0, lines...)
		}

		for _, standaloneRC := range standaloneRCs {
			fmt.Fprintln(out)
			printLines(out, indent, 0, describeRCInServiceGroup(standaloneRC.RC)...)
		}

		allMarkers := osgraph.Markers{}
		allMarkers = append(allMarkers, createForbiddenMarkers(forbiddenResources)...)
		for _, scanner := range getMarkerScanners() {
			allMarkers = append(allMarkers, scanner(g)...)
		}

		fmt.Fprintln(out)

		sort.Stable(osgraph.ByKey(allMarkers))
		sort.Stable(osgraph.ByNodeID(allMarkers))

		errorMarkers := allMarkers.BySeverity(osgraph.ErrorSeverity)
		errorSuggestions := 0
		if len(errorMarkers) > 0 {
			fmt.Fprintln(out, "Errors:")
			for _, marker := range errorMarkers {
				fmt.Fprintln(out, indent+"* "+marker.Message)
				if len(marker.Suggestion) > 0 {
					errorSuggestions++
					if d.Suggest {
						fmt.Fprintln(out, indent+"  "+marker.Suggestion.String())
					}
				}
			}
		}

		warningMarkers := allMarkers.BySeverity(osgraph.WarningSeverity)
		if len(warningMarkers) > 0 {
			if d.Suggest {
				fmt.Fprintln(out, "Warnings:")
			}
			for _, marker := range warningMarkers {
				if d.Suggest {
					fmt.Fprintln(out, indent+"* "+marker.Message)
					if len(marker.Suggestion) > 0 {
						fmt.Fprintln(out, indent+"  "+marker.Suggestion.String())
					}
				}
			}
		}

		// We print errors by default and warnings if -v is used. If we get none,
		// this would be an extra new line.
		if len(errorMarkers) != 0 || (d.Suggest && len(warningMarkers) != 0) {
			fmt.Fprintln(out)
		}

		errors, warnings := "", ""
		if len(errorMarkers) == 1 {
			errors = "1 error"
		} else if len(errorMarkers) > 1 {
			errors = fmt.Sprintf("%d errors", len(errorMarkers))
		}
		if len(warningMarkers) == 1 {
			warnings = "1 warning"
		} else if len(warningMarkers) > 1 {
			warnings = fmt.Sprintf("%d warnings", len(warningMarkers))
		}

		switch {
		case !d.Suggest && len(errorMarkers) > 0 && len(warningMarkers) > 0:
			fmt.Fprintf(out, "%s and %s identified, use 'oc status -v' to see details.\n", errors, warnings)

		case !d.Suggest && len(errorMarkers) > 0 && errorSuggestions > 0:
			fmt.Fprintf(out, "%s identified, use 'oc status -v' to see details.\n", errors)

		case !d.Suggest && len(warningMarkers) > 0:
			fmt.Fprintf(out, "%s identified, use 'oc status -v' to see details.\n", warnings)

		case (len(services) == 0) && (len(standaloneDCs) == 0) && (len(standaloneImages) == 0):
			fmt.Fprintln(out, "You have no services, deployment configs, or build configs.")
			fmt.Fprintln(out, "Run 'oc new-app' to create an application.")

		default:
			fmt.Fprintln(out, "View details with 'oc describe <resource>/<name>' or list everything with 'oc get all'.")
		}

		return nil
	})
}

func createForbiddenMarkers(forbiddenResources sets.String) []osgraph.Marker {
	markers := []osgraph.Marker{}
	for forbiddenResource := range forbiddenResources {
		markers = append(markers, osgraph.Marker{
			Severity: osgraph.WarningSeverity,
			Key:      ForbiddenListWarning,
			Message:  fmt.Sprintf("Unable to list %s resources.  Not all status relationships can be established.", forbiddenResource),
		})
	}
	return markers
}

func getMarkerScanners() []osgraph.MarkerScanner {
	return []osgraph.MarkerScanner{
		kubeanalysis.FindRestartingPods,
		kubeanalysis.FindDuelingReplicationControllers,
		kubeanalysis.FindUnmountableSecrets,
		kubeanalysis.FindMissingSecrets,
		buildanalysis.FindUnpushableBuildConfigs,
		buildanalysis.FindCircularBuilds,
		buildanalysis.FindPendingTags,
		deployanalysis.FindDeploymentConfigTriggerErrors,
		routeanalysis.FindMissingPortMapping,
		routeanalysis.FindMissingTLSTerminationType,
	}
}

func printLines(out io.Writer, indent string, depth int, lines ...string) {
	for i, s := range lines {
		fmt.Fprintf(out, strings.Repeat(indent, depth))
		if i != 0 {
			fmt.Fprint(out, indent)
		}
		fmt.Fprintln(out, s)
	}
}

func indentLines(indent string, lines ...string) []string {
	ret := make([]string, 0, len(lines))
	for _, line := range lines {
		ret = append(ret, indent+line)
	}

	return ret
}

func describeProjectAndServer(project *projectapi.Project, server string) string {
	if server != "" {
		return fmt.Sprintf("In project %s on server %s\n", projectapi.DisplayNameAndNameForProject(project), server)
	} else {
		return fmt.Sprintf("In project %s\n", projectapi.DisplayNameAndNameForProject(project))
	}
}

func describeDeploymentInServiceGroup(deploy graphview.DeploymentConfigPipeline) []string {
	includeLastPass := deploy.ActiveDeployment == nil
	if len(deploy.Images) == 1 {
		lines := []string{fmt.Sprintf("%s deploys %s %s", deploy.Deployment.ResourceString(), describeImageInPipeline(deploy.Images[0], deploy.Deployment.Namespace), describeDeploymentConfigTrigger(deploy.Deployment.DeploymentConfig))}
		if len(lines[0]) > 120 && strings.Contains(lines[0], " <- ") {
			segments := strings.SplitN(lines[0], " <- ", 2)
			lines[0] = segments[0] + " <-"
			lines = append(lines, segments[1])
		}
		lines = append(lines, indentLines("  ", describeAdditionalBuildDetail(deploy.Images[0].Build, deploy.Images[0].LastSuccessfulBuild, deploy.Images[0].LastUnsuccessfulBuild, deploy.Images[0].ActiveBuilds, deploy.Images[0].DestinationResolved, includeLastPass)...)...)
		lines = append(lines, describeDeployments(deploy.Deployment, deploy.ActiveDeployment, deploy.InactiveDeployments, 3)...)
		return lines
	}

	lines := []string{fmt.Sprintf("%s deploys: %s", deploy.Deployment.ResourceString(), describeDeploymentConfigTrigger(deploy.Deployment.DeploymentConfig))}
	for _, image := range deploy.Images {
		lines = append(lines, describeImageInPipeline(image, deploy.Deployment.Namespace))
		lines = append(lines, indentLines("  ", describeAdditionalBuildDetail(image.Build, image.LastSuccessfulBuild, image.LastUnsuccessfulBuild, image.ActiveBuilds, image.DestinationResolved, includeLastPass)...)...)
		lines = append(lines, describeDeployments(deploy.Deployment, deploy.ActiveDeployment, deploy.InactiveDeployments, 3)...)
	}
	return lines
}

func describeRCInServiceGroup(rcNode *kubegraph.ReplicationControllerNode) []string {
	if rcNode.ReplicationController.Spec.Template == nil {
		return []string{}
	}

	images := []string{}
	for _, container := range rcNode.ReplicationController.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	lines := []string{fmt.Sprintf("%s runs %s", rcNode.ResourceString(), strings.Join(images, ", "))}
	lines = append(lines, describeRCStatus(rcNode.ReplicationController))

	return lines
}

func describePodInServiceGroup(podNode *kubegraph.PodNode) []string {
	images := []string{}
	for _, container := range podNode.Pod.Spec.Containers {
		images = append(images, container.Image)
	}

	lines := []string{fmt.Sprintf("%s runs %s", podNode.ResourceString(), strings.Join(images, ", "))}
	return lines
}

func describeRouteInServiceGroup(routeNode *routegraph.RouteNode) []string {
	if routeNode.Spec.Port != nil && len(routeNode.Spec.Port.TargetPort.String()) > 0 {
		return []string{fmt.Sprintf("exposed by %s on pod port %s", routeNode.ResourceString(), routeNode.Spec.Port.TargetPort.String())}
	}
	return []string{fmt.Sprintf("exposed by %s", routeNode.ResourceString())}
}

func describeDeploymentConfigTrigger(dc *deployapi.DeploymentConfig) string {
	if len(dc.Spec.Triggers) == 0 {
		return "(manual)"
	}

	return ""
}

func describeStandaloneBuildGroup(pipeline graphview.ImagePipeline, namespace string) []string {
	switch {
	case pipeline.Build != nil:
		lines := []string{describeBuildInPipeline(pipeline.Build.BuildConfig, pipeline.BaseImage)}
		if pipeline.Image != nil {
			lines = append(lines, fmt.Sprintf("pushes to %s", describeImageTagInPipeline(pipeline.Image, namespace)))
		}
		return lines
	case pipeline.Image != nil:
		return []string{describeImageTagInPipeline(pipeline.Image, namespace)}
	default:
		return []string{"<unknown>"}
	}
}

func describeImageInPipeline(pipeline graphview.ImagePipeline, namespace string) string {
	switch {
	case pipeline.Image != nil && pipeline.Build != nil:
		return fmt.Sprintf("%s <- %s", describeImageTagInPipeline(pipeline.Image, namespace), describeBuildInPipeline(pipeline.Build.BuildConfig, pipeline.BaseImage))
	case pipeline.Image != nil:
		return describeImageTagInPipeline(pipeline.Image, namespace)
	case pipeline.Build != nil:
		return describeBuildInPipeline(pipeline.Build.BuildConfig, pipeline.BaseImage)
	default:
		return "<unknown>"
	}
}

func describeImageTagInPipeline(image graphview.ImageTagLocation, namespace string) string {
	switch t := image.(type) {
	case *imagegraph.ImageStreamTagNode:
		if t.ImageStreamTag.Namespace != namespace {
			return image.ImageSpec()
		}
		return t.ResourceString()
	default:
		return image.ImageSpec()
	}
}

func describeBuildInPipeline(build *buildapi.BuildConfig, baseImage graphview.ImageTagLocation) string {
	switch {
	case build.Spec.Strategy.DockerStrategy != nil:
		// TODO: handle case where no source repo
		source, ok := describeSourceInPipeline(&build.Spec.Source)
		if !ok {
			return fmt.Sprintf("bc/%s unconfigured docker build - no source set", build.Name)
		}
		return fmt.Sprintf("bc/%s docker build of %s", build.Name, source)
	case build.Spec.Strategy.SourceStrategy != nil:
		source, ok := describeSourceInPipeline(&build.Spec.Source)
		if !ok {
			return fmt.Sprintf("bc/%s unconfigured source build", build.Name)
		}
		if baseImage == nil {
			return fmt.Sprintf("bc/%s %s; no image set", build.Name, source)
		}
		return fmt.Sprintf("bc/%s builds %s with %s", build.Name, source, baseImage.ImageSpec())
	case build.Spec.Strategy.CustomStrategy != nil:
		source, ok := describeSourceInPipeline(&build.Spec.Source)
		if !ok {
			return fmt.Sprintf("bc/%s custom build ", build.Name)
		}
		return fmt.Sprintf("bc/%s custom build of %s", build.Name, source)
	default:
		return fmt.Sprintf("bc/%s unrecognized build", build.Name)
	}
}

func describeAdditionalBuildDetail(build *buildgraph.BuildConfigNode, lastSuccessfulBuild *buildgraph.BuildNode, lastUnsuccessfulBuild *buildgraph.BuildNode, activeBuilds []*buildgraph.BuildNode, pushTargetResolved bool, includeSuccess bool) []string {
	if build == nil {
		return nil
	}
	out := []string{}

	passTime := unversioned.Time{}
	if lastSuccessfulBuild != nil {
		passTime = buildTimestamp(lastSuccessfulBuild.Build)
	}
	failTime := unversioned.Time{}
	if lastUnsuccessfulBuild != nil {
		failTime = buildTimestamp(lastUnsuccessfulBuild.Build)
	}

	lastTime := failTime
	if passTime.After(failTime.Time) {
		lastTime = passTime
	}

	// display the last successful build if specifically requested or we're going to display an active build for context
	if lastSuccessfulBuild != nil && (includeSuccess || len(activeBuilds) > 0) {
		out = append(out, describeBuildPhase(lastSuccessfulBuild.Build, &passTime, build.BuildConfig.Name, pushTargetResolved))
	}
	if passTime.Before(failTime) {
		out = append(out, describeBuildPhase(lastUnsuccessfulBuild.Build, &failTime, build.BuildConfig.Name, pushTargetResolved))
	}

	if len(activeBuilds) > 0 {
		activeOut := []string{}
		for i := range activeBuilds {
			activeOut = append(activeOut, describeBuildPhase(activeBuilds[i].Build, nil, build.BuildConfig.Name, pushTargetResolved))
		}

		if buildTimestamp(activeBuilds[0].Build).Before(lastTime) {
			out = append(out, activeOut...)
		} else {
			out = append(activeOut, out...)
		}
	}
	if len(out) == 0 && lastSuccessfulBuild == nil {
		out = append(out, "not built yet")
	}
	return out
}

func describeBuildPhase(build *buildapi.Build, t *unversioned.Time, parentName string, pushTargetResolved bool) string {
	imageStreamFailure := ""
	// if we're using an image stream and that image stream is the internal registry and that registry doesn't exist
	if (build.Spec.Output.To != nil) && !pushTargetResolved {
		imageStreamFailure = " (can't push to image)"
	}

	if t == nil {
		ts := buildTimestamp(build)
		t = &ts
	}
	var time string
	if t.IsZero() {
		time = "<unknown>"
	} else {
		time = strings.ToLower(formatRelativeTime(t.Time))
	}
	buildIdentification := fmt.Sprintf("build/%s", build.Name)
	prefix := parentName + "-"
	if strings.HasPrefix(build.Name, prefix) {
		suffix := build.Name[len(prefix):]

		if buildNumber, err := strconv.Atoi(suffix); err == nil {
			buildIdentification = fmt.Sprintf("#%d build", buildNumber)
		}
	}

	revision := describeSourceRevision(build.Spec.Revision)
	if len(revision) != 0 {
		revision = fmt.Sprintf(" - %s", revision)
	}
	switch build.Status.Phase {
	case buildapi.BuildPhaseComplete:
		return fmt.Sprintf("%s succeeded %s ago%s%s", buildIdentification, time, revision, imageStreamFailure)
	case buildapi.BuildPhaseError:
		return fmt.Sprintf("%s stopped with an error %s ago%s%s", buildIdentification, time, revision, imageStreamFailure)
	case buildapi.BuildPhaseFailed:
		return fmt.Sprintf("%s failed %s ago%s%s", buildIdentification, time, revision, imageStreamFailure)
	default:
		status := strings.ToLower(string(build.Status.Phase))
		return fmt.Sprintf("%s %s for %s%s%s", buildIdentification, status, time, revision, imageStreamFailure)
	}
}

func describeSourceRevision(rev *buildapi.SourceRevision) string {
	if rev == nil {
		return ""
	}
	switch {
	case rev.Git != nil:
		author := describeSourceControlUser(rev.Git.Author)
		if len(author) == 0 {
			author = describeSourceControlUser(rev.Git.Committer)
		}
		if len(author) != 0 {
			author = fmt.Sprintf(" (%s)", author)
		}
		commit := rev.Git.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		return fmt.Sprintf("%s: %s%s", commit, rev.Git.Message, author)
	default:
		return ""
	}
}

func describeSourceControlUser(user buildapi.SourceControlUser) string {
	if len(user.Name) == 0 {
		return user.Email
	}
	if len(user.Email) == 0 {
		return user.Name
	}
	return fmt.Sprintf("%s <%s>", user.Name, user.Email)
}

func buildTimestamp(build *buildapi.Build) unversioned.Time {
	if build == nil {
		return unversioned.Time{}
	}
	if !build.Status.CompletionTimestamp.IsZero() {
		return *build.Status.CompletionTimestamp
	}
	if !build.Status.StartTimestamp.IsZero() {
		return *build.Status.StartTimestamp
	}
	return build.CreationTimestamp
}

func describeSourceInPipeline(source *buildapi.BuildSource) (string, bool) {
	switch {
	case source.Git != nil:
		if len(source.Git.Ref) == 0 {
			return source.Git.URI, true
		}
		return fmt.Sprintf("%s#%s", source.Git.URI, source.Git.Ref), true
	case source.Dockerfile != nil:
		return "Dockerfile", true
	}
	return "", false
}

func describeDeployments(dcNode *deploygraph.DeploymentConfigNode, activeDeployment *kubegraph.ReplicationControllerNode, inactiveDeployments []*kubegraph.ReplicationControllerNode, count int) []string {
	if dcNode == nil {
		return nil
	}
	out := []string{}
	deploymentsToPrint := append([]*kubegraph.ReplicationControllerNode{}, inactiveDeployments...)

	if activeDeployment == nil {
		on, auto := describeDeploymentConfigTriggers(dcNode.DeploymentConfig)
		if dcNode.DeploymentConfig.Status.LatestVersion == 0 {
			out = append(out, fmt.Sprintf("#1 deployment waiting %s", on))
		} else if auto {
			out = append(out, fmt.Sprintf("#%d deployment pending %s", dcNode.DeploymentConfig.Status.LatestVersion, on))
		}
		// TODO: detect new image available?
	} else {
		deploymentsToPrint = append([]*kubegraph.ReplicationControllerNode{activeDeployment}, inactiveDeployments...)
	}

	for i, deployment := range deploymentsToPrint {
		out = append(out, describeDeploymentStatus(deployment.ReplicationController, i == 0))

		switch {
		case count == -1:
			if deployutil.DeploymentStatusFor(deployment) == deployapi.DeploymentStatusComplete {
				return out
			}
		default:
			if i+1 >= count {
				return out
			}
		}
	}
	return out
}

func describeDeploymentStatus(deploy *kapi.ReplicationController, first bool) string {
	timeAt := strings.ToLower(formatRelativeTime(deploy.CreationTimestamp.Time))
	status := deployutil.DeploymentStatusFor(deploy)
	version := deployutil.DeploymentVersionFor(deploy)
	switch status {
	case deployapi.DeploymentStatusFailed:
		reason := deployutil.DeploymentStatusReasonFor(deploy)
		if len(reason) > 0 {
			reason = fmt.Sprintf(": %s", reason)
		}
		// TODO: encode fail time in the rc
		return fmt.Sprintf("#%d deployment failed %s ago%s%s", version, timeAt, reason, describePodSummaryInline(deploy, false))
	case deployapi.DeploymentStatusComplete:
		// TODO: pod status output
		return fmt.Sprintf("#%d deployed %s ago%s", version, timeAt, describePodSummaryInline(deploy, first))
	case deployapi.DeploymentStatusRunning:
		return fmt.Sprintf("#%d deployment running for %s%s", version, timeAt, describePodSummaryInline(deploy, false))
	default:
		return fmt.Sprintf("#%d deployment %s %s ago%s", version, strings.ToLower(string(status)), timeAt, describePodSummaryInline(deploy, false))
	}
}

func describeRCStatus(rc *kapi.ReplicationController) string {
	timeAt := strings.ToLower(formatRelativeTime(rc.CreationTimestamp.Time))
	return fmt.Sprintf("rc/%s created %s ago%s", rc.Name, timeAt, describePodSummaryInline(rc, false))
}

func describePodSummaryInline(rc *kapi.ReplicationController, includeEmpty bool) string {
	s := describePodSummary(rc, includeEmpty)
	if len(s) == 0 {
		return s
	}
	change := ""
	desired := rc.Spec.Replicas
	switch {
	case desired < rc.Status.Replicas:
		change = fmt.Sprintf(" reducing to %d", desired)
	case desired > rc.Status.Replicas:
		change = fmt.Sprintf(" growing to %d", desired)
	}
	return fmt.Sprintf(" - %s%s", s, change)
}

func describePodSummary(rc *kapi.ReplicationController, includeEmpty bool) string {
	actual, requested := rc.Status.Replicas, rc.Spec.Replicas
	if actual == requested {
		switch {
		case actual == 0:
			if !includeEmpty {
				return ""
			}
			return "0 pods"
		case actual > 1:
			return fmt.Sprintf("%d pods", actual)
		default:
			return "1 pod"
		}
	}
	return fmt.Sprintf("%d/%d pods", actual, requested)
}

func describeDeploymentConfigTriggers(config *deployapi.DeploymentConfig) (string, bool) {
	hasConfig, hasImage := false, false
	for _, t := range config.Spec.Triggers {
		switch t.Type {
		case deployapi.DeploymentTriggerOnConfigChange:
			hasConfig = true
		case deployapi.DeploymentTriggerOnImageChange:
			hasImage = true
		}
	}
	switch {
	case hasConfig && hasImage:
		return "on image or update", true
	case hasConfig:
		return "on update", true
	case hasImage:
		return "on image", true
	default:
		return "for manual", false
	}
}

func describeServiceInServiceGroup(svc graphview.ServiceGroup) []string {
	spec := svc.Service.Spec
	ip := spec.ClusterIP
	port := describeServicePorts(spec)
	switch {
	case ip == "None":
		return []string{fmt.Sprintf("%s (headless)%s", svc.Service.ResourceString(), port)}
	case len(ip) == 0:
		return []string{fmt.Sprintf("%s <initializing>%s", svc.Service.ResourceString(), port)}
	default:
		return []string{fmt.Sprintf("%s - %s%s", svc.Service.ResourceString(), ip, port)}
	}
}

func describeServicePorts(spec kapi.ServiceSpec) string {
	switch len(spec.Ports) {
	case 0:
		return " no ports"

	case 1:
		if spec.Ports[0].TargetPort.String() == "0" || spec.ClusterIP == kapi.ClusterIPNone || spec.Ports[0].Port == spec.Ports[0].TargetPort.IntVal {
			return fmt.Sprintf(":%d", spec.Ports[0].Port)
		}
		return fmt.Sprintf(":%d -> %s", spec.Ports[0].Port, spec.Ports[0].TargetPort.String())

	default:
		pairs := []string{}
		for _, port := range spec.Ports {
			if port.TargetPort.String() == "0" || spec.ClusterIP == kapi.ClusterIPNone {
				pairs = append(pairs, fmt.Sprintf("%d", port.Port))
				continue
			}
			if port.Port == port.TargetPort.IntVal {
				pairs = append(pairs, port.TargetPort.String())
			} else {
				pairs = append(pairs, fmt.Sprintf("%d->%s", port.Port, port.TargetPort.String()))
			}
		}
		return " ports " + strings.Join(pairs, ", ")
	}
}

// GraphLoader is a stateful interface that provides methods for building the nodes of a graph
type GraphLoader interface {
	// Load is responsible for gathering and saving the objects this GraphLoader should AddToGraph
	Load() error
	// AddToGraph
	AddToGraph(g osgraph.Graph) error
}

type rcLoader struct {
	namespace string
	lister    kclient.ReplicationControllersNamespacer
	items     []kapi.ReplicationController
}

func (l *rcLoader) Load() error {
	list, err := l.lister.ReplicationControllers(l.namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return err
	}

	l.items = list.Items
	return nil
}

func (l *rcLoader) AddToGraph(g osgraph.Graph) error {
	for i := range l.items {
		kubegraph.EnsureReplicationControllerNode(g, &l.items[i])
	}

	return nil
}

type serviceLoader struct {
	namespace string
	lister    kclient.ServicesNamespacer
	items     []kapi.Service
}

func (l *serviceLoader) Load() error {
	list, err := l.lister.Services(l.namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return err
	}

	l.items = list.Items
	return nil
}

func (l *serviceLoader) AddToGraph(g osgraph.Graph) error {
	for i := range l.items {
		kubegraph.EnsureServiceNode(g, &l.items[i])
	}

	return nil
}

type podLoader struct {
	namespace string
	lister    kclient.PodsNamespacer
	items     []kapi.Pod
}

func (l *podLoader) Load() error {
	list, err := l.lister.Pods(l.namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return err
	}

	l.items = list.Items
	return nil
}

func (l *podLoader) AddToGraph(g osgraph.Graph) error {
	for i := range l.items {
		kubegraph.EnsurePodNode(g, &l.items[i])
	}

	return nil
}

type serviceAccountLoader struct {
	namespace string
	lister    kclient.ServiceAccountsNamespacer
	items     []kapi.ServiceAccount
}

func (l *serviceAccountLoader) Load() error {
	list, err := l.lister.ServiceAccounts(l.namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return err
	}

	l.items = list.Items
	return nil
}

func (l *serviceAccountLoader) AddToGraph(g osgraph.Graph) error {
	for i := range l.items {
		kubegraph.EnsureServiceAccountNode(g, &l.items[i])
	}

	return nil
}

type secretLoader struct {
	namespace string
	lister    kclient.SecretsNamespacer
	items     []kapi.Secret
}

func (l *secretLoader) Load() error {
	list, err := l.lister.Secrets(l.namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return err
	}

	l.items = list.Items
	return nil
}

func (l *secretLoader) AddToGraph(g osgraph.Graph) error {
	for i := range l.items {
		kubegraph.EnsureSecretNode(g, &l.items[i])
	}

	return nil
}

type isLoader struct {
	namespace string
	lister    client.ImageStreamsNamespacer
	items     []imageapi.ImageStream
}

func (l *isLoader) Load() error {
	list, err := l.lister.ImageStreams(l.namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return err
	}

	l.items = list.Items
	return nil
}

func (l *isLoader) AddToGraph(g osgraph.Graph) error {
	for i := range l.items {
		imagegraph.EnsureImageStreamNode(g, &l.items[i])
		imagegraph.EnsureAllImageStreamTagNodes(g, &l.items[i])
	}

	return nil
}

type dcLoader struct {
	namespace string
	lister    client.DeploymentConfigsNamespacer
	items     []deployapi.DeploymentConfig
}

func (l *dcLoader) Load() error {
	list, err := l.lister.DeploymentConfigs(l.namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return err
	}

	l.items = list.Items
	return nil
}

func (l *dcLoader) AddToGraph(g osgraph.Graph) error {
	for i := range l.items {
		deploygraph.EnsureDeploymentConfigNode(g, &l.items[i])
	}

	return nil
}

type bcLoader struct {
	namespace string
	lister    client.BuildConfigsNamespacer
	items     []buildapi.BuildConfig
}

func (l *bcLoader) Load() error {
	list, err := l.lister.BuildConfigs(l.namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return errors.TolerateNotFoundError(err)
	}

	l.items = list.Items
	return nil
}

func (l *bcLoader) AddToGraph(g osgraph.Graph) error {
	for i := range l.items {
		buildgraph.EnsureBuildConfigNode(g, &l.items[i])
	}

	return nil
}

type buildLoader struct {
	namespace string
	lister    client.BuildsNamespacer
	items     []buildapi.Build
}

func (l *buildLoader) Load() error {
	list, err := l.lister.Builds(l.namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return errors.TolerateNotFoundError(err)
	}

	l.items = list.Items
	return nil
}

func (l *buildLoader) AddToGraph(g osgraph.Graph) error {
	for i := range l.items {
		buildgraph.EnsureBuildNode(g, &l.items[i])
	}

	return nil
}

type routeLoader struct {
	namespace string
	lister    client.RoutesNamespacer
	items     []routeapi.Route
}

func (l *routeLoader) Load() error {
	list, err := l.lister.Routes(l.namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return err
	}

	l.items = list.Items
	return nil
}

func (l *routeLoader) AddToGraph(g osgraph.Graph) error {
	for i := range l.items {
		routegraph.EnsureRouteNode(g, &l.items[i])
	}

	return nil
}
