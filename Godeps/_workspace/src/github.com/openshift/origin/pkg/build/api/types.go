package api

import (
	"time"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/util/sets"
)

const (
	// BuildAnnotation is an annotation that identifies a Pod as being for a Build
	BuildAnnotation = "openshift.io/build.name"
	// BuildNumberAnnotation is an annotation whose value is the sequential number for this Build
	BuildNumberAnnotation = "openshift.io/build.number"
	// BuildCloneAnnotation is an annotation whose value is the name of the build this build was cloned from
	BuildCloneAnnotation = "openshift.io/build.clone-of"
	// BuildPodNameAnnotation is an annotation whose value is the name of the pod running this build
	BuildPodNameAnnotation = "openshift.io/build.pod-name"
	// BuildLabel is the key of a Pod label whose value is the Name of a Build which is run.
	BuildLabel = "openshift.io/build.name"
	// DefaultDockerLabelNamespace is the key of a Build label, whose values are build metadata.
	DefaultDockerLabelNamespace = "io.openshift."
)

// Build encapsulates the inputs needed to produce a new deployable image, as well as
// the status of the execution and a reference to the Pod which executed the build.
type Build struct {
	unversioned.TypeMeta
	kapi.ObjectMeta

	// Spec is all the inputs used to execute the build.
	Spec BuildSpec

	// Status is the current status of the build.
	Status BuildStatus
}

// BuildSpec encapsulates all the inputs necessary to represent a build.
type BuildSpec struct {
	// ServiceAccount is the name of the ServiceAccount to use to run the pod
	// created by this build.
	// The pod will be allowed to use secrets referenced by the ServiceAccount
	ServiceAccount string

	// Source describes the SCM in use.
	Source BuildSource

	// Revision is the information from the source for a specific repo snapshot.
	// This is optional.
	Revision *SourceRevision

	// Strategy defines how to perform a build.
	Strategy BuildStrategy

	// Output describes the Docker image the Strategy should produce.
	Output BuildOutput

	// Compute resource requirements to execute the build
	Resources kapi.ResourceRequirements

	// Optional duration in seconds, counted from the time when a build pod gets
	// scheduled in the system, that the build may be active on a node before the
	// system actively tries to terminate the build; value must be positive integer
	CompletionDeadlineSeconds *int64
}

// BuildStatus contains the status of a build
type BuildStatus struct {
	// Phase is the point in the build lifecycle.
	Phase BuildPhase

	// Cancelled describes if a cancel event was triggered for the build.
	Cancelled bool

	// Reason is a brief CamelCase string that describes any failure and is meant for machine parsing and tidy display in the CLI.
	Reason StatusReason

	// Message is a human-readable message indicating details about why the build has this status.
	Message string

	// StartTimestamp is a timestamp representing the server time when this Build started
	// running in a Pod.
	// It is represented in RFC3339 form and is in UTC.
	StartTimestamp *unversioned.Time

	// CompletionTimestamp is a timestamp representing the server time when this Build was
	// finished, whether that build failed or succeeded.  It reflects the time at which
	// the Pod running the Build terminated.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTimestamp *unversioned.Time

	// Duration contains time.Duration object describing build time.
	Duration time.Duration

	// OutputDockerImageReference contains a reference to the Docker image that
	// will be built by this build. It's value is computed from
	// Build.Spec.Output.To, and should include the registry address, so that
	// it can be used to push and pull the image.
	OutputDockerImageReference string

	// Config is an ObjectReference to the BuildConfig this Build is based on.
	Config *kapi.ObjectReference
}

// BuildPhase represents the status of a build at a point in time.
type BuildPhase string

// Valid values for BuildPhase.
const (
	// BuildPhaseNew is automatically assigned to a newly created build.
	BuildPhaseNew BuildPhase = "New"

	// BuildPhasePending indicates that a pod name has been assigned and a build is
	// about to start running.
	BuildPhasePending BuildPhase = "Pending"

	// BuildPhaseRunning indicates that a pod has been created and a build is running.
	BuildPhaseRunning BuildPhase = "Running"

	// BuildPhaseComplete indicates that a build has been successful.
	BuildPhaseComplete BuildPhase = "Complete"

	// BuildPhaseFailed indicates that a build has executed and failed.
	BuildPhaseFailed BuildPhase = "Failed"

	// BuildPhaseError indicates that an error prevented the build from executing.
	BuildPhaseError BuildPhase = "Error"

	// BuildPhaseCancelled indicates that a running/pending build was stopped from executing.
	BuildPhaseCancelled BuildPhase = "Cancelled"
)

