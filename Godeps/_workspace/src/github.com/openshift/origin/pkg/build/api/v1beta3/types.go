package v1beta3

import (
	"time"

	"k8s.io/kubernetes/pkg/api/unversioned"
	kapi "k8s.io/kubernetes/pkg/api/v1beta3"
)

// Build encapsulates the inputs needed to produce a new deployable image, as well as
// the status of the execution and a reference to the Pod which executed the build.
type Build struct {
	unversioned.TypeMeta `json:",inline"`
	kapi.ObjectMeta      `json:"metadata,omitempty"`

	// Spec is all the inputs used to execute the build.
	Spec BuildSpec `json:"spec,omitempty"`

	// Status is the current status of the build.
	Status BuildStatus `json:"status,omitempty"`
}

// BuildSpec encapsulates all the inputs necessary to represent a build.
type BuildSpec struct {
	// ServiceAccount is the name of the ServiceAccount to use to run the pod
	// created by this build.
	// The pod will be allowed to use secrets referenced by the ServiceAccount
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// Source describes the SCM in use.
	Source BuildSource `json:"source,omitempty"`

	// Revision is the information from the source for a specific repo snapshot.
	// This is optional.
	Revision *SourceRevision `json:"revision,omitempty"`

	// Strategy defines how to perform a build.
	Strategy BuildStrategy `json:"strategy"`

	// Output describes the Docker image the Strategy should produce.
	Output BuildOutput `json:"output,omitempty"`

	// Compute resource requirements to execute the build
	Resources kapi.ResourceRequirements `json:"resources,omitempty" description:"the desired compute resources the build should have"`

	// Optional duration in seconds, counted from the time when a build pod gets
	// scheduled in the system, that the build may be active on a node before the
	// system actively tries to terminate the build; value must be positive integer
	CompletionDeadlineSeconds *int64 `json:"completionDeadlineSeconds,omitempty" description:"optional duration in seconds the build may be active on a node before the system will actively try to mark it failed and kill associated containers; value must be a positive integer"`
}

