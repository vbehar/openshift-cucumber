package app

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/golang/glog"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/validation"
	"k8s.io/kubernetes/pkg/runtime"
	kutil "k8s.io/kubernetes/pkg/util"
	kuval "k8s.io/kubernetes/pkg/util/validation"

	deploy "github.com/openshift/origin/pkg/deploy/api"
	image "github.com/openshift/origin/pkg/image/api"
	route "github.com/openshift/origin/pkg/route/api"
	"github.com/openshift/origin/pkg/util/docker/dockerfile"
)

// A PipelineBuilder creates Pipeline instances.
type PipelineBuilder interface {
	To(string) PipelineBuilder

	NewBuildPipeline(string, *ComponentMatch, *SourceRepository) (*Pipeline, error)
	NewImagePipeline(string, *ComponentMatch) (*Pipeline, error)
}

// NewPipelineBuilder returns a PipelineBuilder using name as a base name. A
// PipelineBuilder always creates pipelines with unique names, so that the
// actual name of a pipeline (Pipeline.Name) might differ from the base name.
// The pipelines created with a PipelineBuilder will have access to the given
// environment. The boolean outputDocker controls whether builds will output to
// an image stream tag or docker image reference.
func NewPipelineBuilder(name string, environment Environment, outputDocker bool) PipelineBuilder {
	return &pipelineBuilder{
		nameGenerator: NewUniqueNameGenerator(name),
		environment:   environment,
		outputDocker:  outputDocker,
	}
}

type pipelineBuilder struct {
	nameGenerator UniqueNameGenerator
	environment   Environment
	outputDocker  bool
	to            string
}

func (pb *pipelineBuilder) To(name string) PipelineBuilder {
	pb.to = name
	return pb
}

// NewBuildPipeline creates a new pipeline with components that are expected to
// be built.
func (pb *pipelineBuilder) NewBuildPipeline(from string, resolvedMatch *ComponentMatch, sourceRepository *SourceRepository) (*Pipeline, error) {
	var input *ImageRef
	if resolvedMatch != nil {
		inputImage, err := InputImageFromMatch(resolvedMatch)
		if err != nil {
			return nil, fmt.Errorf("can't build %q: %v", from, err)
		}
		input = inputImage
		if !input.AsImageStream {
			msg := "Could not find an image stream match for %q. Make sure that a Docker image with that tag is available on the node for the build to succeed."
			glog.Warningf(msg, resolvedMatch.Value)
		}
	}

	strategy, source, err := StrategyAndSourceForRepository(sourceRepository, input)
	if err != nil {
		return nil, fmt.Errorf("can't build %q: %v", from, err)
	}

	var name string
	output := &ImageRef{
		OutputImage:   true,
		AsImageStream: !pb.outputDocker,
	}
	if len(pb.to) > 0 {
		outputImageRef, err := image.ParseDockerImageReference(pb.to)
		if err != nil {
			return nil, err
		}
		output.Reference = outputImageRef
		name, err = pb.nameGenerator.Generate(NameSuggestions{source, output, input})
		if err != nil {
			return nil, err
		}
	} else {
		name, err = pb.nameGenerator.Generate(NameSuggestions{source, input})
		if err != nil {
			return nil, err
		}
		output.Reference = image.DockerImageReference{
			Name: name,
			Tag:  image.DefaultImageTag,
		}
	}
	source.Name = name

	// Append any exposed ports from Dockerfile to input image
	if sourceRepository.IsDockerBuild() && sourceRepository.Info() != nil {
		node := sourceRepository.Info().Dockerfile.AST()
		ports := dockerfile.LastExposedPorts(node)
		if len(ports) > 0 {
			if input.Info == nil {
				input.Info = &image.DockerImage{
					Config: &image.DockerConfig{},
				}
			}
			input.Info.Config.ExposedPorts = map[string]struct{}{}
			for _, p := range ports {
				input.Info.Config.ExposedPorts[p] = struct{}{}
			}
		}
	}

	if input != nil {
		// TODO: assumes that build doesn't change the image metadata. In the future
		// we could get away with deferred generation possibly.
		output.Info = input.Info
	}

	build := &BuildRef{
		Source:   source,
		Input:    input,
		Strategy: strategy,
		Output:   output,
		Env:      pb.environment,
	}

	return &Pipeline{
		Name:       name,
		From:       from,
		InputImage: input,
		Image:      output,
		Build:      build,
	}, nil
}

// NewImagePipeline creates a new pipeline with components that are not expected
// to be built.
func (pb *pipelineBuilder) NewImagePipeline(from string, resolvedMatch *ComponentMatch) (*Pipeline, error) {
	input, err := InputImageFromMatch(resolvedMatch)
	if err != nil {
		return nil, fmt.Errorf("can't include %q: %v", from, err)
	}

	name, err := pb.nameGenerator.Generate(input)
	if err != nil {
		return nil, err
	}
	input.ObjectName = name

	return &Pipeline{
		Name:  name,
		From:  from,
		Image: input,
	}, nil
}