// StatusReason is a brief CamelCase string that describes a temporary or
// permanent build error condition, meant for machine parsing and tidy display
// in the CLI.
type StatusReason string

// These are the valid reasons of build statuses.
const (
	// StatusReasonError is a generic reason for a build error condition.
	StatusReasonError StatusReason = "Error"

	// StatusReasonCannotCreateBuildPodSpec is an error condition when the build
	// strategy cannot create a build pod spec.
	StatusReasonCannotCreateBuildPodSpec = "CannotCreateBuildPodSpec"

	// StatusReasonCannotCreateBuildPod is an error condition when a build pod
	// cannot be created.
	StatusReasonCannotCreateBuildPod = "CannotCreateBuildPod"

	// StatusReasonInvalidOutputReference is an error condition when the build
	// output is an invalid reference.
	StatusReasonInvalidOutputReference = "InvalidOutputReference"

	// StatusReasonCancelBuildFailed is an error condition when cancelling a build
	// fails.
	StatusReasonCancelBuildFailed = "CancelBuildFailed"

	// StatusReasonBuildPodDeleted is an error condition when the build pod is
	// deleted before build completion.
	StatusReasonBuildPodDeleted = "BuildPodDeleted"

	// StatusReasonExceededRetryTimeout is an error condition when the build has
	// not completed and retrying the build times out.
	StatusReasonExceededRetryTimeout = "ExceededRetryTimeout"
)

// BuildSource is the input used for the build.
type BuildSource struct {
	// Binary builds accept a binary as their input. The binary is generally assumed to be a tar,
	// gzipped tar, or zip file depending on the strategy. For Docker builds, this is the build
	// context and an optional Dockerfile may be specified to override any Dockerfile in the
	// build context. For Source builds, this is assumed to be an archive as described above. For
	// Source and Docker builds, if binary.asFile is set the build will receive a directory with
	// a single file. contextDir may be used when an archive is provided. Custom builds will
	// receive this binary as input on STDIN.
	Binary *BinaryBuildSource

	// Dockerfile is the raw contents of a Dockerfile which should be built. When this option is
	// specified, the FROM may be modified based on your strategy base image and additional ENV
	// stanzas from your strategy environment will be added after the FROM, but before the rest
	// of your Dockerfile stanzas. The Dockerfile source type may be used with other options like
	// git - in those cases the Git repo will have any innate Dockerfile replaced in the context
	// dir.
	Dockerfile *string

	// Git contains optional information about git build source
	Git *GitBuildSource

	// Image describes an image to be used to provide source for the build
	// EXPERIMENTAL.  This will be changing to an array of images in the near future
	// and no migration/compatibility will be provided.  Use at your own risk.
	Image *ImageSource

	// ContextDir specifies the sub-directory where the source code for the application exists.
	// This allows to have buildable sources in directory other than root of
	// repository.
	ContextDir string

	// SourceSecret is the name of a Secret that would be used for setting
	// up the authentication for cloning private repository.
	// The secret contains valid credentials for remote repository, where the
	// data's key represent the authentication method to be used and value is
	// the base64 encoded credentials. Supported auth methods are: ssh-privatekey.
	// TODO: This needs to move under the GitBuildSource struct since it's only
	// used for git authentication
	SourceSecret *kapi.LocalObjectReference
}

// ImageSource describes an image that is used as source for the build
type ImageSource struct {
	// From is a reference to an ImageStreamTag, ImageStreamImage, or DockerImage to
	// copy source from.
	From kapi.ObjectReference

	// Paths is a list of source and destination paths to copy from the image.
	Paths []ImageSourcePath

	// PullSecret is a reference to a secret to be used to pull the image from a registry
	// If the image is pulled from the OpenShift registry, this field does not need to be set.
	PullSecret *kapi.LocalObjectReference
}

// ImageSourcePath describes a path to be copied from a source image and its destination within the build directory.
type ImageSourcePath struct {
	// SourcePath is the absolute path of the file or directory inside the image to
	// copy to the build directory.
	SourcePath string

	// DestinationDir is the relative directory within the build directory
	// where files copied from the image are placed.
	DestinationDir string
}

type BinaryBuildSource struct {
	// AsFile indicates that the provided binary input should be considered a single file
	// within the build input. For example, specifying "webapp.war" would place the provided
	// binary as `/webapp.war` for the builder. If left empty, the Docker and Source build
	// strategies assume this file is a zip, tar, or tar.gz file and extract it as the source.
	// The custom strategy receives this binary as standard input. This filename may not
	// contain slashes or be '..' or '.'.
	AsFile string
}

