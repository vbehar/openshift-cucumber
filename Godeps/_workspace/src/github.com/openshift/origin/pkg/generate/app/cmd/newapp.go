package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/golang/glog"
	kapi "k8s.io/kubernetes/pkg/api"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/validation"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/errors"

	authapi "github.com/openshift/origin/pkg/authorization/api"
	buildapi "github.com/openshift/origin/pkg/build/api"
	"github.com/openshift/origin/pkg/client"
	cmdutil "github.com/openshift/origin/pkg/cmd/util"
	"github.com/openshift/origin/pkg/dockerregistry"
	"github.com/openshift/origin/pkg/generate/app"
	"github.com/openshift/origin/pkg/generate/dockerfile"
	"github.com/openshift/origin/pkg/generate/source"
	imageapi "github.com/openshift/origin/pkg/image/api"
	"github.com/openshift/origin/pkg/template"
	outil "github.com/openshift/origin/pkg/util"
	dockerfileutil "github.com/openshift/origin/pkg/util/docker/dockerfile"
)

const (
	GeneratedByNamespace = "openshift.io/generated-by"
	GeneratedForJob      = "openshift.io/generated-job"
	GeneratedForJobFor   = "openshift.io/generated-job.for"
	GeneratedByNewApp    = "OpenShiftNewApp"
	GeneratedByNewBuild  = "OpenShiftNewBuild"
)

// ErrNoDockerfileDetected is the error returned when the requested build strategy is Docker
// and no Dockerfile is detected in the repository.
var ErrNoDockerfileDetected = fmt.Errorf("No Dockerfile was found in the repository and the requested build strategy is 'docker'")

// AppConfig contains all the necessary configuration for an application
type AppConfig struct {
	SourceRepositories []string
	ContextDir         string

	Components    []string
	ImageStreams  []string
	DockerImages  []string
	Templates     []string
	TemplateFiles []string

	TemplateParameters []string
	Groups             []string
	Environment        []string
	Labels             map[string]string

	AddEnvironmentToBuild bool

	Dockerfile string

	Name             string
	To               string
	Strategy         string
	InsecureRegistry bool
	OutputDocker     bool
	NoOutput         bool

	ExpectToBuild      bool
	BinaryBuild        bool
	AllowMissingImages bool
	Deploy             bool

	SkipGeneration        bool
	AllowGenerationErrors bool

	AllowSecretUse bool
	SecretAccessor app.SecretAccessor

	AsSearch bool
	AsList   bool
	DryRun   bool

	Out    io.Writer
	ErrOut io.Writer

	KubeClient kclient.Interface

	refBuilder *app.ReferenceBuilder

	dockerSearcher                  app.Searcher
	imageStreamSearcher             app.Searcher
	imageStreamByAnnotationSearcher app.Searcher
	templateSearcher                app.Searcher
	templateFileSearcher            app.Searcher

	detector app.Detector

	typer        runtime.ObjectTyper
	mapper       meta.RESTMapper
	clientMapper resource.ClientMapper

	osclient        client.Interface
	originNamespace string
}

// UsageError is an interface for printing usage errors
type UsageError interface {
	UsageError(commandName string) string
}

// TODO: replace with upstream converting [1]error to error
type errlist interface {
	Errors() []error
}

type ErrRequiresExplicitAccess struct {
	Match app.ComponentMatch
	Input app.GeneratorInput
}

func (e ErrRequiresExplicitAccess) Error() string {
	return fmt.Sprintf("the component %q is requesting access to run with your security credentials and install components - you must explicitly grant that access to continue", e.Match.String())
}

// ErrNoInputs is returned when no inputs are specified
var ErrNoInputs = fmt.Errorf("no inputs provided")

// AppResult contains the results of an application
type AppResult struct {
	List *kapi.List

	Name      string
	HasSource bool
	Namespace string

	GeneratedJobs bool
}

// QueryResult contains the results of a query (search or list)
type QueryResult struct {
	Matches app.ComponentMatches
	List    *kapi.List
}

// NewAppConfig returns a new AppConfig, but you must set your typer, mapper, and clientMapper after the command has been run
// and flags have been parsed.
func NewAppConfig() *AppConfig {
	return &AppConfig{
		detector: app.SourceRepositoryEnumerator{
			Detectors: source.DefaultDetectors,
			Tester:    dockerfile.NewTester(),
		},
		refBuilder: &app.ReferenceBuilder{},
	}
}

func (c *AppConfig) SetMapper(mapper meta.RESTMapper) {
	c.mapper = mapper
}

func (c *AppConfig) SetTyper(typer runtime.ObjectTyper) {
	c.typer = typer
}

func (c *AppConfig) SetClientMapper(clientMapper resource.ClientMapper) {
	c.clientMapper = clientMapper
}