// Pipeline holds components.
type Pipeline struct {
	Name string
	From string

	InputImage *ImageRef
	Build      *BuildRef
	Image      *ImageRef
	Deployment *DeploymentConfigRef
	Labels     map[string]string
}

// NeedsDeployment sets the pipeline for deployment.
func (p *Pipeline) NeedsDeployment(env Environment, labels map[string]string) error {
	if p.Deployment != nil {
		return nil
	}
	p.Deployment = &DeploymentConfigRef{
		Name: p.Name,
		Images: []*ImageRef{
			p.Image,
		},
		Env:    env,
		Labels: labels,
	}
	return nil
}

// Validate checks for logical errors in the pipeline.
func (p *Pipeline) Validate() error {
	if p == nil || p.Build == nil {
		return nil
	}
	input, output := p.Build.Input, p.Build.Output
	if input == nil || output == nil {
		return nil
	}
	if input.AsImageStream && output.AsImageStream && input.Reference.Equal(output.Reference) {
		// If the build input and output image stream tags are the same, given that
		// build configs created by new-app/new-build have an image change trigger,
		// this setup would cause an infinite build loop, most likely unintentionaly.
		return CircularOutputReferenceError{Reference: output.Reference}
	}
	return nil
}

// Objects converts all the components in the pipeline into runtime objects.
func (p *Pipeline) Objects(accept, objectAccept Acceptor) (Objects, error) {
	objects := Objects{}
	if p.InputImage != nil && p.InputImage.AsImageStream && accept.Accept(p.InputImage) {
		repo, err := p.InputImage.ImageStream()
		if err != nil {
			return nil, err
		}
		if objectAccept.Accept(repo) {
			objects = append(objects, repo)
		}
	}
	if p.Image != nil && p.Image.AsImageStream && accept.Accept(p.Image) {
		repo, err := p.Image.ImageStream()
		if err != nil {
			return nil, err
		}
		if objectAccept.Accept(repo) {
			objects = append(objects, repo)
		}
	}
	if p.Build != nil && accept.Accept(p.Build) {
		build, err := p.Build.BuildConfig()
		if err != nil {
			return nil, err
		}
		if objectAccept.Accept(build) {
			objects = append(objects, build)
		}
	}
	if p.Deployment != nil && accept.Accept(p.Deployment) {
		dc, err := p.Deployment.DeploymentConfig()
		if err != nil {
			return nil, err
		}
		if objectAccept.Accept(dc) {
			objects = append(objects, dc)
		}
	}
	return objects, nil
}

// PipelineGroup is a group of Pipelines.
type PipelineGroup []*Pipeline

// Reduce squashes all common components from the pipelines.
func (g PipelineGroup) Reduce() error {
	var deployment *DeploymentConfigRef
	for _, p := range g {
		if p.Deployment == nil || p.Deployment == deployment {
			continue
		}
		if deployment == nil {
			deployment = p.Deployment
		} else {
			deployment.Images = append(deployment.Images, p.Deployment.Images...)
			deployment.Env = NewEnvironment(deployment.Env, p.Deployment.Env)
			p.Deployment = deployment
		}
	}
	return nil
}

func (g PipelineGroup) String() string {
	s := []string{}
	for _, p := range g {
		s = append(s, p.From)
	}
	return strings.Join(s, "+")
}

var invalidServiceChars = regexp.MustCompile("[^-a-z0-9]")

func makeValidServiceName(name string) (string, string) {
	if ok, _ := validation.ValidateServiceName(name, false); ok {
		return name, ""
	}
	name = strings.ToLower(name)
	name = invalidServiceChars.ReplaceAllString(name, "")
	name = strings.TrimFunc(name, func(r rune) bool { return r == '-' })
	switch {
	case len(name) == 0:
		return "", "service-"
	case len(name) > kuval.DNS952LabelMaxLength:
		name = name[:kuval.DNS952LabelMaxLength]
	}
	return name, ""
}

type sortablePorts []kapi.ContainerPort

func (s sortablePorts) Len() int           { return len(s) }
func (s sortablePorts) Less(i, j int) bool { return s[i].ContainerPort < s[j].ContainerPort }
func (s sortablePorts) Swap(i, j int) {
	p := s[i]
	s[i] = s[j]
	s[j] = p
}

// portName returns a unique key for the given port and protocol which can be
// used as a service port name.
func portName(port int, protocol kapi.Protocol) string {
	if protocol == "" {
		protocol = kapi.ProtocolTCP
	}
	return strings.ToLower(fmt.Sprintf("%d-%s", port, protocol))
}

