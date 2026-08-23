package aws

import (
	"github.com/signoz/foundry/internal/contract"
	"github.com/signoz/foundry/internal/domain"
)

// displayName is the unprefixed tag an AWS console shows as a resource's name.
const displayName = "Name"

// Tag renders a fact as an AWS tag key, which accepts the full prefix.
func Tag(key contract.TagKey) string {
	return domain.MetadataPrefix + key.String()
}

// Filter renders a selection as the tag match a data source is keyed by.
func Filter(selection contract.Selection) map[string]string {
	match := selection.Match()

	tags := make(map[string]string, len(match))
	for key, value := range match {
		tags[Tag(key)] = value
	}

	return tags
}