func (c *AppConfig) dockerRegistrySearcher() app.Searcher {
	return app.DockerRegistrySearcher{
		Client:        dockerregistry.NewClient(30 * time.Second),
		AllowInsecure: c.InsecureRegistry,
	}
}

func (c *AppConfig) ensureDockerSearcher() {
	if c.dockerSearcher == nil {
		c.dockerSearcher = c.dockerRegistrySearcher()
	}
}

// SetDockerClient sets the passed Docker client in the application configuration
func (c *AppConfig) SetDockerClient(dockerclient *docker.Client) {
	c.dockerSearcher = app.DockerClientSearcher{
		Client:             dockerclient,
		RegistrySearcher:   c.dockerRegistrySearcher(),
		Insecure:           c.InsecureRegistry,
		AllowMissingImages: c.AllowMissingImages,
	}
}

// SetOpenShiftClient sets the passed OpenShift client in the application configuration
func (c *AppConfig) SetOpenShiftClient(osclient client.Interface, originNamespace string) {
	c.osclient = osclient
	c.originNamespace = originNamespace
	namespaces := []string{originNamespace}
	if openshiftNamespace := "openshift"; originNamespace != openshiftNamespace {
		namespaces = append(namespaces, openshiftNamespace)
	}
	c.imageStreamSearcher = app.ImageStreamSearcher{
		Client:            osclient,
		ImageStreamImages: osclient,
		Namespaces:        namespaces,
	}
	c.imageStreamByAnnotationSearcher = app.NewImageStreamByAnnotationSearcher(osclient, osclient, namespaces)
	c.templateSearcher = app.TemplateSearcher{
		Client: osclient,
		TemplateConfigsNamespacer: osclient,
		Namespaces:                namespaces,
	}
	c.templateFileSearcher = &app.TemplateFileSearcher{
		Typer:        c.typer,
		Mapper:       c.mapper,
		ClientMapper: c.clientMapper,
		Namespace:    originNamespace,
	}
}

// AddArguments converts command line arguments into the appropriate bucket based on what they look like
func (c *AppConfig) AddArguments(args []string) []string {
	unknown := []string{}
	for _, s := range args {
		switch {
		case cmdutil.IsEnvironmentArgument(s):
			c.Environment = append(c.Environment, s)
		case app.IsPossibleSourceRepository(s):
			c.SourceRepositories = append(c.SourceRepositories, s)
		case app.IsComponentReference(s):
			c.Components = append(c.Components, s)
		case app.IsPossibleTemplateFile(s):
			c.Components = append(c.Components, s)
		default:
			if len(s) == 0 {
				break
			}
			unknown = append(unknown, s)
		}
	}
	return unknown
}

// individualSourceRepositories collects the list of SourceRepositories specified in the
// command line that are not associated with a builder using a '~'.
func (c *AppConfig) individualSourceRepositories() (app.SourceRepositories, error) {
	for _, s := range c.SourceRepositories {
		if repo, ok := c.refBuilder.AddSourceRepository(s); ok {
			repo.SetContextDir(c.ContextDir)
			if c.Strategy == "docker" {
				repo.BuildWithDocker()
			}
		}
	}
	if len(c.Dockerfile) > 0 {
		if err := c.addDockerfile(); err != nil {
			return nil, err
		}
	}
	_, repos, errs := c.refBuilder.Result()
	return repos, errors.NewAggregate(errs)
}

// addDockerfile adds a Dockerfile passed in the command line to the reference
// builder.
func (c *AppConfig) addDockerfile() error {
	if len(c.Strategy) != 0 && c.Strategy != "docker" {
		return fmt.Errorf("when directly referencing a Dockerfile, the strategy must must be 'docker'")
	}
	_, repos, errs := c.refBuilder.Result()
	if err := errors.NewAggregate(errs); err != nil {
		return err
	}
	switch len(repos) {
	case 0:
		// Create a new SourceRepository with the Dockerfile.
		repo, err := app.NewSourceRepositoryForDockerfile(c.Dockerfile)
		if err != nil {
			return fmt.Errorf("provided Dockerfile is not valid: %v", err)
		}
		c.refBuilder.AddExistingSourceRepository(repo)
	case 1:
		// Add the Dockerfile to the existing SourceRepository, so that
		// eventually we generate a single BuildConfig with multiple
		// sources.
		if err := repos[0].AddDockerfile(c.Dockerfile); err != nil {
			return fmt.Errorf("provided Dockerfile is not valid: %v", err)
		}
	default:
		// Invalid.
		return fmt.Errorf("--dockerfile cannot be used with multiple source repositories")
	}
	return nil
}

