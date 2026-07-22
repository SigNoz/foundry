package domain

import (
	"strings"

	"github.com/signoz/foundry/internal/errors"
)

// Image is a container image reference split into its repository and tag.
type Image struct {
	repository string
	tag        string
}

// NewImage requires a non-empty repository and tag.
func NewImage(repository, tag string) (Image, error) {
	if repository == "" {
		return Image{}, errors.Newf(errors.TypeInvalidInput, "failed to create image: repository is empty")
	}

	if tag == "" {
		return Image{}, errors.Newf(errors.TypeInvalidInput, "failed to create image: tag is empty")
	}

	return Image{repository: repository, tag: tag}, nil
}

// MustNewImage is NewImage for known-good literals; it panics on error.
func MustNewImage(repository, tag string) Image {
	image, err := NewImage(repository, tag)
	if err != nil {
		panic(err)
	}

	return image
}

// ParseImage accepts "repository[:tag]". The tag is the segment after the final
// colon, but only when that colon follows the final slash, so a registry port
// (e.g. "host:5000/repo") is not mistaken for a tag. A missing tag is "latest".
func ParseImage(raw string) (Image, error) {
	if raw == "" {
		return Image{}, errors.Newf(errors.TypeInvalidInput, "failed to create image from %q: reference is empty", raw)
	}

	repository, tag := raw, "latest"
	if i := strings.LastIndexByte(raw, ':'); i >= 0 && !strings.ContainsRune(raw[i+1:], '/') {
		repository, tag = raw[:i], raw[i+1:]
	}

	return NewImage(repository, tag)
}

func (i Image) Repository() string {
	return i.repository
}

func (i Image) Tag() string {
	return i.tag
}

// WithTag returns a copy of the image with its tag replaced.
func (i Image) WithTag(tag string) Image {
	return Image{repository: i.repository, tag: tag}
}

// Version parses the tag as a semantic version. ok is false for non-semver tags
// such as "latest".
func (i Image) Version() (Version, bool) {
	version, err := ParseVersion(i.tag)
	if err != nil {
		return Version{}, false
	}

	return version, true
}

// String renders the reference as "repository:tag".
func (i Image) String() string {
	return i.repository + ":" + i.tag
}
