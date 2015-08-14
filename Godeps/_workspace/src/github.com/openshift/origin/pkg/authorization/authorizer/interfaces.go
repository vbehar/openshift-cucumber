package authorizer

import (
	"net/http"

	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/auth/user"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

type Authorizer interface {
	Authorize(ctx kapi.Context, a AuthorizationAttributes) (allowed bool, reason string, err error)
	GetAllowedSubjects(ctx kapi.Context, attributes AuthorizationAttributes) (util.StringSet, util.StringSet, error)
}

type AuthorizationAttributeBuilder interface {
	GetAttributes(request *http.Request) (AuthorizationAttributes, error)
}

type AuthorizationAttributes interface {
	GetVerb() string
	GetAPIVersion() string
	// GetResource returns the resource type.  If IsNonResourceURL() is true, then GetResource() is "".
	GetResource() string
	GetResourceName() string
	// GetRequestAttributes is of type interface{} because different verbs and different Authorizer/AuthorizationAttributeBuilder pairs may have different contract requirements.
	GetRequestAttributes() interface{}
	// IsNonResourceURL returns true if this is not an action performed against the resource API
	IsNonResourceURL() bool
	// GetURL returns the URL path being requested, including the leading '/'
	GetURL() string
}

// ForbiddenMessageMaker creates a forbidden message from a MessageContext
type ForbiddenMessageMaker interface {
	MakeMessage(ctx MessageContext) (string, error)
}

// MessageContext contains sufficient information to create a forbidden message.  It is bundled in this one object to make it easy and obvious how to build a golang template
type MessageContext struct {
	User       user.Info
	Namespace  string
	Attributes AuthorizationAttributes
}
