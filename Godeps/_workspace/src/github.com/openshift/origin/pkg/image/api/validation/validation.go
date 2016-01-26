package validation

import (
	"fmt"
	"regexp"

	"github.com/docker/distribution/reference"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/validation"
	"k8s.io/kubernetes/pkg/util/fielderrors"

	oapi "github.com/openshift/origin/pkg/api"
	"github.com/openshift/origin/pkg/image/api"
)

// RepositoryNameComponentRegexp restricts registry path component names to
// start with at least one letter or number, with following parts able to
// be separated by one period, dash or underscore.
// Copied from github.com/docker/distribution/registry/api/v2/names.go v2.1.1
var RepositoryNameComponentRegexp = regexp.MustCompile(`[a-z0-9]+(?:[._-][a-z0-9]+)*`)

// RepositoryNameComponentAnchoredRegexp is the version of
// RepositoryNameComponentRegexp which must completely match the content
// Copied from github.com/docker/distribution/registry/api/v2/names.go v2.1.1
var RepositoryNameComponentAnchoredRegexp = regexp.MustCompile(`^` + RepositoryNameComponentRegexp.String() + `$`)

// RepositoryNameRegexp builds on RepositoryNameComponentRegexp to allow
// multiple path components, separated by a forward slash.
// Copied from github.com/docker/distribution/registry/api/v2/names.go v2.1.1
var RepositoryNameRegexp = regexp.MustCompile(`(?:` + RepositoryNameComponentRegexp.String() + `/)*` + RepositoryNameComponentRegexp.String())

func ValidateImageStreamName(name string, prefix bool) (bool, string) {
	if ok, reason := oapi.MinimalNameRequirements(name, prefix); !ok {
		return ok, reason
	}

	if !RepositoryNameComponentAnchoredRegexp.MatchString(name) {
		return false, fmt.Sprintf("must match %q", RepositoryNameComponentRegexp.String())
	}
	return true, ""
}

// ValidateImage tests required fields for an Image.
func ValidateImage(image *api.Image) fielderrors.ValidationErrorList {
	result := fielderrors.ValidationErrorList{}

	result = append(result, validation.ValidateObjectMeta(&image.ObjectMeta, false, oapi.MinimalNameRequirements).Prefix("metadata")...)

	if len(image.DockerImageReference) == 0 {
		result = append(result, fielderrors.NewFieldRequired("dockerImageReference"))
	} else {
		if _, err := api.ParseDockerImageReference(image.DockerImageReference); err != nil {
			result = append(result, fielderrors.NewFieldInvalid("dockerImageReference", image.DockerImageReference, err.Error()))
		}
	}

	return result
}

func ValidateImageUpdate(newImage, oldImage *api.Image) fielderrors.ValidationErrorList {
	result := fielderrors.ValidationErrorList{}

	result = append(result, validation.ValidateObjectMetaUpdate(&newImage.ObjectMeta, &oldImage.ObjectMeta).Prefix("metadata")...)
	result = append(result, ValidateImage(newImage)...)

	return result
}

// ValidateImageStream tests required fields for an ImageStream.
func ValidateImageStream(stream *api.ImageStream) fielderrors.ValidationErrorList {
	result := fielderrors.ValidationErrorList{}
	result = append(result, validation.ValidateObjectMeta(&stream.ObjectMeta, true, ValidateImageStreamName).Prefix("metadata")...)

	// Ensure we can generate a valid docker image repository from namespace/name
	if len(stream.Namespace+"/"+stream.Name) > reference.NameTotalLengthMax {
		result = append(result, fielderrors.NewFieldInvalid("metadata.name", stream.Name, fmt.Sprintf("'namespace/name' cannot be longer than %d characters", reference.NameTotalLengthMax)))
	}

	if stream.Spec.Tags == nil {
		stream.Spec.Tags = make(map[string]api.TagReference)
	}

	if len(stream.Spec.DockerImageRepository) != 0 {
		if ref, err := api.ParseDockerImageReference(stream.Spec.DockerImageRepository); err != nil {
			result = append(result, fielderrors.NewFieldInvalid("spec.dockerImageRepository", stream.Spec.DockerImageRepository, err.Error()))
		} else {
			if len(ref.Tag) > 0 {
				result = append(result, fielderrors.NewFieldInvalid("spec.dockerImageRepository", stream.Spec.DockerImageRepository, "the repository name may not contain a tag"))
			}
			if len(ref.ID) > 0 {
				result = append(result, fielderrors.NewFieldInvalid("spec.dockerImageRepository", stream.Spec.DockerImageRepository, "the repository name may not contain an ID"))
			}
		}
	}
	for tag, tagRef := range stream.Spec.Tags {
		if tagRef.From != nil {
			switch tagRef.From.Kind {
			case "DockerImage", "ImageStreamImage", "ImageStreamTag":
			default:
				result = append(result, fielderrors.NewFieldInvalid(fmt.Sprintf("spec.tags[%s].from.kind", tag), tagRef.From.Kind, "valid values are 'DockerImage', 'ImageStreamImage', 'ImageStreamTag'"))
			}
		}
	}
	for tag, history := range stream.Status.Tags {
		for i, tagEvent := range history.Items {
			if len(tagEvent.DockerImageReference) == 0 {
				result = append(result, fielderrors.NewFieldRequired(fmt.Sprintf("status.tags[%s].items[%d].dockerImageReference", tag, i)))
			}
		}
	}

	return result
}