// set up the components to be used by the reference builder
func (c *AppConfig) addReferenceBuilderComponents(b *app.ReferenceBuilder) {
	b.AddComponents(c.DockerImages, func(input *app.ComponentInput) app.ComponentReference {
		input.Argument = fmt.Sprintf("--docker-image=%q", input.From)
		input.Searcher = c.dockerSearcher
		if c.dockerSearcher != nil {
			resolver := app.PerfectMatchWeightedResolver{}
			resolver = append(resolver, app.WeightedResolver{Searcher: c.dockerSearcher, Weight: 0.0})
			if c.AllowMissingImages {
				resolver = append(resolver, app.WeightedResolver{Searcher: app.MissingImageSearcher{}, Weight: 100.0})
			}
			input.Resolver = resolver
		}
		return input
	})
	b.AddComponents(c.ImageStreams, func(input *app.ComponentInput) app.ComponentReference {
		input.Argument = fmt.Sprintf("--image-stream=%q", input.From)
		input.Searcher = c.imageStreamSearcher
		if c.imageStreamSearcher != nil {
			input.Resolver = app.FirstMatchResolver{Searcher: c.imageStreamSearcher}
		}
		return input
	})
	b.AddComponents(c.Templates, func(input *app.ComponentInput) app.ComponentReference {
		input.Argument = fmt.Sprintf("--template=%q", input.From)
		input.Searcher = c.templateSearcher
		if c.templateSearcher != nil {
			input.Resolver = app.HighestScoreResolver{Searcher: c.templateSearcher}
		}
		return input
	})
	b.AddComponents(c.TemplateFiles, func(input *app.ComponentInput) app.ComponentReference {
		input.Argument = fmt.Sprintf("--file=%q", input.From)
		input.Searcher = c.templateFileSearcher
		if c.templateFileSearcher != nil {
			input.Resolver = app.FirstMatchResolver{Searcher: c.templateFileSearcher}
		}
		return input
	})
	b.AddComponents(c.Components, func(input *app.ComponentInput) app.ComponentReference {
		resolver := app.PerfectMatchWeightedResolver{}
		searcher := app.MultiWeightedSearcher{}
		if c.imageStreamSearcher != nil {
			resolver = append(resolver, app.WeightedResolver{Searcher: c.imageStreamSearcher, Weight: 0.0})
			searcher = append(searcher, app.WeightedSearcher{Searcher: c.imageStreamSearcher, Weight: 0.0})
		}
		if c.templateSearcher != nil {
			resolver = append(resolver, app.WeightedResolver{Searcher: c.templateSearcher, Weight: 0.0})
			searcher = append(searcher, app.WeightedSearcher{Searcher: c.templateSearcher, Weight: 0.0})
		}
		if c.templateFileSearcher != nil {
			resolver = append(resolver, app.WeightedResolver{Searcher: c.templateFileSearcher, Weight: 0.0})
		}
		if c.dockerSearcher != nil {
			resolver = append(resolver, app.WeightedResolver{Searcher: c.dockerSearcher, Weight: 2.0})
			searcher = append(searcher, app.WeightedSearcher{Searcher: c.dockerSearcher, Weight: 1.0})
		}
		if c.AllowMissingImages {
			resolver = append(resolver, app.WeightedResolver{Searcher: app.MissingImageSearcher{}, Weight: 100.0})
		}
		input.Resolver = resolver
		input.Searcher = searcher
		return input
	})

	_, repos, _ := b.Result()
	for _, repo := range repos {
		repo.SetContextDir(c.ContextDir)
	}
}

// validate converts all of the arguments on the config into references to objects, or returns an error
func (c *AppConfig) validate() (app.ComponentReferences, app.SourceRepositories, cmdutil.Environment, cmdutil.Environment, error) {
	b := c.refBuilder
	c.addReferenceBuilderComponents(b)
	b.AddGroups(c.Groups)
	refs, repos, errs := b.Result()

	if len(c.Strategy) != 0 && len(repos) == 0 {
		errs = append(errs, fmt.Errorf("when --strategy is specified you must provide at least one source code location"))
	}

	if c.BinaryBuild && (len(repos) > 0 || refs.HasSource()) {
		errs = append(errs, fmt.Errorf("specifying binary builds and source repositories at the same time is not allowed"))
	}

	env, duplicateEnv, envErrs := cmdutil.ParseEnvironmentArguments(c.Environment)
	for _, s := range duplicateEnv {
		glog.V(1).Infof("The environment variable %q was overwritten", s)
	}
	errs = append(errs, envErrs...)

	parms, duplicateParms, parmsErrs := cmdutil.ParseEnvironmentArguments(c.TemplateParameters)
	for _, s := range duplicateParms {
		glog.V(1).Infof("The template parameter %q was overwritten", s)
	}
	errs = append(errs, parmsErrs...)

	return refs, repos, env, parms, errors.NewAggregate(errs)
}