// SourceRevision is the revision or commit information from the source for the build
type SourceRevision struct {
	// Git contains information about git-based build source
	Git *GitSourceRevision
}

// GitSourceRevision is the commit information from a git source for a build
type GitSourceRevision struct {
	// Commit is the commit hash identifying a specific commit
	Commit string

	// Author is the author of a specific commit
	Author SourceControlUser

	// Committer is the committer of a specific commit
	Committer SourceControlUser

	// Message is the description of a specific commit
	Message string
}

// GitBuildSource defines the parameters of a Git SCM
type GitBuildSource struct {
	// URI points to the source that will be built. The structure of the source
	// will depend on the type of build to run
	URI string

	// Ref is the branch/tag/ref to build.
	Ref string

	// HTTPProxy is a proxy used to reach the git repository over http
	HTTPProxy string

	// HTTPSProxy is a proxy used to reach the git repository over https
	HTTPSProxy string
}

// SourceControlUser defines the identity of a user of source control
type SourceControlUser struct {
	// Name of the source control user
	Name string

	// Email of the source control user
	Email string
}

// BuildStrategy contains the details of how to perform a build.
type BuildStrategy struct {
	// DockerStrategy holds the parameters to the Docker build strategy.
	DockerStrategy *DockerBuildStrategy

	// SourceStrategy holds the parameters to the Source build strategy.
	SourceStrategy *SourceBuildStrategy

	// CustomStrategy holds the parameters to the Custom build strategy
	CustomStrategy *CustomBuildStrategy
}

// BuildStrategyType describes a particular way of performing a build.
type BuildStrategyType string

const (
	// CustomBuildStrategyBaseImageKey is the environment variable that indicates the base image to be used when
	// performing a custom build, if needed.
	CustomBuildStrategyBaseImageKey = "OPENSHIFT_CUSTOM_BUILD_BASE_IMAGE"
)

// CustomBuildStrategy defines input parameters specific to Custom build.
type CustomBuildStrategy struct {
	// From is reference to an DockerImage, ImageStream, ImageStreamTag, or ImageStreamImage from which
	// the docker image should be pulled
	From kapi.ObjectReference

	// PullSecret is the name of a Secret that would be used for setting up
	// the authentication for pulling the Docker images from the private Docker
	// registries
	PullSecret *kapi.LocalObjectReference

	// Env contains additional environment variables you want to pass into a builder container
	Env []kapi.EnvVar

	// ExposeDockerSocket will allow running Docker commands (and build Docker images) from
	// inside the Docker container.
	// TODO: Allow admins to enforce 'false' for this option
	ExposeDockerSocket bool

	// ForcePull describes if the controller should configure the build pod to always pull the images
	// for the builder or only pull if it is not present locally
	ForcePull bool

	// Secrets is a list of additional secrets that will be included in the custom build pod
	Secrets []SecretSpec
}

// DockerBuildStrategy defines input parameters specific to Docker build.
type DockerBuildStrategy struct {
	// From is reference to an DockerImage, ImageStream, ImageStreamTag, or ImageStreamImage from which
	// the docker image should be pulled
	// the resulting image will be used in the FROM line of the Dockerfile for this build.
	From *kapi.ObjectReference

	// PullSecret is the name of a Secret that would be used for setting up
	// the authentication for pulling the Docker images from the private Docker
	// registries
	PullSecret *kapi.LocalObjectReference

	// NoCache if set to true indicates that the docker build must be executed with the
	// --no-cache=true flag
	NoCache bool

	// Env contains additional environment variables you want to pass into a builder container
	Env []kapi.EnvVar

	// ForcePull describes if the builder should pull the images from registry prior to building.
	ForcePull bool

	// DockerfilePath is the path of the Dockerfile that will be used to build the Docker image,
	// relative to the root of the context (contextDir).
	DockerfilePath string
}

// SourceBuildStrategy defines input parameters specific to an Source build.
type SourceBuildStrategy struct {
	// From is reference to an DockerImage, ImageStream, ImageStreamTag, or ImageStreamImage from which
	// the docker image should be pulled
	From kapi.ObjectReference

	// PullSecret is the name of a Secret that would be used for setting up
	// the authentication for pulling the Docker images from the private Docker
	// registries
	PullSecret *kapi.LocalObjectReference

	// Env contains additional environment variables you want to pass into a builder container
	Env []kapi.EnvVar

	// Scripts is the location of Source scripts
	Scripts string

	// Incremental flag forces the Source build to do incremental builds if true.
	Incremental bool

	// ForcePull describes if the builder should pull the images from registry prior to building.
	ForcePull bool
}

