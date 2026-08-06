package aws

import (
	"github.com/signoz/foundry/internal/convention"
	"github.com/signoz/foundry/internal/domain"
)

// displayName is unprefixed. "Name" is what an AWS console shows.
const displayName = "Name"

// Tag renders a fact as an AWS tag key. Only AWS accepts the full prefix.
func Tag(key convention.TagKey) string {
	return domain.MetadataPrefix + key.String()
}

// Filter renders a selection as the tag match a data source is keyed by.
func Filter(selection convention.Selection) map[string]string {
	match := selection.Match()

	tags := make(map[string]string, len(match))
	for key, value := range match {
		tags[Tag(key)] = value
	}

	return tags
}