// componentsForRepos creates components for repositories that have not been previously associated by a builder
// these components have already gone through source code detection and have a SourceRepositoryInfo attached to them
func (c *AppConfig) componentsForRepos(repositories app.SourceRepositories) (app.ComponentReferences, error) {
	b := c.refBuilder
	errs := []error{}
	result := app.ComponentReferences{}
	for _, repo := range repositories {
		info := repo.Info()
		switch {
		case info == nil:
			errs = append(errs, fmt.Errorf("source not detected for repository %q", repo))
			continue
		case info.Dockerfile != nil && (len(c.Strategy) == 0 || c.Strategy == "docker"):
			node := info.Dockerfile.AST()
			baseImage := dockerfileutil.LastBaseImage(node)
			if baseImage == "" {
				errs = append(errs, fmt.Errorf("the Dockerfile in the repository %q has no FROM instruction", info.Path))
				continue
			}
			refs := b.AddComponents([]string{baseImage}, func(input *app.ComponentInput) app.ComponentReference {
				resolver := app.PerfectMatchWeightedResolver{}
				if c.imageStreamSearcher != nil {
					resolver = append(resolver, app.WeightedResolver{Searcher: c.imageStreamSearcher, Weight: 0.0})
				}
				if c.dockerSearcher != nil {
					resolver = append(resolver, app.WeightedResolver{Searcher: c.dockerSearcher, Weight: 1.0})
				}
				resolver = append(resolver, app.WeightedResolver{Searcher: &app.PassThroughDockerSearcher{}, Weight: 2.0})
				input.Resolver = resolver
				input.Use(repo)
				input.ExpectToBuild = true
				repo.UsedBy(input)
				repo.BuildWithDocker()
				return input
			})
			result = append(result, refs...)
		default:
			// TODO: Add support for searching for more than one language if len(info.Types) > 1
			if len(info.Types) == 0 {
				errs = append(errs, fmt.Errorf("no language was detected for repository at %q; please specify a builder image to use with your repository: [builder-image]~%s", repo, repo))

				continue
			}
			refs := b.AddComponents([]string{info.Types[0].Term()}, func(input *app.ComponentInput) app.ComponentReference {
				resolver := app.PerfectMatchWeightedResolver{}
				if c.imageStreamByAnnotationSearcher != nil {
					resolver = append(resolver, app.WeightedResolver{Searcher: c.imageStreamByAnnotationSearcher, Weight: 0.0})
				}
				if c.imageStreamSearcher != nil {
					resolver = append(resolver, app.WeightedResolver{Searcher: c.imageStreamSearcher, Weight: 1.0})
				}
				if c.dockerSearcher != nil {
					resolver = append(resolver, app.WeightedResolver{Searcher: c.dockerSearcher, Weight: 2.0})
				}
				input.Resolver = resolver
				input.ExpectToBuild = true
				input.Use(repo)
				repo.UsedBy(input)
				return input
			})
			result = append(result, refs...)
		}
	}
	return result, errors.NewAggregate(errs)
}

// resolve the references to ensure they are all valid, and identify any images that don't match user input.
func (c *AppConfig) resolve(components app.ComponentReferences) error {
	errs := []error{}
	for _, ref := range components {
		if err := ref.Resolve(); err != nil {
			errs = append(errs, err)
			continue
		}
	}
	return errors.NewAggregate(errs)
}

// searches on all references
func (c *AppConfig) search(components app.ComponentReferences) error {
	errs := []error{}
	for _, ref := range components {
		if err := ref.Search(); err != nil {
			errs = append(errs, err)
			continue
		}
	}
	return errors.NewAggregate(errs)
}