// BuildOutput is input to a build strategy and describes the Docker image that the strategy
// should produce.
type BuildOutput struct {
	// To defines an optional location to push the output of this build to.
	// Kind must be one of 'ImageStreamTag' or 'DockerImage'.
	// This value will be used to look up a Docker image repository to push to.
	// In the case of an ImageStreamTag, the ImageStreamTag will be looked for in the namespace of
	// the build unless Namespace is specified.
	To *kapi.ObjectReference

	// PushSecret is the name of a Secret that would be used for setting
	// up the authentication for executing the Docker push to authentication
	// enabled Docker Registry (or Docker Hub).
	PushSecret *kapi.LocalObjectReference
}

const (
	// BuildConfigLabel is the key of a Build label whose value is the ID of a BuildConfig
	// on which the Build is based.
	BuildConfigLabel = "openshift.io/build-config.name"
	// BuildConfigLabelDeprecated was used as BuildConfigLabel before adding namespaces.
	// We keep it for backward compatibility.
	BuildConfigLabelDeprecated = "buildconfig"
)

// BuildConfig is a template which can be used to create new builds.
type BuildConfig struct {
	unversioned.TypeMeta
	kapi.ObjectMeta

	// Spec holds all the input necessary to produce a new build, and the conditions when
	// to trigger them.
	Spec BuildConfigSpec
	// Status holds any relevant information about a build config
	Status BuildConfigStatus
}

// BuildConfigSpec describes when and how builds are created
type BuildConfigSpec struct {
	// Triggers determine how new Builds can be launched from a BuildConfig. If no triggers
	// are defined, a new build can only occur as a result of an explicit client build creation.
	Triggers []BuildTriggerPolicy

	// BuildSpec is the desired build specification
	BuildSpec
}

// BuildConfigStatus contains current state of the build config object.
type BuildConfigStatus struct {
	// LastVersion is used to inform about number of last triggered build.
	LastVersion int
}

// WebHookTrigger is a trigger that gets invoked using a webhook type of post
type WebHookTrigger struct {
	// Secret used to validate requests.
	Secret string
}

// ImageChangeTrigger allows builds to be triggered when an ImageStream changes
type ImageChangeTrigger struct {
	// LastTriggeredImageID is used internally by the ImageChangeController to save last
	// used image ID for build
	LastTriggeredImageID string

	// From is a reference to an ImageStreamTag that will trigger a build when updated
	// It is optional. If no From is specified, the From image from the build strategy
	// will be used. Only one ImageChangeTrigger with an empty From reference is allowed in
	// a build configuration.
	From *kapi.ObjectReference
}

// BuildTriggerPolicy describes a policy for a single trigger that results in a new Build.
type BuildTriggerPolicy struct {
	// Type is the type of build trigger
	Type BuildTriggerType

	// GitHubWebHook contains the parameters for a GitHub webhook type of trigger
	GitHubWebHook *WebHookTrigger

	// GenericWebHook contains the parameters for a Generic webhook type of trigger
	GenericWebHook *WebHookTrigger

	// ImageChange contains parameters for an ImageChange type of trigger
	ImageChange *ImageChangeTrigger
}

// BuildTriggerType refers to a specific BuildTriggerPolicy implementation.
type BuildTriggerType string

//NOTE: Adding a new trigger type requires adding the type to KnownTriggerTypes
var KnownTriggerTypes = sets.NewString(
	string(GitHubWebHookBuildTriggerType),
	string(GenericWebHookBuildTriggerType),
	string(ImageChangeBuildTriggerType),
	string(ConfigChangeBuildTriggerType),
)

const (
	// GitHubWebHookBuildTriggerType represents a trigger that launches builds on
	// GitHub webhook invocations
	GitHubWebHookBuildTriggerType           BuildTriggerType = "GitHub"
	GitHubWebHookBuildTriggerTypeDeprecated BuildTriggerType = "github"

	// GenericWebHookBuildTriggerType represents a trigger that launches builds on
	// generic webhook invocations
	GenericWebHookBuildTriggerType           BuildTriggerType = "Generic"
	GenericWebHookBuildTriggerTypeDeprecated BuildTriggerType = "generic"

	// ImageChangeBuildTriggerType represents a trigger that launches builds on
	// availability of a new version of an image
	ImageChangeBuildTriggerType           BuildTriggerType = "ImageChange"
	ImageChangeBuildTriggerTypeDeprecated BuildTriggerType = "imageChange"

	// ConfigChangeBuildTriggerType will trigger a build on an initial build config creation
	// WARNING: In the future the behavior will change to trigger a build on any config change
	ConfigChangeBuildTriggerType BuildTriggerType = "ConfigChange"
)