// BuildStatus contains the status of a build
type BuildStatus struct {
	// Phase is the point in the build lifecycle.
	Phase BuildPhase `json:"phase"`

	// Cancelled describes if a cancel event was triggered for the build.
	Cancelled bool `json:"cancelled,omitempty"`

	// Reason is a brief CamelCase string that describes any failure and is meant for machine parsing and tidy display in the CLI.
	Reason StatusReason `json:"reason,omitempty" description:"brief CamelCase string describing a failure, meant for machine parsing and tidy display in the CLI"`

	// Message is a human-readable message indicating details about why the build has this status.
	Message string `json:"message,omitempty"`

	// StartTimestamp is a timestamp representing the server time when this Build started
	// running in a Pod.
	// It is represented in RFC3339 form and is in UTC.
	StartTimestamp *unversioned.Time `json:"startTimestamp,omitempty"`

	// CompletionTimestamp is a timestamp representing the server time when this Build was
	// finished, whether that build failed or succeeded.  It reflects the time at which
	// the Pod running the Build terminated.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTimestamp *unversioned.Time `json:"completionTimestamp,omitempty"`

	// Duration contains time.Duration object describing build time.
	Duration time.Duration `json:"duration,omitempty"`

	// OutputDockerImageReference contains a reference to the Docker image that
	// will be built by this build. It's value is computed from
	// Build.Spec.Output.To, and should include the registry address, so that
	// it can be used to push and pull the image.
	OutputDockerImageReference string `json:"outputDockerImageReference,omitempty" description:"reference to the Docker image built by this build, computed from build.spec.output.to, and can be used to push and pull the image"`

	// Config is an ObjectReference to the BuildConfig this Build is based on.
	Config *kapi.ObjectReference `json:"config,omitempty"`
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

// BuildSourceType is the type of SCM used.
type BuildSourceType string

// Valid values for BuildSourceType.
const (
	//BuildSourceGit instructs a build to use a Git source control repository as the build input.
	BuildSourceGit BuildSourceType = "Git"
	// BuildSourceDockerfile uses a Dockerfile as the start of a build
	BuildSourceDockerfile BuildSourceType = "Dockerfile"
	// BuildSourceBinary indicates the build will accept a Binary file as input.
	BuildSourceBinary BuildSourceType = "Binary"
	// BuildSourceImage indicates the build will accept an image as input
	BuildSourceImage BuildSourceType = "Image"
)

// BuildSource is the SCM used for the build.
type BuildSource struct {
	// Type of build input to accept
	Type BuildSourceType `json:"type" description:"type of build input to accept"`

	// Binary builds accept a binary as their input. The binary is generally assumed to be a tar,
	// gzipped tar, or zip file depending on the strategy. For Docker builds, this is the build
	// context and an optional Dockerfile may be specified to override any Dockerfile in the
	// build context. For Source builds, this is assumed to be an archive as described above. For
	// Source and Docker builds, if binary.asFile is set the build will receive a directory with
	// a single file. contextDir may be used when an archive is provided. Custom builds will
	// receive this binary as input on STDIN.
	Binary *BinaryBuildSource `json:"binary,omitempty" description:"the binary will be provided by the builder as an archive or file to be placed within the input directory; allows Dockerfile to be optionally set; may not be set with git source type also set"`

	// Dockerfile is the raw contents of a Dockerfile which should be built. When this option is
	// specified, the From and Env on the Docker build strategy are applied on top of this file.
	Dockerfile *string `json:"dockerfile,omitempty" description:"the contents of a Dockerfile to build; FROM and ENV may be overridden if you have specified 'from' and 'env' on the build strategy"`

	// Git contains optional information about git build source.
	Git *GitBuildSource `json:"git,omitempty"`

	// Image describes an image to be used to provide source for the build
	// EXPERIMENTAL.  This will be changing to an array of images in the near future
	// and no migration/compatibility will be provided.  Use at your own risk.
	Image *ImageSource `json:"image,omitempty" description:"optional image build source.  EXPERIMENTAL: This will be changing to an array of images in the near future and no migration/compatibility will be provided.  Use at your own risk."`

	// Specify the sub-directory where the source code for the application exists.
	// This allows to have buildable sources in directory other than root of
	// repository.
	ContextDir string `json:"contextDir,omitempty"`

	// SourceSecret is the name of a Secret that would be used for setting
	// up the authentication for cloning private repository.
	// The secret contains valid credentials for remote repository, where the
	// data's key represent the authentication method to be used and value is
	// the base64 encoded credentials. Supported auth methods are: ssh-privatekey.
	SourceSecret *kapi.LocalObjectReference `json:"sourceSecret,omitempty" description:"supported auth methods are: ssh-privatekey"`
}

// ImageSource describes an image that is used as source for the build
type ImageSource struct {
	// From is a reference to an ImageStreamTag, ImageStreamImage, or DockerImage to
	// copy source from.
	From kapi.ObjectReference `json:"from" description:"reference to ImageStreamTag, ImageStreamImage, or DockerImage"`

	// Paths is a list of source and destination paths to copy from the image.
	Paths []ImageSourcePath `json:"paths" description:"paths to copy from image"`

	// PullSecret is a reference to a secret to be used to pull the image from a registry
	// If the image is pulled from the OpenShift registry, this field does not need to be set.
	PullSecret *kapi.LocalObjectReference `json:"pullSecret,omitempty" description:"overrides the default pull secret for the source image"`
}

// ImageSourcePath describes a path to be copied from a source image and its destination within the build directory.
type ImageSourcePath struct {
	// SourcePath is the absolute path of the file or directory inside the image to
	// copy to the build directory.
	SourcePath string `json:"sourcePath" description:"source path (directory or file) inside image"`

	// DestinationDir is the relative directory within the build directory
	// where files copied from the image are placed.
	DestinationDir string `json:"destinationDir" description:"relative destination directory in build home"`
}

type BinaryBuildSource struct {
	// AsFile indicates that the provided binary input should be considered a single file
	// within the build input. For example, specifying "webapp.war" would place the provided
	// binary as `/webapp.war` for the builder. If left empty, the Docker and Source build
	// strategies assume this file is a zip, tar, or tar.gz file and extract it as the source.
	// The custom strategy receives this binary as standard input. This filename may not
	// contain slashes or be '..' or '.'.
	AsFile string `json:"asFile,omitempty" description:"indicate the provided binary should be considered a single file placed within the root of the input; must be a valid filename with no path segments"`
}

// SourceRevision is the revision or commit information from the source for the build
type SourceRevision struct {
	Type BuildSourceType    `json:"type"`
	Git  *GitSourceRevision `json:"git,omitempty"`
}

// GitSourceRevision is the commit information from a git source for a build
type GitSourceRevision struct {
	// Commit is the commit hash identifying a specific commit
	Commit string `json:"commit,omitempty"`

	// Author is the author of a specific commit
	Author SourceControlUser `json:"author,omitempty"`

	// Committer is the committer of a specific commit
	Committer SourceControlUser `json:"committer,omitempty"`

	// Message is the description of a specific commit
	Message string `json:"message,omitempty"`
}

// GitBuildSource defines the parameters of a Git SCM
type GitBuildSource struct {
	// URI points to the source that will be built. The structure of the source
	// will depend on the type of build to run
	URI string `json:"uri"`

	// Ref is the branch/tag/ref to build.
	Ref string `json:"ref,omitempty"`

	// HTTPProxy is a proxy used to reach the git repository over http
	HTTPProxy string `json:"httpProxy,omitempty" description:"specifies a http proxy to be used during git clone operations"`

	// HTTPSProxy is a proxy used to reach the git repository over https
	HTTPSProxy string `json:"httpsProxy,omitempty" description:"specifies a https proxy to be used during git clone operations"`
}

// SourceControlUser defines the identity of a user of source control
type SourceControlUser struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// BuildStrategy contains the details of how to perform a build.
type BuildStrategy struct {
	// Type is the kind of build strategy.
	Type BuildStrategyType `json:"type"`

	// DockerStrategy holds the parameters to the Docker build strategy.
	DockerStrategy *DockerBuildStrategy `json:"dockerStrategy,omitempty"`

	// SourceStrategy holds the parameters to the Source build strategy.
	SourceStrategy *SourceBuildStrategy `json:"sourceStrategy,omitempty"`

	// CustomStrategy holds the parameters to the Custom build strategy
	CustomStrategy *CustomBuildStrategy `json:"customStrategy,omitempty"`
}

// BuildStrategyType describes a particular way of performing a build.
type BuildStrategyType string

// Valid values for BuildStrategyType.
const (
	// DockerBuildStrategyType performs builds using a Dockerfile.
	DockerBuildStrategyType BuildStrategyType = "Docker"

	// SourceBuildStrategyType performs builds build using Source To Images with a Git repository
	// and a builder image.
	SourceBuildStrategyType BuildStrategyType = "Source"

	// CustomBuildStrategyType performs builds using custom builder Docker image.
	CustomBuildStrategyType BuildStrategyType = "Custom"
)

// CustomBuildStrategy defines input parameters specific to Custom build.
type CustomBuildStrategy struct {
	// From is reference to an ImageStreamTag, or ImageStreamImage from which
	// the docker image should be pulled
	From kapi.ObjectReference `json:"from"`

	// PullSecret is the name of a Secret that would be used for setting up
	// the authentication for pulling the Docker images from the private Docker
	// registries
	PullSecret *kapi.LocalObjectReference `json:"pullSecret,omitempty" description:"supported type: dockercfg"`

	// Additional environment variables you want to pass into a builder container
	Env []kapi.EnvVar `json:"env,omitempty"`

	// ExposeDockerSocket will allow running Docker commands (and build Docker images) from
	// inside the Docker container.
	// TODO: Allow admins to enforce 'false' for this option
	ExposeDockerSocket bool `json:"exposeDockerSocket,omitempty"`

	// ForcePull describes if the controller should configure the build pod to always pull the images
	// for the builder or only pull if it is not present locally
	ForcePull bool `json:"forcePull,omitempty" description:"forces pulling of builder image from remote registry if true"`

	// Secrets is a list of additional secrets that will be included in the build pod
	Secrets []SecretSpec `json:"secrets,omitempty" description:"a list of secrets to include in the build pod in addition to pull, push and source secrets"`
}

// DockerBuildStrategy defines input parameters specific to Docker build.
type DockerBuildStrategy struct {
	// From is reference to an ImageStreamTag, or ImageStreamImage from which
	// the docker image should be pulled
	// the resulting image will be used in the FROM line of the Dockerfile for this build.
	From *kapi.ObjectReference `json:"from,omitempty"`

	// PullSecret is the name of a Secret that would be used for setting up
	// the authentication for pulling the Docker images from the private Docker
	// registries
	PullSecret *kapi.LocalObjectReference `json:"pullSecret,omitempty" description:"supported type: dockercfg"`

	// NoCache if set to true indicates that the docker build must be executed with the
	// --no-cache=true flag
	NoCache bool `json:"noCache,omitempty"`

	// Env contains additional environment variables you want to pass into a builder container
	Env []kapi.EnvVar `json:"env,omitempty" description:"additional environment variables you want to pass into a builder container"`

	// ForcePull describes if the builder should pull the images from registry prior to building.
	ForcePull bool `json:"forcePull,omitempty" description:"forces the source build to pull the image if true"`

	// DockerfilePath is the path of the Dockerfile that will be used to build the Docker image,
	// relative to the root of the context (contextDir).
	DockerfilePath string `json:"dockerfilePath,omitempty" description:"path of the Dockerfile to use for building the Docker image, relative to the contextDir, if set"`
}

// SourceBuildStrategy defines input parameters specific to an Source build.
type SourceBuildStrategy struct {
	// From is reference to an ImageStreamTag, or ImageStreamImage from which
	// the docker image should be pulled
	From kapi.ObjectReference `json:"from"`

	// PullSecret is the name of a Secret that would be used for setting up
	// the authentication for pulling the Docker images from the private Docker
	// registries
	PullSecret *kapi.LocalObjectReference `json:"pullSecret,omitempty" description:"supported type: dockercfg"`

	// Additional environment variables you want to pass into a builder container
	Env []kapi.EnvVar `json:"env,omitempty"`

	// Scripts is the location of Source scripts
	Scripts string `json:"scripts,omitempty"`

	// Incremental flag forces the Source build to do incremental builds if true.
	Incremental bool `json:"incremental,omitempty"`

	// ForcePull describes if the builder should pull the images from registry prior to building.
	ForcePull bool `json:"forcePull,omitempty" description:"forces the source build to pull the image if true"`
}

// BuildOutput is input to a build strategy and describes the Docker image that the strategy
// should produce.
type BuildOutput struct {
	// To defines an optional ImageStream to push the output of this build to. The namespace
	// may be empty, in which case the ImageStream will be looked for in the namespace of
	// the build. Kind must be one of 'ImageStreamImage', 'ImageStreamTag' or 'DockerImage'.
	// This value will be used to look up a Docker image repository to push to.
	To *kapi.ObjectReference `json:"to,omitempty"`

	// PushSecret is the name of a Secret that would be used for setting
	// up the authentication for executing the Docker push to authentication
	// enabled Docker Registry (or Docker Hub).
	PushSecret *kapi.LocalObjectReference `json:"pushSecret,omitempty" description:"supported type: dockercfg"`
}

// BuildConfig is a template which can be used to create new builds.
type BuildConfig struct {
	unversioned.TypeMeta `json:",inline"`
	kapi.ObjectMeta      `json:"metadata,omitempty"`

	// Spec holds all the input necessary to produce a new build, and the conditions when
	// to trigger them.
	Spec BuildConfigSpec `json:"spec"`
	// Status holds any relevant information about a build config
	Status BuildConfigStatus `json:"status"`
}

// BuildConfigSpec describes when and how builds are created
type BuildConfigSpec struct {
	// Triggers determine how new Builds can be launched from a BuildConfig. If no triggers
	// are defined, a new build can only occur as a result of an explicit client build creation.
	Triggers []BuildTriggerPolicy `json:"triggers"`

	BuildSpec `json:",inline"`
}

// BuildConfigStatus contains current state of the build config object.
type BuildConfigStatus struct {
	// LastVersion is used to inform about number of last triggered build.
	LastVersion int `json:"lastVersion"`
}

// WebHookTrigger is a trigger that gets invoked using a webhook type of post
type WebHookTrigger struct {
	// Secret used to validate requests.
	Secret string `json:"secret,omitempty"`
}

// ImageChangeTrigger allows builds to be triggered when an ImageStream changes
type ImageChangeTrigger struct {
	// LastTriggeredImageID is used internally by the ImageChangeController to save last
	// used image ID for build
	LastTriggeredImageID string `json:"lastTriggeredImageID,omitempty"`

	// From is a reference to an ImageStreamTag that will trigger a build when updated
	// It is optional. If no From is specified, the From image from the build strategy
	// will be used. Only one ImageChangeTrigger with an empty From reference is allowed in
	// a build configuration.
	From *kapi.ObjectReference `json:"from,omitempty" description:"reference to an ImageStreamTag that will trigger the build"`
}

// BuildTriggerPolicy describes a policy for a single trigger that results in a new Build.
type BuildTriggerPolicy struct {
	// Type is the type of build trigger
	Type BuildTriggerType `json:"type"`

	// GitHubWebHook contains the parameters for a GitHub webhook type of trigger
	GitHubWebHook *WebHookTrigger `json:"github,omitempty"`

	// GenericWebHook contains the parameters for a Generic webhook type of trigger
	GenericWebHook *WebHookTrigger `json:"generic,omitempty"`

	// ImageChange contains parameters for an ImageChange type of trigger
	ImageChange *ImageChangeTrigger `json:"imageChange,omitempty"`
}

// BuildTriggerType refers to a specific BuildTriggerPolicy implementation.
type BuildTriggerType string

const (
	// GitHubWebHookBuildTriggerType represents a trigger that launches builds on
	// GitHub webhook invocations
	GitHubWebHookBuildTriggerType BuildTriggerType = "github"

	// GenericWebHookBuildTriggerType represents a trigger that launches builds on
	// generic webhook invocations
	GenericWebHookBuildTriggerType BuildTriggerType = "generic"

	// ImageChangeBuildTriggerType represents a trigger that launches builds on
	// availability of a new version of an image
	ImageChangeBuildTriggerType BuildTriggerType = "imageChange"

	// ConfigChangeBuildTriggerType will trigger a build on an initial build config creation
	// WARNING: In the future the behavior will change to trigger a build on any config change
	ConfigChangeBuildTriggerType BuildTriggerType = "ConfigChange"
)

// BuildList is a collection of Builds.
type BuildList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []Build `json:"items"`
}