// inferBuildTypes infers build status and mismatches between source and docker builders
func (c *AppConfig) inferBuildTypes(components app.ComponentReferences) (app.ComponentReferences, error) {
	errs := []error{}
	for _, ref := range components {
		input := ref.Input()

		// identify whether the input is a builder and whether generation is requested
		input.ResolvedMatch.Builder = app.IsBuilderMatch(input.ResolvedMatch)
		generatorInput, err := app.GeneratorInputFromMatch(input.ResolvedMatch)
		if err != nil && !c.AllowGenerationErrors {
			errs = append(errs, err)
			continue
		}
		input.ResolvedMatch.GeneratorInput = generatorInput

		// if the strategy is explicitly Docker, all repos should assume docker
		if c.Strategy == "docker" && input.Uses != nil {
			input.Uses.BuildWithDocker()
		}

		// if we are expecting build inputs, or get a build input when strategy is not docker, expect to build
		if c.ExpectToBuild || (input.ResolvedMatch.Builder && c.Strategy != "docker") {
			input.ExpectToBuild = true
		}

		switch {
		case input.ExpectToBuild && input.ResolvedMatch.IsTemplate():
			// TODO: harder - break the template pieces and check if source code can be attached (look for a build config, build image, etc)
			errs = append(errs, fmt.Errorf("template with source code explicitly attached is not supported - you must either specify the template and source code separately or attach an image to the source code using the '[image]~[code]' form"))
			continue
		case input.ExpectToBuild && !input.ResolvedMatch.Builder && input.Uses != nil && !input.Uses.IsDockerBuild():
			if len(c.Strategy) == 0 {
				errs = append(errs, fmt.Errorf("the resolved match %q for component %q cannot build source code - check whether this is the image you want to use, then use --strategy=source to build using source or --strategy=docker to treat this as a Docker base image and set up a layered Docker build", input.ResolvedMatch.Name, ref))
				continue
			}
		case input.ResolvedMatch.Score != 0.0:
			errs = append(errs, fmt.Errorf("component %q had only a partial match of %q - if this is the value you want to use, specify it explicitly", input.From, input.ResolvedMatch.Name))
			continue
		}
	}
	if len(components) == 0 && c.BinaryBuild {
		if len(c.Name) == 0 {
			return nil, fmt.Errorf("you must provide a --name when you don't specify a source repository or base image")
		}
		ref := &app.ComponentInput{
			From:          "--binary",
			Argument:      "--binary",
			Value:         c.Name,
			ScratchImage:  true,
			ExpectToBuild: true,
		}
		components = append(components, ref)
	}

	return components, errors.NewAggregate(errs)
}

// ensureHasSource ensure every builder component has source code associated with it. It takes a list of component references
// that are builders and have not been associated with source, and a set of source repositories that have not been associated
// with a builder
func (c *AppConfig) ensureHasSource(components app.ComponentReferences, repositories app.SourceRepositories) error {
	if len(components) > 0 {
		switch {
		case len(repositories) > 1:
			if len(components) == 1 {
				component := components[0]
				suggestions := ""

				for _, repo := range repositories {
					suggestions += fmt.Sprintf("%s~%s\n", component, repo)
				}
				return fmt.Errorf("there are multiple code locations provided - use one of the following suggestions to declare which code goes with the image:\n%s", suggestions)
			}
			return fmt.Errorf("the following images require source code: %s\n"+
				" and the following repositories are not used: %s\nUse '[image]~[repo]' to declare which code goes with which image", components, repositories)
		case len(repositories) == 1:
			glog.Infof("Using %q as the source for build", repositories[0])
			for _, component := range components {
				component.Input().Use(repositories[0])
				repositories[0].UsedBy(component)
			}
		default:
			switch {
			case c.BinaryBuild && c.ExpectToBuild:
				// create new "fake" binary repos for any component that doesn't already have a repo
				// TODO: source repository should possibly be refactored to be an interface or a type that better reflects
				//   the different types of inputs
				for _, component := range components {
					input := component.Input()
					if input.Uses != nil {
						continue
					}
					repo := app.NewBinarySourceRepository()
					if c.Strategy == "docker" || len(c.Strategy) == 0 {
						repo.BuildWithDocker()
					}
					input.Use(repo)
					repo.UsedBy(input)
					input.ExpectToBuild = true
				}
			case c.ExpectToBuild:
				return fmt.Errorf("you must specify at least one source repository URL, provide a Dockerfile, or indicate you wish to use binary builds")
			default:
				for _, component := range components {
					component.Input().ExpectToBuild = false
				}
			}
		}
	}
	return nil
}

// detectSource runs a code detector on the passed in repositories to obtain a SourceRepositoryInfo
func (c *AppConfig) detectSource(repositories []*app.SourceRepository) error {
	errs := []error{}
	for _, repo := range repositories {
		err := repo.Detect(c.detector, c.Strategy == "docker")
		if err != nil {
			if c.Strategy == "docker" && err == app.ErrNoLanguageDetected {
				errs = append(errs, ErrNoDockerfileDetected)
			} else {
				errs = append(errs, err)
			}
			continue
		}
	}
	return errors.NewAggregate(errs)
}

func validateEnforcedName(name string) error {
	if ok, _ := validation.ValidateServiceName(name, false); !ok {
		return fmt.Errorf("invalid name: %s. Must be an a lower case alphanumeric (a-z, and 0-9) string with a maximum length of 24 characters, where the first character is a letter (a-z), and the '-' character is allowed anywhere except the first or last character.", name)
	}
	return nil
}

