package convention

// TagKey names one fact a substrate records, unqualified. Spelling is the
// provider's grammar and is rendered where the resource is created: a GCP label
// key rejects the dot and the slash, an Azure tag name rejects the slash.
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