// BuildConfigList is a collection of BuildConfigs.
type BuildConfigList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []BuildConfig `json:"items"`
}

// GenericWebHookEvent is the payload expected for a generic webhook post
type GenericWebHookEvent struct {
	// Type is the type of source repository
	Type BuildSourceType `json:"type,omitempty"`

	// Git is the git information if the Type is BuildSourceGit
	Git *GitInfo `json:"git,omitempty"`
}

// GitInfo is the aggregated git information for a generic webhook post
type GitInfo struct {
	GitBuildSource    `json:",inline"`
	GitSourceRevision `json:",inline"`
}

// BuildLog is the (unused) resource associated with the build log redirector
type BuildLog struct {
	unversioned.TypeMeta `json:",inline"`
}

// BuildRequest is the resource used to pass parameters to build generator
type BuildRequest struct {
	unversioned.TypeMeta `json:",inline"`
	kapi.ObjectMeta      `json:"metadata,omitempty"`

	// Revision is the information from the source for a specific repo snapshot.
	Revision *SourceRevision `json:"revision,omitempty"`

	// TriggeredByImage is the Image that triggered this build.
	TriggeredByImage *kapi.ObjectReference `json:"triggeredByImage,omitempty"`

	// From is the reference to the ImageStreamTag that triggered the build.
	From *kapi.ObjectReference `json:"from,omitempty" description:"ImageStreamTag that triggered this build"`

	// Binary indicates a request to build from a binary provided to the builder
	Binary *BinaryBuildSource `json:"binary,omitempty" description:"the binary will be provided by the builder as an archive or file to be placed within the input directory; allows Dockerfile to be optionally set; may not be set with git source type also set"`

	// LastVersion (optional) is the LastVersion of the BuildConfig that was used
	// to generate the build. If the BuildConfig in the generator doesn't match, a build will
	// not be generated.
	LastVersion *int `json:"lastVersion,omitempty" description:"LastVersion of the BuildConfig that triggered this build"`

	// Env contains additional environment variables you want to pass into a builder container
	Env []kapi.EnvVar `json:"env,omitempty" description:"additional environment variables you want to pass into a builder container"`
}