func validateOutputImageReference(ref string) error {
	if _, err := imageapi.ParseDockerImageReference(ref); err != nil {
		return fmt.Errorf("invalid output image reference: %s", ref)
	}
	return nil
}

// buildPipelines converts a set of resolved, valid references into pipelines.
func (c *AppConfig) buildPipelines(components app.ComponentReferences, environment app.Environment) (app.PipelineGroup, error) {
	pipelines := app.PipelineGroup{}
	pipelineBuilder := app.NewPipelineBuilder(c.Name, c.GetBuildEnvironment(environment), c.OutputDocker).To(c.To)
	for _, group := range components.Group() {
		glog.V(4).Infof("found group: %v", group)
		common := app.PipelineGroup{}
		for _, ref := range group {
			refInput := ref.Input()
			from := refInput.String()
			var (
				pipeline *app.Pipeline
				err      error
			)
			switch {
			case refInput.ExpectToBuild:
				glog.V(4).Infof("will use %q as the base image for a source build of %q", ref, refInput.Uses)
				if pipeline, err = pipelineBuilder.NewBuildPipeline(from, refInput.ResolvedMatch, refInput.Uses); err != nil {
					return nil, fmt.Errorf("can't build %q: %v", refInput.Uses, err)
				}
			default:
				glog.V(4).Infof("will include %q", ref)
				if pipeline, err = pipelineBuilder.NewImagePipeline(from, refInput.ResolvedMatch); err != nil {
					return nil, fmt.Errorf("can't include %q: %v", refInput, err)
				}
			}
			if c.Deploy {
				if err := pipeline.NeedsDeployment(environment, c.Labels); err != nil {
					return nil, fmt.Errorf("can't set up a deployment for %q: %v", refInput, err)
				}
			}
			if c.NoOutput {
				pipeline.Build.Output = nil
			}
			if err := pipeline.Validate(); err != nil {
				switch err.(type) {
				case app.CircularOutputReferenceError:
					if len(c.To) == 0 {
						// Output reference was generated, return error.
						return nil, err
					}
					// Output reference was explicitly provided, print warning.
					fmt.Fprintf(c.ErrOut, "--> WARNING: %v\n", err)
				default:
					return nil, err
				}
			}
			common = append(common, pipeline)
			if err := common.Reduce(); err != nil {
				return nil, fmt.Errorf("can't create a pipeline from %s: %v", common, err)
			}
			describeBuildPipelineWithImage(c.Out, ref, pipeline, c.originNamespace)
		}
		pipelines = append(pipelines, common...)
	}
	return pipelines, nil
}

// buildTemplates converts a set of resolved, valid references into references to template objects.
func (c *AppConfig) buildTemplates(components app.ComponentReferences, environment app.Environment) ([]runtime.Object, error) {
	objects := []runtime.Object{}

	for _, ref := range components {
		tpl := ref.Input().ResolvedMatch.Template

		glog.V(4).Infof("processing template %s/%s", c.originNamespace, tpl.Name)
		for _, env := range environment.List() {
			// only set environment values that match what's expected by the template.
			if v := template.GetParameterByName(tpl, env.Name); v != nil {
				v.Value = env.Value
				v.Generate = ""
				template.AddParameter(tpl, *v)
			} else {
				return nil, fmt.Errorf("unexpected parameter name %q", env.Name)
			}
		}

		result, err := c.osclient.TemplateConfigs(c.originNamespace).Create(tpl)
		if err != nil {
			return nil, fmt.Errorf("error processing template %s/%s: %v", c.originNamespace, tpl.Name, err)
		}
		errs := runtime.DecodeList(result.Objects, kapi.Scheme)
		if len(errs) > 0 {
			err = errors.NewAggregate(errs)
			return nil, fmt.Errorf("error processing template %s/%s: %v", c.originNamespace, tpl.Name, errs)
		}
		objects = append(objects, result.Objects...)

		describeGeneratedTemplate(c.Out, ref, result, c.originNamespace)
	}
	return objects, nil
}

// fakeSecretAccessor is used during dry runs of installation
type fakeSecretAccessor struct {
	token string
}

func (a *fakeSecretAccessor) Token() (string, error) {
	return a.token, nil
}
func (a *fakeSecretAccessor) CACert() (string, error) {
	return "", nil
}

