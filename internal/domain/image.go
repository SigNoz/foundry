package domain

import (
	"github.com/distribution/reference"
	"github.com/signoz/foundry/internal/errors"
)

// Image is a tagged reference normalized the way docker does (docker.io,
// library/); digest references are refused since foundry pins by tag.
type Image struct {
	named reference.NamedTagged
}

func NewImage(repository, tag string) (Image, error) {
	if repository == "" {
		return Image{}, errors.Newf(errors.TypeInvalidInput, "failed to create image: repository is empty")
	}

	if tag == "" {
		return Image{}, errors.Newf(errors.TypeInvalidInput, "failed to create image: tag is empty")
	}

	named, err := reference.ParseNormalizedNamed(repository)
	if err != nil {
		return Image{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to create image from %q", repository)
	}

	tagged, err := reference.WithTag(named, tag)
	if err != nil {
		return Image{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to create image from %q: tag %q", repository, tag)
	}

	return Image{named: tagged}, nil
}

func MustNewImage(repository, tag string) Image {
	image, err := NewImage(repository, tag)
	if err != nil {
		panic(err)
	}

	return image
}

// ParseImage accepts "[registry/]repository[:tag]"; a missing tag is "latest".
func ParseImage(raw string) (Image, error) {
	if raw == "" {
		return Image{}, errors.Newf(errors.TypeInvalidInput, "failed to create image from %q: reference is empty", raw)
	}

	named, err := reference.ParseNormalizedNamed(raw)
	if err != nil {
		return Image{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to create image from %q", raw)
	}

	if _, ok := named.(reference.Canonical); ok {
		return Image{}, errors.Newf(errors.TypeUnsupported, "failed to create image from %q: digest references are not supported", raw)
	}

	tagged, ok := reference.TagNameOnly(named).(reference.NamedTagged)
	if !ok {
		return Image{}, errors.Newf(errors.TypeInternal, "failed to create image from %q: no tag resolved", raw)
	}

	return Image{named: tagged}, nil
}

// Registry is always named: docker.io for an unqualified reference.
func (i Image) Registry() string {
	return reference.Domain(i.named)
}

// Repository is the path within the registry: library/postgres for a bare name.
func (i Image) Repository() string {
	return reference.Path(i.named)
}

func (i Image) Tag() string {
	return i.named.Tag()
}

func (i Image) WithTag(tag string) Image {
	return MustNewImage(reference.FamiliarName(i.named), tag)
}

// Version parses the tag as a semantic version. ok is false for non-semver tags
// such as "latest".
func (i Image) Version() (Version, bool) {
	version, err := ParseVersion(i.named.Tag())
	if err != nil {
		return Version{}, false
	}

	return version, true
}

// String is the familiar form: docker.io and library/ elided.
func (i Image) String() string {
	return reference.FamiliarString(i.named)
}