func ValidateImageStreamUpdate(newStream, oldStream *api.ImageStream) fielderrors.ValidationErrorList {
	result := fielderrors.ValidationErrorList{}

	result = append(result, validation.ValidateObjectMetaUpdate(&newStream.ObjectMeta, &oldStream.ObjectMeta).Prefix("metadata")...)
	result = append(result, ValidateImageStream(newStream)...)

	return result
}

// ValidateImageStreamStatusUpdate tests required fields for an ImageStream status update.
func ValidateImageStreamStatusUpdate(newStream, oldStream *api.ImageStream) fielderrors.ValidationErrorList {
	result := fielderrors.ValidationErrorList{}
	result = append(result, validation.ValidateObjectMetaUpdate(&newStream.ObjectMeta, &oldStream.ObjectMeta).Prefix("metadata")...)
	newStream.Spec.Tags = oldStream.Spec.Tags
	newStream.Spec.DockerImageRepository = oldStream.Spec.DockerImageRepository
	return result
}

// ValidateImageStreamMapping tests required fields for an ImageStreamMapping.
func ValidateImageStreamMapping(mapping *api.ImageStreamMapping) fielderrors.ValidationErrorList {
	result := fielderrors.ValidationErrorList{}
	result = append(result, validation.ValidateObjectMeta(&mapping.ObjectMeta, true, oapi.MinimalNameRequirements).Prefix("metadata")...)

	hasRepository := len(mapping.DockerImageRepository) != 0
	hasName := len(mapping.Name) != 0
	switch {
	case hasRepository:
		if _, err := api.ParseDockerImageReference(mapping.DockerImageRepository); err != nil {
			result = append(result, fielderrors.NewFieldInvalid("dockerImageRepository", mapping.DockerImageRepository, err.Error()))
		}
	case hasName:
	default:
		result = append(result, fielderrors.NewFieldRequired("name"))
		result = append(result, fielderrors.NewFieldRequired("dockerImageRepository"))
	}

	if ok, msg := validation.ValidateNamespaceName(mapping.Namespace, false); !ok {
		result = append(result, fielderrors.NewFieldInvalid("namespace", mapping.Namespace, msg))
	}
	if len(mapping.Tag) == 0 {
		result = append(result, fielderrors.NewFieldRequired("tag"))
	}
	if errs := ValidateImage(&mapping.Image).Prefix("image"); len(errs) != 0 {
		result = append(result, errs...)
	}
	return result
}

// ValidateImageStreamTag is essentially a no-op.  We don't allow direct creation of istags
func ValidateImageStreamTag(ist *api.ImageStreamTag) fielderrors.ValidationErrorList {
	result := fielderrors.ValidationErrorList{}
	result = append(result, validation.ValidateObjectMeta(&ist.ObjectMeta, true, oapi.MinimalNameRequirements).Prefix("metadata")...)

	return result
}

// ValidateImageStreamTagUpdate ensures that only the annotations of the IST have changed
func ValidateImageStreamTagUpdate(newIST, oldIST *api.ImageStreamTag) fielderrors.ValidationErrorList {
	result := fielderrors.ValidationErrorList{}

	result = append(result, validation.ValidateObjectMetaUpdate(&newIST.ObjectMeta, &oldIST.ObjectMeta).Prefix("metadata")...)

	// ensure that only annotations have changed
	newISTCopy := *newIST
	oldISTCopy := *oldIST
	newISTCopy.Annotations = nil
	oldISTCopy.Annotations = nil
	if !kapi.Semantic.Equalities.DeepEqual(&newISTCopy, &oldISTCopy) {
		result = append(result, fielderrors.NewFieldInvalid("metadata", "", "may not update fields other than metadata.annotations"))
	}

	return result
}