// installComponents attempts to create pods to run installable images identified by the user. If an image
// is installable, we check whether it requires access to the user token. If so, the caller must have
// explicitly granted that access (because the token may be the user's).
func (c *AppConfig) installComponents(components app.ComponentReferences, env app.Environment) ([]runtime.Object, string, error) {
	if c.SkipGeneration {
		return nil, "", nil
	}

	jobs := components.InstallableComponentRefs()
	switch {
	case len(jobs) > 1:
		return nil, "", fmt.Errorf("only one installable component may be provided: %s", jobs.HumanString(", "))
	case len(jobs) == 0:
		return nil, "", nil
	}

	job := jobs[0]
	if len(components) > 1 {
		return nil, "", fmt.Errorf("%q is installable and may not be specified with other components", job.Input().Value)
	}
	input := job.Input()

	imageRef, err := app.InputImageFromMatch(input.ResolvedMatch)
	if err != nil {
		return nil, "", fmt.Errorf("can't include %q: %v", input, err)
	}
	glog.V(4).Infof("Resolved match for installer %#v", input.ResolvedMatch)

	imageRef.AsImageStream = false
	imageRef.AsResolvedImage = true
	imageRef.Env = env

	name := c.Name
	if len(name) == 0 {
		var ok bool
		name, ok = imageRef.SuggestName()
		if !ok {
			return nil, "", fmt.Errorf("can't suggest a valid name, please specify a name with --name")
		}
	}
	imageRef.ObjectName = name
	glog.V(4).Infof("Proposed installable image %#v", imageRef)

	secretAccessor := c.SecretAccessor
	generatorInput := input.ResolvedMatch.GeneratorInput
	token := generatorInput.Token
	if token != nil && !c.AllowSecretUse || secretAccessor == nil {
		if !c.DryRun {
			return nil, "", ErrRequiresExplicitAccess{Match: *input.ResolvedMatch, Input: generatorInput}
		}
		secretAccessor = &fakeSecretAccessor{token: "FAKE_TOKEN"}
	}

	objects := []runtime.Object{}

	serviceAccountName := "installer"
	if token != nil && token.ServiceAccount {
		if _, err := c.KubeClient.ServiceAccounts(c.originNamespace).Get(serviceAccountName); err != nil {
			if kerrors.IsNotFound(err) {
				objects = append(objects,
					// create a new service account
					&kapi.ServiceAccount{ObjectMeta: kapi.ObjectMeta{Name: serviceAccountName}},
					// grant the service account the edit role on the project (TODO: installer)
					&authapi.RoleBinding{
						ObjectMeta: kapi.ObjectMeta{Name: "installer-role-binding"},
						Subjects:   []kapi.ObjectReference{{Kind: "ServiceAccount", Name: serviceAccountName}},
						RoleRef:    kapi.ObjectReference{Name: "edit"},
					},
				)
			}
		}
	}

	pod, secret, err := imageRef.InstallablePod(generatorInput, secretAccessor, serviceAccountName)
	if err != nil {
		return nil, "", err
	}
	objects = append(objects, pod)
	if secret != nil {
		objects = append(objects, secret)
	}
	for i := range objects {
		outil.AddObjectAnnotations(objects[i], map[string]string{
			GeneratedForJob:    "true",
			GeneratedForJobFor: input.String(),
		})
	}

	describeGeneratedJob(c.Out, job, pod, secret, c.originNamespace)

	return objects, name, nil
}

// Run executes the provided config to generate objects.
func (c *AppConfig) Run() (*AppResult, error) {
	return c.run(app.Acceptors{app.NewAcceptUnique(c.typer), app.AcceptNew})
}

// RunQuery executes the provided config and returns the result of the resolution.
func (c *AppConfig) RunQuery() (*QueryResult, error) {
	c.ensureDockerSearcher()
	repositories, err := c.individualSourceRepositories()
	if err != nil {
		return nil, err
	}

	if c.AsList {
		if c.AsSearch {
			return nil, fmt.Errorf("--list and --search can't be used together")
		}
		if c.HasArguments() {
			return nil, fmt.Errorf("--list can't be used with arguments")
		}
		c.Components = append(c.Components, "*")
	}

	components, repositories, environment, parameters, err := c.validate()
	if err != nil {
		return nil, err
	}

	if len(components) == 0 && !c.AsList {
		return nil, ErrNoInputs
	}

	errs := []error{}
	if len(repositories) > 0 {
		errs = append(errs, fmt.Errorf("--search can't be used with source code"))
	}
	if len(environment) > 0 {
		errs = append(errs, fmt.Errorf("--search can't be used with --env"))
	}
	if len(parameters) > 0 {
		errs = append(errs, fmt.Errorf("--search can't be used with --param"))
	}
	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}

	if err := c.search(components); err != nil {
		return nil, err
	}

	glog.V(4).Infof("Code %v", repositories)
	glog.V(4).Infof("Components %v", components)

	matches := app.ComponentMatches{}
	objects := app.Objects{}
	for _, ref := range components {
		for _, match := range ref.Input().SearchMatches {
			matches = append(matches, match)
			if match.IsTemplate() {
				objects = append(objects, match.Template)
			} else if match.IsImage() {
				if match.ImageStream != nil {
					objects = append(objects, match.ImageStream)
				}
				if match.Image != nil {
					objects = append(objects, match.Image)
				}
			}
		}
	}
	return &QueryResult{
		Matches: matches,
		List:    &kapi.List{Items: objects},
	}, nil
}