// AddServices sets up services for the provided objects.
func AddServices(objects Objects, firstPortOnly bool) Objects {
	svcs := []runtime.Object{}
	for _, o := range objects {
		switch t := o.(type) {
		case *deploy.DeploymentConfig:
			name, generateName := makeValidServiceName(t.Name)
			svc := &kapi.Service{
				ObjectMeta: kapi.ObjectMeta{
					Name:         name,
					GenerateName: generateName,
					Labels:       t.Labels,
				},
				Spec: kapi.ServiceSpec{
					Selector: t.Spec.Selector,
				},
			}

			svcPorts := map[string]struct{}{}
			for _, container := range t.Spec.Template.Spec.Containers {
				ports := sortablePorts(container.Ports)
				sort.Sort(&ports)
				for _, p := range ports {
					name := portName(p.ContainerPort, p.Protocol)
					_, exists := svcPorts[name]
					if exists {
						continue
					}
					svcPorts[name] = struct{}{}
					svc.Spec.Ports = append(svc.Spec.Ports, kapi.ServicePort{
						Name:       name,
						Port:       p.ContainerPort,
						Protocol:   p.Protocol,
						TargetPort: kutil.NewIntOrStringFromInt(p.ContainerPort),
					})
					if firstPortOnly {
						break
					}
				}
			}
			if len(svc.Spec.Ports) == 0 {
				continue
			}
			svcs = append(svcs, svc)
		}
	}
	return append(objects, svcs...)
}

// AddRoutes sets up routes for the provided objects.
func AddRoutes(objects Objects) Objects {
	routes := []runtime.Object{}
	for _, o := range objects {
		switch t := o.(type) {
		case *kapi.Service:
			routes = append(routes, &route.Route{
				ObjectMeta: kapi.ObjectMeta{
					Name:   t.Name,
					Labels: t.Labels,
				},
				Spec: route.RouteSpec{
					To: kapi.ObjectReference{
						Name: t.Name,
					},
				},
			})
		}
	}
	return append(objects, routes...)
}

type acceptNew struct{}

// AcceptNew only accepts runtime.Objects with an empty resource version.
var AcceptNew Acceptor = acceptNew{}

// Accept accepts any kind of object.
func (acceptNew) Accept(from interface{}) bool {
	_, meta, err := objectMetaData(from)
	if err != nil {
		return false
	}
	if len(meta.ResourceVersion) > 0 {
		return false
	}
	return true
}

type acceptUnique struct {
	typer   runtime.ObjectTyper
	objects map[string]struct{}
}

// Accept accepts any kind of object it hasn't accepted before.
func (a *acceptUnique) Accept(from interface{}) bool {
	obj, meta, err := objectMetaData(from)
	if err != nil {
		return false
	}
	_, kind, err := a.typer.ObjectVersionAndKind(obj)
	if err != nil {
		return false
	}
	key := fmt.Sprintf("%s/%s/%s", kind, meta.Namespace, meta.Name)
	_, exists := a.objects[key]
	if exists {
		return false
	}
	a.objects[key] = struct{}{}
	return true
}

// NewAcceptUnique creates an acceptor that only accepts unique objects by kind
// and name.
func NewAcceptUnique(typer runtime.ObjectTyper) Acceptor {
	return &acceptUnique{
		typer:   typer,
		objects: map[string]struct{}{},
	}
}

func objectMetaData(raw interface{}) (runtime.Object, *kapi.ObjectMeta, error) {
	obj, ok := raw.(runtime.Object)
	if !ok {
		return nil, nil, fmt.Errorf("%#v is not a runtime.Object", raw)
	}
	meta, err := kapi.ObjectMetaFor(obj)
	if err != nil {
		return nil, nil, err
	}
	return obj, meta, nil
}

type acceptBuildConfigs struct {
	typer runtime.ObjectTyper
}

// Accept accepts BuildConfigs and ImageStreams.
func (a *acceptBuildConfigs) Accept(from interface{}) bool {
	obj, _, err := objectMetaData(from)
	if err != nil {
		return false
	}
	_, kind, err := a.typer.ObjectVersionAndKind(obj)
	if err != nil {
		return false
	}
	return kind == "BuildConfig" || kind == "ImageStream"
}

// NewAcceptBuildConfigs creates an acceptor accepting BuildConfig objects
// and ImageStreams objects.
func NewAcceptBuildConfigs(typer runtime.ObjectTyper) Acceptor {
	return &acceptBuildConfigs{
		typer: typer,
	}
}

// Acceptors is a list of acceptors that behave like a single acceptor.
// All acceptors must accept an object for it to be accepted.
type Acceptors []Acceptor

// Accept iterates through all acceptors and determines whether the object
// should be accepted.
func (aa Acceptors) Accept(from interface{}) bool {
	for _, a := range aa {
		if !a.Accept(from) {
			return false
		}
	}
	return true
}

type acceptAll struct{}

// AcceptAll accepts all objects.
var AcceptAll Acceptor = acceptAll{}

// Accept accepts everything.
func (acceptAll) Accept(_ interface{}) bool {
	return true
}

// Objects is a set of runtime objects.
type Objects []runtime.Object

// Acceptor is an interface for accepting objects.
type Acceptor interface {
	Accept(from interface{}) bool
}

type acceptFirst struct {
	handled map[interface{}]struct{}
}

// NewAcceptFirst returns a new Acceptor.
func NewAcceptFirst() Acceptor {
	return &acceptFirst{make(map[interface{}]struct{})}
}

// Accept accepts any object it hasn't accepted before.
func (s *acceptFirst) Accept(from interface{}) bool {
	if _, ok := s.handled[from]; ok {
		return false
	}
	s.handled[from] = struct{}{}
	return true
}