// BuildList is a collection of Builds.
type BuildList struct {
	unversioned.TypeMeta
	unversioned.ListMeta

	// Items is a list of builds
	Items []Build
}

// BuildConfigList is a collection of BuildConfigs.
type BuildConfigList struct {
	unversioned.TypeMeta
	unversioned.ListMeta

	// Items is a list of build configs
	Items []BuildConfig
}

// GenericWebHookEvent is the payload expected for a generic webhook post
type GenericWebHookEvent struct {
	// Git is the git information, if any.
	Git *GitInfo
}

// GitInfo is the aggregated git information for a generic webhook post
type GitInfo struct {
	GitBuildSource
	GitSourceRevision

	// Refs is a list of GitRefs for the provided repo - generally sent
	// when used from a post-receive hook. This field is optional and is
	// used when sending multiple refs
	Refs []GitRefInfo
}

// GitRefInfo is a single ref
type GitRefInfo struct {
	GitBuildSource
	GitSourceRevision
}

// BuildLog is the (unused) resource associated with the build log redirector
type BuildLog struct {
	unversioned.TypeMeta
}

// BuildRequest is the resource used to pass parameters to build generator
type BuildRequest struct {
	unversioned.TypeMeta
	// TODO: build request should allow name generation via Name and GenerateName, build config
	// name should be provided as a separate field
	kapi.ObjectMeta

	// Revision is the information from the source for a specific repo snapshot.
	Revision *SourceRevision

	// TriggeredByImage is the Image that triggered this build.
	TriggeredByImage *kapi.ObjectReference

	// From is the reference to the ImageStreamTag that triggered the build.
	From *kapi.ObjectReference

	// Binary indicates a request to build from a binary provided to the builder
	Binary *BinaryBuildSource

	// LastVersion (optional) is the LastVersion of the BuildConfig that was used
	// to generate the build. If the BuildConfig in the generator doesn't match, a build will
	// not be generated.
	LastVersion *int

	// Env contains additional environment variables you want to pass into a builder container
	Env []kapi.EnvVar
}

type BinaryBuildRequestOptions struct {
	unversioned.TypeMeta
	kapi.ObjectMeta

	AsFile string

	// TODO: support structs in query arguments in the future (inline and nested fields)

	// Commit is the value identifying a specific commit
	Commit string

	// Message is the description of a specific commit
	Message string

	// AuthorName of the source control user
	AuthorName string

	// AuthorEmail of the source control user
	AuthorEmail string

	// CommitterName of the source control user
	CommitterName string

	// CommitterEmail of the source control user
	CommitterEmail string
}

// BuildLogOptions is the REST options for a build log
type BuildLogOptions struct {
	unversioned.TypeMeta

	// Container for which to return logs
	Container string
	// Follow if true indicates that the build log should be streamed until
	// the build terminates.
	Follow bool
	// If true, return previous terminated container logs
	Previous bool
	// A relative time in seconds before the current time from which to show logs. If this value
	// precedes the time a pod was started, only logs since the pod start will be returned.
	// If this value is in the future, no logs will be returned.
	// Only one of sinceSeconds or sinceTime may be specified.
	SinceSeconds *int64
	// An RFC3339 timestamp from which to show logs. If this value
	// preceeds the time a pod was started, only logs since the pod start will be returned.
	// If this value is in the future, no logs will be returned.
	// Only one of sinceSeconds or sinceTime may be specified.
	SinceTime *unversioned.Time
	// If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line
	// of log output.
	Timestamps bool
	// If set, the number of lines from the end of the logs to show. If not specified,
	// logs are shown from the creation of the container or sinceSeconds or sinceTime
	TailLines *int64
	// If set, the number of bytes to read from the server before terminating the
	// log output. This may not display a complete final line of logging, and may return
	// slightly more or slightly less than the specified limit.
	LimitBytes *int64

	// NoWait if true causes the call to return immediately even if the build
	// is not available yet. Otherwise the server will wait until the build has started.
	NoWait bool

	// Version of the build for which to view logs.
	Version *int64
}

// SecretSpec specifies a secret to be included in a build pod and its corresponding mount point
type SecretSpec struct {
	// SecretSource is a reference to the secret
	SecretSource kapi.LocalObjectReference

	// MountPath is the path at which to mount the secret
	MountPath string
}