// run executes the provided config applying provided acceptors.
func (c *AppConfig) run(acceptors app.Acceptors) (*AppResult, error) {
	c.ensureDockerSearcher()
	repositories, err := c.individualSourceRepositories()
	if err != nil {
		return nil, err
	}
	err = c.detectSource(repositories)
	if err != nil {
		return nil, err
	}
	components, repositories, environment, parameters, err := c.validate()
	if err != nil {
		return nil, err
	}

	if err := c.resolve(components); err != nil {
		return nil, err
	}

	components, err = c.inferBuildTypes(components)
	if err != nil {
		return nil, err
	}

	// Couple source with resolved builder components if possible
	if err := c.ensureHasSource(components.NeedsSource(), repositories.NotUsed()); err != nil {
		return nil, err
	}

	// For source repos that are not yet coupled with a component, create components
	sourceComponents, err := c.componentsForRepos(repositories.NotUsed())
	if err != nil {
		return nil, err
	}

	// resolve the source repo components
	if err := c.resolve(sourceComponents); err != nil {
		return nil, err
	}
	components = append(components, sourceComponents...)

	glog.V(4).Infof("Code [%v]", repositories)
	glog.V(4).Infof("Components [%v]", components)

	if len(repositories) == 0 && len(components) == 0 {
		return nil, ErrNoInputs
	}

	if len(c.Name) > 0 {
		if err := validateEnforcedName(c.Name); err != nil {
			return nil, err
		}
	}

	if len(c.To) > 0 {
		if err := validateOutputImageReference(c.To); err != nil {
			return nil, err
		}
	}

	imageRefs := components.ImageComponentRefs()
	if len(imageRefs) > 1 && len(c.Name) > 0 {
		return nil, fmt.Errorf("only one component or source repository can be used when specifying a name")
	}
	if len(imageRefs) > 1 && len(c.To) > 0 {
		return nil, fmt.Errorf("only one component or source repository can be used when specifying an output image reference")
	}

	env := app.Environment(environment)

	// identify if there are installable components in the input provided by the user
	installables, name, err := c.installComponents(components, env)
	if err != nil {
		return nil, err
	}
	if len(installables) > 0 {
		return &AppResult{
			List:      &kapi.List{Items: installables},
			Name:      name,
			Namespace: c.originNamespace,

			GeneratedJobs: true,
		}, nil
	}

	pipelines, err := c.buildPipelines(imageRefs, env)
	if err != nil {
		if err == app.ErrNameRequired {
			return nil, fmt.Errorf("can't suggest a valid name, please specify a name with --name")
		}
		if err, ok := err.(app.CircularOutputReferenceError); ok {
			return nil, fmt.Errorf("%v, please specify a different output reference with --to", err)
		}
		return nil, err
	}

	objects := app.Objects{}
	accept := app.NewAcceptFirst()
	for _, p := range pipelines {
		accepted, err := p.Objects(accept, acceptors)
		if err != nil {
			return nil, fmt.Errorf("can't setup %q: %v", p.From, err)
		}
		objects = append(objects, accepted...)
	}

	objects = app.AddServices(objects, false)

	templateObjects, err := c.buildTemplates(components.TemplateComponentRefs(), app.Environment(parameters))
	if err != nil {
		return nil, err
	}
	objects = append(objects, templateObjects...)

	name = c.Name
	if len(name) == 0 {
		for _, pipeline := range pipelines {
			if pipeline.Deployment != nil {
				name = pipeline.Deployment.Name
				break
			}
		}
	}
	if len(name) == 0 {
		for _, obj := range objects {
			if bc, ok := obj.(*buildapi.BuildConfig); ok {
				name = bc.Name
				break
			}
		}
	}

	return &AppResult{
		List:      &kapi.List{Items: objects},
		Name:      name,
		HasSource: len(repositories) != 0,
		Namespace: c.originNamespace,
	}, nil
}

func (c *AppConfig) Querying() bool {
	return c.AsList || c.AsSearch
}

func (c *AppConfig) HasArguments() bool {
	return len(c.Components) > 0 ||
		len(c.ImageStreams) > 0 ||
		len(c.DockerImages) > 0 ||
		len(c.Templates) > 0 ||
		len(c.TemplateFiles) > 0
}

func (c *AppConfig) GetBuildEnvironment(environment app.Environment) app.Environment {
	if c.AddEnvironmentToBuild {
		return environment
	}
	return app.Environment{}
}
