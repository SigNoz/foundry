package contract

// TagKey is one fact a substrate records, without a provider's prefix or
// grammar. A GCP label key rejects the dot and the slash, an Azure tag name
// rejects the slash, so each provider renders the key its own way.
type TagKey struct {
	s string
}

var (
	TagKeyName       = TagKey{s: "name"}
	TagKeyStorage    = TagKey{s: "storage"}
	TagKeyIdentities = TagKey{s: "identities"}
	TagKeyOwner      = TagKey{s: "owner"}
	TagKeySubnetType = TagKey{s: "subnet-type"}
)

func (tagKey TagKey) String() string {
	return tagKey.s
}