type BinaryBuildRequestOptions struct {
	unversioned.TypeMeta `json:",inline"`
	kapi.ObjectMeta      `json:"metadata,omitempty"`

	AsFile string `json:"asFile,omitempty"`

	// Commit is the value identifying a specific commit
	Commit string `json:"revision.commit,omitempty" description:"string identifying a specific commit"`

	// Message is the description of a specific commit
	Message string `json:"revision.message,omitempty" description:"description of a specific commit"`

	// AuthorName of the source control user
	AuthorName string `json:"revision.authorName,omitempty" description:"name of the user who authored the commit"`

	// AuthorEmail of the source control user
	AuthorEmail string `json:"revision.authorEmail,omitempty" description:"e-mail of the user who authored the commit"`

	// CommitterName of the source control user
	CommitterName string `json:"revision.committerName,omitempty" description:"name of the user who added the commit"`

	// CommitterEmail of the source control user
	CommitterEmail string `json:"revision.committerEmail,omitempty" description:"e-mail of the user who added the commit"`
}

// BuildLogOptions is the REST options for a build log
type BuildLogOptions struct {
	unversioned.TypeMeta

	// The container for which to stream logs. Defaults to only container if there is one container in the pod.
	Container string `json:"container,omitempty" description:"the container for which to stream logs; defaults to only container if there is one container in the pod"`
	// Follow if true indicates that the build log should be streamed until
	// the build terminates.
	Follow bool `json:"follow,omitempty" description:"if true indicates that the log should be streamed; defaults to false"`
	// Return previous terminated container logs. Defaults to false.
	Previous bool `json:"previous,omitempty" description:"return previous terminated container logs; defaults to false."`
	// A relative time in seconds before the current time from which to show logs. If this value
	// precedes the time a pod was started, only logs since the pod start will be returned.
	// If this value is in the future, no logs will be returned.
	// Only one of sinceSeconds or sinceTime may be specified.
	SinceSeconds *int64 `json:"sinceSeconds,omitempty" description:"relative time in seconds before the current time from which to show logs"`
	// An RFC3339 timestamp from which to show logs. If this value
	// preceeds the time a pod was started, only logs since the pod start will be returned.
	// If this value is in the future, no logs will be returned.
	// Only one of sinceSeconds or sinceTime may be specified.
	SinceTime *unversioned.Time `json:"sinceTime,omitempty" description:"relative time in seconds before the current time from which to show logs"`
	// If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line
	// of log output. Defaults to false.
	Timestamps bool `json:"timestamps,omitempty" description:"add an RFC3339 or RFC3339Nano timestamp at the beginning of every line of log output"`
	// If set, the number of lines from the end of the logs to show. If not specified,
	// logs are shown from the creation of the container or sinceSeconds or sinceTime
	TailLines *int64 `json:"tailLines,omitempty" description:"the number of lines from the end of the logs to show"`
	// If set, the number of bytes to read from the server before terminating the
	// log output. This may not display a complete final line of logging, and may return
	// slightly more or slightly less than the specified limit.
	LimitBytes *int64 `json:"limitBytes,omitempty" description:"the number of bytes to read from the server before terminating the log output"`

	// NoWait if true causes the call to return immediately even if the build
	// is not available yet. Otherwise the server will wait until the build has started.
	NoWait bool `json:"nowait,omitempty" description:"if true indicates that the server should not wait for a log to be available before returning; defaults to false"`

	// Version of the build for which to view logs.
	Version *int64 `json:"version,omitempty" description:"the version of the build for which to view logs"`
}

// SecretSpec specifies a secret to be included in a build pod and its corresponding mount point
type SecretSpec struct {
	// SecretSource is a reference to the secret
	SecretSource kapi.LocalObjectReference `json:"secretSource" description:"a reference to a secret"`

	// MountPath is the path at which to mount the secret
	MountPath string `json:"mountPath" description:"path within the container at which the secret should be mounted"`
}
